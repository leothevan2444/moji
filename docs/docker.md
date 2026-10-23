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

The image builds the React UI and static Go binary in separate stages. Its Alpine 3.23
runtime installs the distribution's `ffmpeg` package, which includes both
[ffmpeg and ffprobe](https://pkgs.alpinelinux.org/package/v3.23/community/x86_64/ffmpeg),
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

`docker-test` checks the non-root user, H.264/AAC media with embedded subtitle
conversion and two-part FFmpeg assembly, Docker health status, built UI,
frontend and GraphQL versions, and the OCI version label. It starts Moji with a
read-only root filesystem, saves settings through GraphQL, checks graceful
shutdown, and verifies config and SQLite persistence across container replacement.
It creates and removes its own containers and volume. CI runs the same check on
native amd64 and arm64 runners before publishing either architecture.

The binary's `--version`, frontend bundle, GraphQL `version`, image tag and OCI
version label all use the value derived from Git. `make docker-build` also records the
source commit in the OCI revision label. Local builds receive a `moji:<version>`
tag; no registry upload is performed. For a direct Docker build, supply
`--build-arg VERSION="$(sh scripts/version.sh)"` and
`--build-arg VCS_REF="$(git rev-parse HEAD)"`, and tag it with the same version.
FFmpeg's exact package version is determined at image build time
and can be inspected with the commands above. Base-image tags and Alpine package
updates mean rebuilding an application version is not byte-for-byte reproducible.

### Alpine migration validation (2026-10-23)

Before changing the repository Dockerfile, isolated candidate images were built
from clean source revision `c0cee9c9b506` with version `dev-c0cee9c9b506`.
The Node and Go build stages stayed Debian-based; only the runtime changed.
The arm64 candidate used native build stages to cross-compile the same static
Go binary, then executed the arm64 runtime and FFmpeg through QEMU on amd64.
Native ARM hardware was not used for this local validation; CI now covers it.

| Check | amd64 | arm64 (QEMU) |
| --- | --- | --- |
| `internal/mediafiles`, including real assembly, subtitles, COPY/MOVE, cancellation and restart recovery | 17 passed | 17 passed |
| `internal/config` | 13 passed | 13 passed |
| `internal/stashboxcache` | 8 passed | 8 passed |
| `internal/taskruntime`, including SQLite upgrades | 124 passed | 124 passed |
| `internal/graphqlapi` | 60 passed | 60 passed |
| `cmd/moji` | 18 passed | 18 passed |
| H.264/H.265, AAC/AC3, MP4/MKV, mov_text/SRT/WebVTT, Unicode filenames | Passed | Passed |
| Enhanced `docker-test` | Passed | Passed |
| UID/GID 1000 and 2000; read-only root; config save and recreation | Passed | Passed |
| Docker DNS, HTTP, verified local TLS, system CA roots, Asia/Shanghai timezone | Passed | Passed |
| SIGTERM via tini, exit code 0 within 10 seconds | Passed | Passed |

Test counts are top-level tests: 240 per architecture, with no skips. Test
binaries were compiled with `CGO_ENABLED=0` and run inside the candidate images.
The main package's tests used a writable temporary working directory because
one test writes a log file there. The app itself was separately tested with a
read-only root and writable data volume. The media suite also passed against
the same-source Debian baseline. A Debian-to-Alpine container replacement
preserved saved settings and the SQLite cache using the same data volume.

Local uncompressed image sizes for that clean source revision:

| Runtime | Architecture | Bytes | Decimal MB |
| --- | --- | ---: | ---: |
| Debian bookworm-slim | amd64 | 565441753 | 565.4 |
| Alpine 3.23 | amd64 | 169206214 | 169.2 |
| Alpine 3.23 | arm64 | 143548408 | 143.5 |

The amd64 reduction is 70.1%. These are local image sizes, not registry download
sizes. The tested Alpine package was FFmpeg 8.0.1; the Debian baseline used
FFmpeg 5.1.9. Validation covers Moji's exercised media operations and fixtures;
it does not establish compatibility with every possible media file or hardware
acceleration setup.

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
to publish the tested `linux/amd64` and `linux/arm64` images. The version must
match the triggering tag, and invalid or ambiguous tags stop publishing.

The Docker matrix builds each platform once on its native runner using Buildx,
loads that result into Docker, and runs `make docker-test` against it. Build
caches are separated into `moji-amd64` and `moji-arm64` scopes. Version arguments
are declared after dependency installation so a new Git version does not
invalidate the npm or Alpine package-install layers.

For publishable main/tag runs, each successful Docker job uploads
`moji-image-amd64` or `moji-image-arm64`, containing a Docker image archive and
its post-test inspection record. These artifacts stay within the current
workflow run and expire after one day. Pull requests and other branches run
the image checks without uploading publication artifacts or using registry
credentials.

After every required CI job passes, the reusable publication workflow downloads
both artifacts. `scripts/docker_publish.py` checks the Linux architecture,
Git-derived version and source revision against the current run, loads the
archives, and verifies their image IDs against the post-test records. Both
images must pass before any registry push. It pushes the verified IDs under
unique `ci-<run-id>-<attempt>-<arch>` tags and merges their immutable registry
digests with `docker buildx imagetools create`. These staging tags remain in the
repository; they share the image layers with release tags. No QEMU or second
image build is used by publication. OCI version/revision labels stay in each
tested image; metadata-action controls the release tags on the final manifest
list. Docker archives do not retain BuildKit provenance attestations.

Configure the repository variable `DOCKERHUB_USERNAME` and the secret
`DOCKERHUB_TOKEN` with permission to push `<username>/moji`. Stable tags publish
the version and update `latest`. Prereleases publish their version without
updating `latest`. Successful `main` pushes update `edge`; every published image
also gets a `sha-<commit>` tag. To retry manually, run the CI workflow on `main`
or a release tag; publication still waits for its tests. To retry a failed
publication after its image artifacts expire, rerun the whole CI workflow.
`make docker-build` and Compose continue to build locally and do not push images.

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
