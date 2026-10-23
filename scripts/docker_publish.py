"""Publish the exact native images tested by this GitHub Actions run."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess


ARCHITECTURES = ("amd64", "arm64")
VERSION_LABEL = "org.opencontainers.image.version"
REVISION_LABEL = "org.opencontainers.image.revision"
DIGEST = re.compile(r"sha256:[0-9a-f]{64}")


def docker(*args, capture=False):
    result = subprocess.run(
        ["docker", *map(str, args)], check=True, text=True,
        stdout=subprocess.PIPE if capture else None,
    )
    return result.stdout if capture else None


def validate_image(info, arch, version, revision):
    if info.get("Os") != "linux" or info.get("Architecture") != arch:
        raise ValueError(f"Expected linux/{arch}, got {info.get('Os')}/{info.get('Architecture')}")
    labels = info.get("Config", {}).get("Labels") or {}
    if labels.get(VERSION_LABEL) != version or labels.get(REVISION_LABEL) != revision:
        raise ValueError(f"linux/{arch} image version or source revision does not match this CI run")
    image_id = info.get("Id", "")
    if not DIGEST.fullmatch(image_id):
        raise ValueError(f"Invalid linux/{arch} image ID: {image_id}")
    return image_id


def publish(artifacts, image, version, revision, run_key, metadata):
    if not re.fullmatch(r"[0-9]+-[0-9]+", run_key):
        raise ValueError("Run key must contain the GitHub run ID and attempt")
    tags = metadata.get("tags", [])
    if not tags or not all(isinstance(tag, str) and tag.startswith(image + ":") for tag in tags):
        raise ValueError("Release tags must belong to the configured image repository")
    labels = dict(metadata.get("labels", {}))
    for key, expected in ((VERSION_LABEL, version), (REVISION_LABEL, revision)):
        if labels.get(key, expected) != expected:
            raise ValueError(f"Release metadata does not match {key}")

    # Check that both complete artifacts exist before loading or pushing anything.
    images = {}
    for arch in ARCHITECTURES:
        directory = Path(artifacts) / f"moji-image-{arch}"
        archive = directory / "image.tar"
        if not archive.is_file():
            raise ValueError(f"Missing tested linux/{arch} image archive: {archive}")
        info = json.loads((directory / "image.json").read_text())[0]
        images[arch] = (archive, validate_image(info, arch, version, revision))

    # Use content IDs after checking the loaded image against the post-test record.
    for arch, (archive, tested_id) in images.items():
        docker("image", "load", "--input", archive)
        local_tag = f"moji-ci-{arch}:{version}"
        loaded = json.loads(docker("image", "inspect", local_tag, capture=True))[0]
        if validate_image(loaded, arch, version, revision) != tested_id:
            raise ValueError(f"Loaded linux/{arch} image differs from the tested image")
        print(f"Verified linux/{arch}: {tested_id}", flush=True)

    # Unique staging tags prevent concurrent CI runs from mixing architectures.
    sources = []
    for arch, (_, tested_id) in images.items():
        staging = f"{image}:ci-{run_key}-{arch}"
        docker("image", "tag", tested_id, staging)
        docker("image", "push", staging)
        pushed = json.loads(docker("image", "inspect", staging, capture=True))[0]
        digests = set(pushed.get("RepoDigests") or [])
        matching = [ref for ref in digests if ref.startswith(image + "@") and DIGEST.fullmatch(ref.split("@", 1)[1])]
        if len(matching) != 1:
            raise ValueError(f"Expected one pushed linux/{arch} digest for {image}, got {matching}")
        sources.append(matching[0])

    command = ["buildx", "imagetools", "create"]
    for tag in dict.fromkeys(tags):
        command.extend(("--tag", tag))
    docker(*command, *sources)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifacts", required=True, type=Path)
    parser.add_argument("--image", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--revision", required=True)
    parser.add_argument("--run-key", required=True)
    args = parser.parse_args()
    publish(**vars(args), metadata=json.loads(os.environ["DOCKER_METADATA_JSON"]))


if __name__ == "__main__":
    main()
