# Docker packaging

Git tags are the only release version source. A clean checkout with a single
release tag such as `v0.2.0` builds version `0.2.0`; `v0.3.0-rc.1` builds
`0.3.0-rc.1`. Other commits use `dev-<12-character-commit>`, with `-dirty` appended
for tracked changes or untracked files. Ignored build output and local data do
not affect the version. Release tags must follow SemVer without `+metadata` and
fit Docker's 128-character tag limit. Invalid or ambiguous `v*` tags fail the
build. Use one release tag per commit.

Make, Vite and `docker/compose.sh` resolve the version with `scripts/version.sh`.
Make and Compose ignore inherited version overrides. Docker receives the resolved
version through the required `VERSION` build argument, injects it into the Go
binary and frontend build, and uses it for the OCI version label. Git metadata
is not sent in the Docker context and Git is not needed in the runtime image.
The private npm package and lockfile have no independent application version.
Plain `go build`/`go run` produce `dev`; Make builds inject the Git-derived version.

The image builds the React UI and Go binary in separate stages. Its Debian
runtime installs the distribution's `ffmpeg` package, which includes both
[ffmpeg and ffprobe](https://packages.debian.org/bookworm/amd64/ffmpeg/filelist),
along with CA certificates, timezone data, curl for health checks and tini for
process supervision. Node and Go build tools are absent from the runtime.

## Build and inspect

From the repository root:

```sh
make docker-build
version=$(sh scripts/version.sh)
docker run --rm "moji:$version" --version
docker run --rm --entrypoint ffmpeg "moji:$version" -version
docker run --rm --entrypoint ffprobe "moji:$version" -version
docker image inspect "moji:$version" --format '{{json .Config.Labels}}'
make docker-test
```

`docker-test` checks the non-root user, real two-part FFmpeg assembly, HTTP
health, built UI, frontend and GraphQL versions, and persistent data across container replacement.
It creates and removes its own container and volume. CI runs this same check
and verifies the OCI version label.

The binary's `--version`, frontend bundle, GraphQL `version`, image tag and OCI
version label all use the value derived from Git. `make docker-build` also records the
source commit in the OCI revision label. Local builds receive a `moji:<version>`
tag; no registry upload is performed. For a direct Docker build, supply
`--build-arg VERSION="$(sh scripts/version.sh)"` and
`--build-arg VCS_REF="$(git rev-parse HEAD)"`, and tag it with the same version.
FFmpeg's exact package version is determined at image build time
and can be inspected with the commands above. Base-image tags and Debian package
updates mean rebuilding an application version is not byte-for-byte reproducible.

## Start with Compose

From the repository root, create the bind-mount directories as the user that
will run Moji:

```sh
mkdir -p docker/data docker/downloads docker/library
cp docker/config.example.yaml docker/data/config.yaml
```

Edit `docker/data/config.yaml`: Jackett URL and API key are required at startup;
fill in qBittorrent and Stash connection details to enable the complete workflow.
The example host addresses assume those services are published on the Docker
host. If they run on a shared Docker network, use their service names and attach
Moji to that network instead. `localhost` inside Moji means the Moji container.

For bind-mount permissions, write your host UID/GID into the ignored
`docker/.env` file (the image defaults to `1000:1000`):

```sh
printf 'MOJI_UID=%s\nMOJI_GID=%s\n' "$(id -u)" "$(id -g)" > docker/.env
./docker/compose.sh up -d --build
```

Open <http://localhost:10000>. To expose it on the local network, set
`MOJI_BIND_ADDRESS=0.0.0.0` in `docker/.env`. Other optional Compose settings are
`MOJI_PORT`, `MOJI_DOWNLOADS_DIR`, `MOJI_LIBRARY_DIR`, and `TZ`. Relative bind-mount
paths resolve from the `docker/` directory. Existing media directories must be
readable by Moji; the library directory must be writable. MOVE also requires
write permission to the downloads directory. Use absolute host paths for
existing downloads and library directories.

Use `docker/compose.sh` for Compose commands, such as `./docker/compose.sh logs`
and `./docker/compose.sh down`. It resolves the Git version and revision from the
repository regardless of the working directory; do not maintain `MOJI_VERSION`
or `MOJI_VCS_REF` in `.env`. Keep the same checkout while operating a deployment.
After checking out another revision, restart development servers or rebuild
production artifacts to pick up the new compiled version.

## Automatic publishing

Push a `v*` release tag after committing the source changes; no version file
needs updating. CI fetches tags and runs all Go, frontend, visual and Docker
smoke checks. Its publish job then calls `.github/workflows/docker-publish.yml`
to build and push `linux/amd64` and `linux/arm64` images. The version must match
the triggering tag, and invalid or ambiguous tags stop publishing.

Configure the repository variable `DOCKERHUB_USERNAME` and the secret
`DOCKERHUB_TOKEN` with permission to push `<username>/moji`. Stable tags publish
the version and update `latest`. Prereleases publish their version without
updating `latest`. Successful `main` pushes update `edge`; every published image
also gets a `sha-<commit>` tag. To retry manually, run the CI workflow on `main`
or a release tag; publication still waits for its tests. `make docker-build`
and Compose continue to build locally and do not push images.

## Persistent data and media paths

| Container path | Purpose |
| --- | --- |
| `/data` | Writable config, SQLite databases, logs and caches |
| `/downloads` | The download directory shared with qBittorrent |
| `/library` | Prepared movies and subtitles shared with Stash |
| `/opt/moji/web/dist` | Built UI, outside the persistent data mount |

Mount the entire `/data` directory: the UI saves config changes by replacing the
config file. A standalone config-file bind mount can prevent those writes.
Container removal or replacement preserves host bind-mount data.

`ingest.downloads.qb_root` must match the path **reported by qBittorrent**;
`moji_root` is the corresponding path inside Moji. Similarly,
`ingest.library.stash_root` is the shared library path **inside Stash**, while
`moji_root` is its path inside Moji. For example, Stash can see `/media/moji`
while Moji sees `/library`, provided both mount the same host directory. Add
that path to Stash's library configuration. Stash still owns metadata scanning,
generation and identification.

The image's working directory is `/data`, `MOJI_CONFIG` defaults to
`/data/config.yaml`, and `MOJI_WEB_DIST` defaults to `/opt/moji/web/dist`.
Preparation temporary files use `/data` or the destination library filesystem.
Plan free space for the final assembled movie in addition to source downloads.
The health check probes `/healthz` on port 10000; changing the internal listening
port also requires updating the health check.

Before upgrading, stop Moji and back up `/data`, including config and SQLite
files. Update to a new source release, rebuild, and recreate the container with
the same mounts. For this source-built Compose setup the image is built locally;
there is no assumed public registry or `latest` tag.
