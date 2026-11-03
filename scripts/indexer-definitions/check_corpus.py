#!/usr/bin/env python3
"""Verify a pinned public corpus without checking production definitions into Git.

Network access requires --download. An existing archive can be supplied for
offline runs. Only declarative v11 files are extracted; downloaded code is never
executed. Temporary rules disappear after the checker exits.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import tarfile
import tempfile
import urllib.request


ROOT = Path(__file__).resolve().parents[2]
LOCK = Path(__file__).with_name("corpus.lock.json")
MAX_ARCHIVE = 32 << 20
MAX_UNPACKED = 128 << 20
MAX_FILE = 1 << 20


def extract(archive, destination):
    total = 0
    seen = set()
    with tarfile.open(archive, "r:gz") as source:
        for count, entry in enumerate(source):
            if count >= 10000:
                raise ValueError("archive file-count budget exceeded")
            parts = PurePosixPath(entry.name).parts
            if entry.name.startswith("/") or ".." in parts or "\\" in entry.name:
                raise ValueError("unsafe archive path")
            if entry.issym() or entry.islnk() or not (entry.isfile() or entry.isdir()):
                raise ValueError("unsafe archive entry type")
            total += entry.size
            if total > MAX_UNPACKED:
                raise ValueError("unpacked archive budget exceeded")
            if len(parts) != 4 or parts[1:3] != ("definitions", "v11"):
                continue
            name = parts[-1]
            if not (name.endswith(".yml") or name == "schema.json"):
                continue
            if not entry.isfile() or entry.size > MAX_FILE or name in seen:
                raise ValueError("invalid or duplicate definition entry")
            seen.add(name)
            content = source.extractfile(entry)
            if content is None:
                raise ValueError("missing archive file content")
            data = content.read(MAX_FILE + 1)
            if len(data) != entry.size:
                raise ValueError("truncated or oversized definition")
            target = destination / name
            with target.open("xb") as handle:
                handle.write(data)
            target.chmod(0o600)
    if not any(name.endswith(".yml") for name in seen):
        raise ValueError("archive does not contain v11 definitions")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", type=Path)
    parser.add_argument("--download", action="store_true")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    if args.archive is None and not args.download:
        parser.error("use --archive for an offline run or explicitly pass --download")
    lock = json.loads(LOCK.read_text())
    commit = lock["commit"]
    if len(commit) != 40 or any(c not in "0123456789abcdef" for c in commit):
        raise ValueError("invalid pinned commit")
    with tempfile.TemporaryDirectory(prefix="moji-indexer-corpus-") as temporary:
        work = Path(temporary)
        archive = args.archive
        if archive is None:
            archive = work / "archive.tar.gz"
            request = urllib.request.Request(
                f"https://codeload.github.com/Prowlarr/Indexers/tar.gz/{commit}",
                headers={"User-Agent": "Moji-Cardigann-Compatibility-Check"},
            )
            with urllib.request.urlopen(request, timeout=90) as response:
                data = response.read(MAX_ARCHIVE + 1)
            if len(data) > MAX_ARCHIVE:
                raise ValueError("compressed archive budget exceeded")
            archive.write_bytes(data)
        if archive.stat().st_size > MAX_ARCHIVE:
            raise ValueError("compressed archive budget exceeded")
        if hashlib.sha256(archive.read_bytes()).hexdigest() != lock["archiveSha256"]:
            raise ValueError("pinned archive checksum mismatch")
        definitions = work / "definitions"
        definitions.mkdir(mode=0o700)
        extract(archive, definitions)
        report = args.report.resolve() if args.report else work / "report.json"
        environment = dict(os.environ)
        environment.setdefault("GOCACHE", "/tmp/moji-go-cache")
        result = subprocess.run(
            ["go", "run", "./cmd/moji-indexer-check", "-definitions", str(definitions), "-report", str(report)],
            cwd=ROOT, env=environment, check=False,
        )
        if not report.exists():
            raise RuntimeError("checker did not produce a report")
        summary = json.loads(report.read_text())
        count = summary["definitions"]
        if count != lock["definitions"]:
            raise ValueError(f"corpus count mismatch: {count}")
        print(f"Cardigann v11: {summary['loaded']}/{count} statically compiled; "
              f"{len(summary['warnings'])} definitions with warnings; "
              f"{len(summary['failures'])} failures.")
        if summary["failures"]:
            for name, failure in summary["failures"].items():
                print(f"{name}: {failure}")
        return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
