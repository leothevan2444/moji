#!/bin/sh
set -eu

repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

revision=$(git rev-parse --short=12 HEAD)
tags=$(git tag --points-at HEAD --list 'v*')

if [ -n "$tags" ]; then
    if [ "$(printf '%s\n' "$tags" | wc -l)" -ne 1 ]; then
        echo 'Version error: HEAD has multiple v* tags; keep one release tag per commit.' >&2
        exit 1
    fi
    # Docker tags cannot contain SemVer build metadata (+...). Numeric identifiers
    # must not have leading zeroes; prerelease identifiers follow SemVer rules.
    number='(0|[1-9][0-9]*)'
    identifier='(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
    if [ "${#tags}" -gt 129 ] || ! printf '%s\n' "$tags" | LC_ALL=C grep -Eq "^v$number\.$number\.$number(-$identifier(\.$identifier)*)?$"; then
        echo "Version error: invalid release tag '$tags'; use vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-prerelease (no +metadata)." >&2
        exit 1
    fi
fi

dirty=$(git status --porcelain --untracked-files=normal)
if [ -n "$tags" ] && [ -z "$dirty" ]; then
    printf '%s\n' "${tags#v}"
else
    suffix=''
    if [ -n "$dirty" ]; then suffix='-dirty'; fi
    printf 'dev-%s%s\n' "$revision" "$suffix"
fi
