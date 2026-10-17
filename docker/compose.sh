#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
# Always replace inherited values: VERSION is the only release input.
MOJI_VERSION=$(cat "$script_dir/../VERSION")
export MOJI_VERSION
exec docker compose -f "$script_dir/docker-compose.yml" "$@"
