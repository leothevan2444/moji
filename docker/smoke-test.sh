#!/bin/sh
set -eu

version=$(cat "$(dirname "$0")/../VERSION")
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
test "$(docker run --rm "$image" --version)" = "moji $version"
docker run --rm --entrypoint sh "$image" -ec '
    test "$(id -u)" != 0
    ffprobe -version >/dev/null
    work=$(mktemp -d)
    trap '\''rm -rf "$work"'\'' EXIT
    cd "$work"
    for color in red blue; do
        ffmpeg -v error -f lavfi -i "color=c=$color:s=64x64:r=10:d=1" \
            -an -c:v mpeg4 -threads 1 "$color.mkv"
    done
    printf "file red.mkv\nfile blue.mkv\n" > parts.txt
    ffmpeg -v error -f concat -safe 0 -i parts.txt -c copy merged.mkv
    ffprobe -v error -show_entries format=duration -of csv=p=0 merged.mkv \
        | awk '\''{ if ($1 < 1.8 || $1 > 2.2) exit 1; seen=1 } END { if (!seen) exit 1 }'\''
'

docker volume create "$volume" >/dev/null
docker run --rm -i --entrypoint sh \
    --mount "type=volume,src=$volume,dst=/data" "$image" \
    -c 'cat > /data/config.yaml' <<'YAML'
connection:
  jackett:
    url: http://127.0.0.1:1
    api_key: smoke-test
automation:
  task_progress_sync_interval_seconds: -1
  subscription_poll_interval_hours: -1
YAML

docker run -d --name "$name" \
    --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null

wait_for_server() {
    attempt=0
    until docker exec "$name" curl --fail --silent http://127.0.0.1:10000/healthz >/dev/null; do
        attempt=$((attempt + 1))
        if [ "$attempt" -ge 30 ]; then
            docker logs "$name"
            exit 1
        fi
        sleep 1
    done
}
wait_for_server

docker exec "$name" curl --fail --silent http://127.0.0.1:10000/ \
    | grep -qi '<!doctype html>'
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' --data '{"query":"{ version }"}' \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/version.json \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" sh -ec 'test -s /data/cache/stashbox/cache.db; printf persisted > /data/smoke-marker'
docker stop "$name" >/dev/null
docker rm "$name" >/dev/null
docker run -d --name "$name" \
    --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null
wait_for_server
docker exec "$name" sh -ec 'test -s /data/cache/stashbox/cache.db; test "$(cat /data/smoke-marker)" = persisted'
printf 'Image smoke test passed: %s\n' "$image"
