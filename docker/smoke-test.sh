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
test "$(docker run --rm "$image" --version)" = "moji $version"
test "$(docker image inspect "$image" --format '{{index .Config.Labels "org.opencontainers.image.version"}}')" = "$version"
docker run --rm --entrypoint sh "$image" -ec '
    test "$(id -u)" != 0
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

docker run -d --name "$name" --read-only --tmpfs /tmp:mode=1777 \
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

docker exec "$name" curl --fail --silent http://127.0.0.1:10000/ \
    | grep -qi '<!doctype html>'
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' --data '{"query":"{ version }"}' \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" curl --fail --silent http://127.0.0.1:10000/version.json \
    | grep -Fq "\"version\":\"$version\""
docker exec "$name" sh -ec 'test -s /data/cache/stashbox/cache.db; printf persisted > /data/smoke-marker'
docker exec "$name" curl --fail --silent --show-error http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' \
    --data '{"query":"mutation { updateSystemSettings(input: {taskDeletePolicy: KEEP_ONLY, stashBoxDataCache: {ttlHours: 72}}) { system { stashBoxDataCache { ttlHours } } } }"}' \
    | grep -Fq '"ttlHours":72'
docker stop --time 10 "$name" >/dev/null
test "$(docker inspect "$name" --format '{{.State.ExitCode}}')" = 0
docker rm "$name" >/dev/null
docker run -d --name "$name" --read-only --tmpfs /tmp:mode=1777 \
    --health-interval=1s --health-start-period=1s \
    --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null
wait_for_server
docker exec "$name" sh -ec 'test -s /data/cache/stashbox/cache.db; test "$(cat /data/smoke-marker)" = persisted'
docker exec "$name" curl --fail --silent --show-error http://127.0.0.1:10000/graphql \
    -H 'Content-Type: application/json' \
    --data '{"query":"{ settings { system { stashBoxDataCache { ttlHours } } } }"}' \
    | grep -Fq '"ttlHours":72'
printf 'Image smoke test passed: %s\n' "$image"
