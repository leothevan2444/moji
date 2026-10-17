# Moji

Moji is a self-hosted companion for [Stash](https://github.com/stashapp/stash)
that tracks JAV performers and releases, searches for torrents through
[Jackett](https://github.com/Jackett/Jackett), and downloads them with
[qBittorrent](https://www.qbittorrent.org/).

Moji prepares and delivers the movie files. Stash handles library scanning,
metadata identification, and generated playback resources. Moji orchestrates
those Stash operations rather than writing movie metadata itself.

## Current features

- Performer subscriptions and manual movie tasks, with persisted progress,
  retries, and operator intervention in the web UI.
- Torrent file selection that excludes recognized advertisements, samples,
  screenshots, and other extras. Ambiguous videos wait for manual selection.
- Release-code matching across common filename variants, ordered movie-part
  assembly, and removal of byte-identical duplicate video files.
- Delivery as a single `CODE/CODE.mkv`. Compatible parts are joined without
  re-encoding; incompatible parts block delivery for review.
- Optional retention of existing subtitles in every language, including time
  shifting and merging supported text subtitles for movie parts. Retention is
  off by default. Moji does not generate or translate subtitles.
- Stash scan, generation, and identification after file delivery. Metadata
  completeness depends on the identification sources configured in Stash.
- A web UI in Simplified Chinese and English, with task updates, service status,
  logs, and runtime settings.

See [media delivery](docs/media-delivery.md) for classification, merging,
subtitle handling, and recovery behavior.

## Docker quick start

Docker Compose is the recommended deployment method. The image includes the
built web UI, Moji, FFmpeg, and ffprobe; host installation of FFmpeg is not needed.
The following steps build a local image from source, without assuming a public
registry image.

You need Docker with Compose and running Jackett, qBittorrent, and Stash services
reachable from the Moji container. Configure Stash's library and identification
sources for the movies Moji will deliver.

Clone the repository, then prepare the directories and configuration:

```sh
git clone https://github.com/leothevan2444/moji.git
cd moji
mkdir -p docker/data docker/downloads docker/library
cp docker/config.example.yaml docker/data/config.yaml
printf 'MOJI_UID=%s\nMOJI_GID=%s\n' "$(id -u)" "$(id -g)" > docker/.env
```

Edit `docker/data/config.yaml` before starting:

- Set the required Jackett URL and API key. Set its dashboard password to enable
  the dashboard's Jackett status checks.
- Set the qBittorrent URL, username, and password to enable downloading.
- Set the Stash URL and API key, if authentication is enabled.
- Set the download and library roots described below. For an existing media
  setup, point the Compose mounts at those existing host directories.

Then build and start Moji:

```sh
./docker/compose.sh up -d --build
./docker/compose.sh exec moji moji --version
```

Open <http://localhost:10000>. The default port binding accepts connections from
the Docker host only. To allow access from your local network, add
`MOJI_BIND_ADDRESS=0.0.0.0` to `docker/.env` and recreate the container.

Use the wrapper for subsequent Compose commands so the image version is always
read from the repository's [VERSION](VERSION):

```sh
./docker/compose.sh ps
./docker/compose.sh logs -f moji
./docker/compose.sh down
```

## Configuration and directory mapping

The image runs as a non-root user. The quick start sets its UID and GID to match
your host user; existing media directories must grant that user the access
needed for reading downloads and writing prepared movies.

The default mounts, relative to the repository root, are:

| Host directory | Moji container path | Purpose |
| --- | --- | --- |
| `docker/data` | `/data` | Configuration, SQLite databases, logs, and caches |
| `docker/downloads` | `/downloads` | Downloads shared with qBittorrent |
| `docker/library` | `/library` | Prepared movies shared with Stash |

Mount the entire `/data` directory so UI configuration changes can be saved.
Configuration is stored at `/data/config.yaml`. To use existing media folders,
set `MOJI_DOWNLOADS_DIR` and `MOJI_LIBRARY_DIR` to their absolute host paths in
`docker/.env`.

The same host files may have different paths in each service:

- `ingest.downloads.qb_root` is the download root reported by qBittorrent;
  `ingest.downloads.moji_root` is that directory as seen inside Moji.
- `ingest.library.moji_root` is the destination inside Moji;
  `ingest.library.stash_root` is that same directory inside Stash. Add the Stash
  path to Stash's configured libraries.

All four roots must be configured for delivery. The example uses `/downloads`
for both download roots and `/library` for both library roots. Moji and Stash
must share the actual library directory, not just use matching path names.

`ingest.transfer.action` accepts `COPY` or `MOVE`. COPY retains downloaded
sources. MOVE stops the torrent and removes selected sources after the prepared
movie and subtitles have been delivered; it also requires write permission to
the download directory. `ingest.keep_subtitles` controls subtitle retention.

The example service URLs use `host.docker.internal` to reach ports published on
the Docker host. Use service names when connecting containers on a shared Docker
network, and attach Moji to that network. `localhost` inside Moji refers to the
Moji container.

See [Docker packaging](docs/docker.md) for all Compose options, image inspection,
permissions, and detailed path mapping.

## Versions and upgrades

The root [VERSION](VERSION) is the only application version source. Go embeds it
at compile time, Vite uses it for the frontend bundle and `web/dist/version.json`,
and the Docker build uses it for the image tag and OCI version label. The private
web package does not maintain a separate release version.

When preparing a release, change `VERSION` and rebuild. Development servers must
be restarted to pick up a changed version. Check the running version with:

```sh
./docker/compose.sh exec moji moji --version
```

To upgrade an installation, stop Moji and back up `docker/data` (or your custom
`/data` mount), including the config and SQLite files. Check out the desired
source revision, then rebuild and recreate the container with the same mounts:

```sh
./docker/compose.sh up -d --build
```

Container replacement preserves data in the host-mounted directories.

## Development and verification

Native execution is supported. Use Go as specified in [go.mod](go.mod), Node.js
22, npm, and FFmpeg with ffprobe in `PATH` for movie preparation.

Create `config.yaml` from [the example](docker/config.example.yaml), replacing
container service URLs and `/data`, `/downloads`, and `/library` paths with your
local values, including `logging.file_path`. Then build and run from the
repository root:

```sh
npm --prefix web ci
make web-build
make moji
./moji -config ./config.yaml
```

For frontend development, run `make web-dev` in another terminal. Vite serves
the UI on port 5173 and proxies API requests to the backend on port 10000 by
default. Override the config path with `-config` or `MOJI_CONFIG`; the backend
listen address can be changed with `-addr`.

Run the relevant checks from the repository root:

```sh
go test ./...
npm --prefix web test
npm --prefix web run i18n:check
npm --prefix web run build
make test-media
make docker-build
make docker-test
```

`test-media` requires FFmpeg and ffprobe and exercises real movie-part assembly
and subtitle processing. `docker-test` uses temporary containers and a volume to
verify FFmpeg assembly, the web UI, frontend and backend versions, and data
persistence across container replacement; it cleans up its test resources.

After changing Moji's GraphQL schema, run `make generate-moji-graphql` to update
the Go and frontend bindings. CI covers backend tests, frontend tests,
internationalization, the web build, visual regression, and Docker smoke tests.

### Internationalization

The UI supports Simplified Chinese (`zh-CN`) and English (`en`), with an `auto`
preference that follows the browser. UI copy uses stable semantic `i18next` keys.
After editing locale resources, run `npm --prefix web run i18n:export`, then
`npm --prefix web run i18n:check`. The check enforces aligned locale keys,
non-empty values, matching interpolation variables, and no hard-coded CJK UI
strings.

`qps-ploc` and `ar-XB` are development resources for text expansion and RTL
checks, rather than user-facing locales. New locales need complete resources,
pseudo/RTL visual review, and review by a fluent speaker before being added to
`SUPPORTED_LOCALES`.

GraphQL UI errors use `extensions.code`, `params`, and `correlationId`. Translate
the stable error code instead of displaying raw server diagnostics; unknown codes
use a generic localized message, with the correlation ID available for log lookup.
