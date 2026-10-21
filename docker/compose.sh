#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
# Always replace inherited values: Git tags are the only release input.
MOJI_VERSION=$(sh "$script_dir/../scripts/version.sh")
MOJI_VCS_REF=$(git -C "$script_dir/.." rev-parse HEAD)
export MOJI_VERSION
export MOJI_VCS_REF
exec docker compose -f "$script_dir/docker-compose.yml" "$@"
