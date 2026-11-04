#!/bin/sh
set -eu

version=$(sh "$(dirname "$0")/../scripts/version.sh")
image=${1:-moji:$version}
name="moji-smoke-$$"
volume="$name-data"

cleanup() {
    docker rm -f "$name" >/dev/null 2>&1 || true
    docker volume rm "$volume" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

# Exercise the exact FFmpeg binaries and shared libraries shipped in the image.
test "$(docker run --rm --network none "$image" --version)" = "moji $version"
test "$(docker image inspect "$image" --format '{{index .Config.Labels "org.opencontainers.image.version"}}')" = "$version"
# Preserve attribution and complete license texts for the compiled-in encoding
# mappings and Cardigann dependencies, without needing build-time module sources.
cardigann_dir="$(dirname "$0")/../internal/cardigann"
for notice in "$cardigann_dir/NOTICE.md" "$cardigann_dir/DOTNET-LICENSE.txt" "$cardigann_dir"/licenses/*; do
    relative_notice=${notice#"$cardigann_dir/"}
    expected_notice=$(sha256sum "$notice")
    actual_notice=$(docker run --rm --network none --entrypoint sha256sum "$image" "/usr/share/licenses/moji/cardigann/$relative_notice")
    test "${actual_notice%% *}" = "${expected_notice%% *}"
done
docker run --rm --network none --entrypoint sh "$image" -ec '
    test "$(id -u)" != 0
    grep -qx "config_version: 3" /opt/moji/config.example.yaml
    ffprobe -version >/dev/null
    work=$(mktemp -d)
    trap '\''rm -rf "$work"'\'' EXIT
    cd "$work"
    printf '\''1\n00:00:00,100 --> 00:00:00,800\nsmoke subtitle\n'\'' > subtitle.srt
    for color in red blue; do
        ffmpeg -v error -f lavfi -i "color=c=$color:s=64x64:r=10:d=1" \
            -f lavfi -i sine=frequency=440:duration=1 -i subtitle.srt \
            -map 0:v -map 1:a -map 2:s -c:v libx264 -preset ultrafast \
            -threads 1 -c:a aac -c:s mov_text "$color.mp4"
        ffmpeg -v error -i "$color.mp4" -map 0 -c:v copy -c:a copy \
            -c:s srt "$color.mkv"
    done
    printf "file red.mkv\nfile blue.mkv\n" > parts.txt
    ffmpeg -v error -f concat -safe 0 -i parts.txt -c copy merged.mkv
    ffprobe -v error -show_entries format=duration -of csv=p=0 merged.mkv \
        | awk '\''{ if ($1 < 1.8 || $1 > 2.4) exit 1; seen=1 } END { if (!seen) exit 1 }'\''
    ffmpeg -v error -i merged.mkv -map 0:s:0 merged.srt
    grep -q '\''smoke subtitle'\'' merged.srt
'

docker volume create "$volume" >/dev/null
docker run --rm -i --network none --entrypoint sh \
    --mount "type=volume,src=$volume,dst=/data" "$image" \
    -c 'cat > /data/config.yaml' <<'YAML'
config_version: 3
tasks:
  task_processing_enabled: false
  task_progress_sync_interval_seconds: -1
  subscription_poll_interval_hours: -1
YAML

# A fresh installation must serve its UI without Jackett or a rule download.
# No network also prevents this test from contacting any real integration.
docker run -d --name "$name" --network none --read-only --tmpfs /tmp:mode=1777 \
    --health-interval=1s --health-start-period=1s \
    --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null

wait_for_server() {
    attempt=0
    until [ "$(docker inspect "$name" --format '{{.State.Health.Status}}')" = healthy ]; do
        attempt=$((attempt + 1))
        if [ "$attempt" -ge 30 ]; then
            docker logs "$name"
            exit 1
        fi
        sleep 1
    done
}
wait_for_server

assert_unconfigured_indexers() {
    response=$(docker exec "$name" curl --fail --silent --show-error http://127.0.0.1:10000/graphql \
        -H 'Content-Type: application/json' \
        --data '{"query":"{ homeIndexerSummary { total builtinEnabled jackett { sites } } indexerInstances { jackettEnabled items { id } } indexerDefinitionStatus { activeRevision available installed { revision } lastCheckedAt job { id } errorCode } }"}')
    expected='{"data":{"homeIndexerSummary":{"total":0,"builtinEnabled":0,"jackett":null},"indexerInstances":{"jackettEnabled":false,"items":[]},"indexerDefinitionStatus":{"activeRevision":"","available":false,"installed":[],"lastCheckedAt":null,"job":null,"errorCode":"DEFINITION_LIBRARY_NOT_INSTALLED"}}}'
    if [ "$response" != "$expected" ]; then
        printf 'Unexpected unconfigured indexer state: %s\n' "$response" >&2
        return 1
    fi
}
assert_unconfigured_indexers

docker exec "$name" curl --fail --silent http://127.0.0.1:10000/ \
    | grep -qi '<!doctype html>'
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' --data '{"query":"{ version }"}' \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/version.json \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" sh -ec '
    database_root=/data/.storage/database
    generation_name=$(cat "$database_root/CURRENT")
    test -s "$database_root/$generation_name/moji.db"
    test -s "$database_root/$generation_name/runtime.db"
    test -s "$database_root/$generation_name/cache/stashbox/cache.db"
    printf "%s" "$generation_name" > /data/smoke-generation
    printf persisted > /data/smoke-marker
'
docker exec "$name" curl --fail --silent --show-error http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' \
    --data '{"query":"mutation { updateSystemSettings(input: {taskDeletePolicy: KEEP_ONLY, stashBoxDataCache: {ttlHours: 72}}) { system { stashBoxDataCache { ttlHours } } } }"}' \
    | grep -Fq '"ttlHours":72'
docker stop --time 10 "$name" >/dev/null
test "$(docker inspect "$name" --format '{{.State.ExitCode}}')" = 0
docker rm "$name" >/dev/null
docker run -d --name "$name" --network none --read-only --tmpfs /tmp:mode=1777 \
    --health-interval=1s --health-start-period=1s \
    --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null
wait_for_server
assert_unconfigured_indexers
docker exec "$name" sh -ec '
    database_root=/data/.storage/database
    generation_name=$(cat "$database_root/CURRENT")
    test "$generation_name" = "$(cat /data/smoke-generation)"
    test -s "$database_root/$generation_name/moji.db"
    test -s "$database_root/$generation_name/runtime.db"
    test -s "$database_root/$generation_name/cache/stashbox/cache.db"
    test "$(cat /data/smoke-marker)" = persisted
'
docker exec "$name" curl --fail --silent --show-error http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' \
    --data '{"query":"{ settings { system { stashBoxDataCache { ttlHours } } } }"}' \
    | grep -Fq '"ttlHours":72'
printf 'Image smoke test passed: %s\n' "$image"
