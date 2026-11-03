#!/usr/bin/env python3
"""Compare checked-in encoding oracles with an isolated official .NET runtime.

The pinned image must already be installed. No image pull or network is allowed
by this command; the container mounts only a disposable directory.
"""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


DIRECTORY = Path(__file__).resolve().parent
ROOT = DIRECTORY.parents[1]
IMAGE = "mcr.microsoft.com/dotnet/sdk@sha256:6125486580cdf529a2a337d51c5557af08eaa04ef08b04f5b1d5a94f6bec7f1a"
FIXTURE = ROOT / "internal/cardigann/testdata/dotnet-encoding-v9.json"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="replace fixture with verified runtime output")
    args = parser.parse_args()
    lock = json.loads((DIRECTORY / "sources.lock.json").read_text())
    with tempfile.TemporaryDirectory(prefix="moji-dotnet-reference-") as directory:
        work = Path(directory)
        for source in (DIRECTORY / "reference").iterdir():
            shutil.copyfile(source, work / source.name)
        (work / "empty-feed").mkdir()
        (work / "code-pages.json").write_text(json.dumps(lock["codePages"]))
        command = [
            "docker", "run", "--rm", "--pull=never", "--network", "none", "--read-only",
            "--tmpfs", "/tmp:rw,size=256m", "--memory", "1g", "--cpus", "2",
            "--user", f"{os.getuid()}:{os.getgid()}",
            "-e", "DOTNET_CLI_HOME=/tmp/dotnet-home", "-e", "DOTNET_CLI_TELEMETRY_OPTOUT=1",
            "-e", "DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1", "-e", "DOTNET_EnableDiagnostics=0",
            "-e", "DOTNET_NOLOGO=1", "-v", f"{work}:/work", "-w", "/work", IMAGE,
            "sh", "-c", "set -e\ndotnet restore --source /work/empty-feed\n"
            "dotnet run --no-restore --project EncodingReference.csproj -- code-pages.json reference.json",
        ]
        subprocess.run(command, check=True, timeout=120)
        actual = json.loads((work / "reference.json").read_text())
    if actual["runtime"] != "9.0.0" or set(actual["codePages"]) != {str(p) for p in lock["codePages"]}:
        raise ValueError("unexpected reference runtime or code-page coverage")
    actual["sdkImage"] = IMAGE
    if args.write:
        FIXTURE.write_text(json.dumps(actual, indent=2, ensure_ascii=True) + "\n")
    elif json.loads(FIXTURE.read_text()) != actual:
        raise ValueError("encoding oracle differs from official runtime")
    print(f".NET {actual['runtime']}: {len(actual['codePages'])} code-page oracles match")


if __name__ == "__main__":
    main()
