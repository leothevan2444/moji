#!/usr/bin/env python3
"""Build pinned Jackett in an isolated temporary source tree for comparisons.

An existing official archive can be supplied. Network access for downloading
source requires --download; package restore is an explicit build step.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import subprocess
import tarfile
import urllib.request
import uuid


DIRECTORY = Path(__file__).resolve().parent
LOCK = json.loads((DIRECTORY / "upstream.lock.json").read_text())
MAX_ARCHIVE = 32 << 20
PACKAGE_LOCK_TARGETS = (
    "moji-reference/packages.lock.json",
    "src/DateTimeRoutines/packages.lock.json",
    "src/Jackett.Common/packages.lock.json",
)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_package_locks(root=None):
    pins = LOCK.get("nugetLocks", {})
    if set(pins) != set(PACKAGE_LOCK_TARGETS):
        raise ValueError("incomplete or unknown NuGet lock targets")
    for relative, expected in pins.items():
        source = DIRECTORY / "nuget-locks" / relative
        if not source.is_file() or digest(source) != expected:
            raise ValueError(f"NuGet source lock checksum mismatch: {relative}")
        if root is not None:
            installed = root / relative
            if not installed.is_file() or digest(installed) != expected:
                raise ValueError(f"NuGet installed lock checksum mismatch: {relative}")


def install_package_locks(root):
    verify_package_locks()
    for relative in PACKAGE_LOCK_TARGETS:
        shutil.copyfile(DIRECTORY / "nuget-locks" / relative, root / relative)


def extract(archive, root):
    total = 0
    with tarfile.open(archive) as source:
        for count, entry in enumerate(source):
            parts = PurePosixPath(entry.name).parts
            if count > 20000 or entry.name.startswith("/") or ".." in parts or "\\" in entry.name:
                raise ValueError("unsafe source archive")
            if entry.issym() or entry.islnk() or not (entry.isfile() or entry.isdir()):
                raise ValueError("unsupported source member")
            total += entry.size
            if total > 128 << 20:
                raise ValueError("source archive exceeds unpacked budget")
            if len(parts) < 2:
                continue
            target = root.joinpath(*parts[1:])
            if entry.isdir():
                target.mkdir(parents=True, exist_ok=True)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                with target.open("xb") as output:
                    output.write(source.extractfile(entry).read())


def container_arguments(root, name, network):
    return ["docker", "run", "--rm", "--pull=never", "--name", name,
            "--network", network, "--read-only", "--tmpfs", "/tmp:rw,size=512m",
            "--memory", "2g", "--cpus", "2", "--user", f"{os.getuid()}:{os.getgid()}",
            "-e", "DOTNET_CLI_HOME=/tmp/dotnet-home", "-e", "DOTNET_CLI_TELEMETRY_OPTOUT=1",
            "-e", "DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1", "-e", "DOTNET_NOLOGO=1",
            "-e", "DOTNET_EnableDiagnostics=0", "-e", "TZ=UTC",
            "-v", f"{root}:/work", "-w", "/work", LOCK["sdkImage"]]


def run_container(root, command, network, timeout):
    name = "moji-indexer-reference-" + uuid.uuid4().hex
    try:
        subprocess.run(container_arguments(root, name, network) + command, check=True, timeout=timeout)
    finally:
        subprocess.run(["docker", "rm", "-f", name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)


def write_provenance(root):
    verify_package_locks(root)
    directory = root / "moji-reference/bin/Release/net9.0"
    binaries = {p.name: digest(p) for p in sorted(directory.glob("*.dll"))}
    source_files = {p.name: digest(p) for p in (DIRECTORY / "reference").iterdir()}
    provenance = {**LOCK, "binaries": binaries, "runnerSources": source_files}
    (root / "moji-reference-provenance.json").write_text(json.dumps(provenance, indent=2) + "\n")
    return provenance


def verify_provenance(root):
    data = json.loads((root / "moji-reference-provenance.json").read_text())
    for key, value in LOCK.items():
        if data.get(key) != value:
            raise ValueError("reference provenance differs from pin")
    verify_package_locks(root)
    for path in (DIRECTORY / "reference").iterdir():
        if data["runnerSources"].get(path.name) != digest(path):
            raise ValueError("reference runner changed; rebuild required")
    if "Jackett.Common.dll" not in data["binaries"] or "IndexerReference.dll" not in data["binaries"]:
        raise ValueError("incomplete reference binaries")
    for name, expected in data["binaries"].items():
        if digest(root / "moji-reference/bin/Release/net9.0" / name) != expected:
            raise ValueError(f"reference binary checksum mismatch: {name}")
    return data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--work-dir", required=True, type=Path)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--archive", type=Path)
    source.add_argument("--download", action="store_true")
    args = parser.parse_args()
    root = args.work_dir.resolve()
    root.mkdir(parents=True, exist_ok=True)
    if any(root.iterdir()):
        raise ValueError("use an empty work directory")
    archive = args.archive
    if args.download:
        archive = root / "upstream.tar.gz"
        url = f"https://codeload.github.com/{LOCK['repository']}/tar.gz/{LOCK['commit']}"
        with urllib.request.urlopen(url, timeout=90) as response:
            content = response.read(MAX_ARCHIVE + 1)
        if len(content) > MAX_ARCHIVE:
            raise ValueError("source archive exceeds byte budget")
        archive.write_bytes(content)
    if archive.stat().st_size > MAX_ARCHIVE or digest(archive) != LOCK["archiveSha256"]:
        raise ValueError("pinned source archive checksum mismatch")
    extract(archive, root)
    project = root / "moji-reference"
    project.mkdir()
    for source in (DIRECTORY / "reference").iterdir():
        shutil.copyfile(source, project / source.name)
    install_package_locks(root)
    command = ["dotnet", "restore", "moji-reference/IndexerReference.csproj",
               "-p:JackettSourceRoot=/work", "-p:TargetFrameworks=net9.0",
               "--packages", "/work/.nuget", "-p:NuGetAudit=false", "--locked-mode"]
    run_container(root, command, "bridge", 600)
    verify_package_locks(root)
    command = ["dotnet", "build", "moji-reference/IndexerReference.csproj",
               "-p:JackettSourceRoot=/work", "-p:TargetFrameworks=net9.0",
               "--no-restore", "-c", "Release"]
    run_container(root, command, "none", 600)
    provenance = write_provenance(root)
    print(f"Reference built: {provenance['commit']}, {len(provenance['binaries'])} pinned binary digests")


if __name__ == "__main__":
    main()
