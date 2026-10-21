#!/bin/sh
set -eu

repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
# This version must match @playwright/test and the CI container.
exec docker run --rm --ipc=host \
    --user "$(id -u):$(id -g)" \
    --env HOME=/tmp --env CI=1 \
    --mount "type=bind,source=$repo_dir,target=/work" \
    --workdir /work/web \
    mcr.microsoft.com/playwright:v1.61.1-noble \
    npm run test:visual -- "$@"
