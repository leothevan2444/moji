# Moji

## Internationalization

The web UI uses stable, language-neutral domain values and `i18next` resources for user-facing copy. The current locales are Simplified Chinese (`zh-CN`) and English (`en`); the saved preference may also be `auto` to follow the browser.

Run `npm run i18n:audit` in `web/` to report possible hard-coded CJK UI strings. New UI copy should use semantic translation keys grouped by feature rather than source text as keys. Run `npm run i18n:audit:strict` once the reported migration backlog reaches zero.

The CI gate uses `npm run i18n:check` to enforce zero hard-coded UI strings, matching official locale key sets, non-empty values, and matching interpolation variables. `qps-ploc` and `ar-XB` are development-only test resources for expansion and RTL checks; they are not user-facing locales. New locales must provide the complete key set, pass pseudo/RTL visual review, and be reviewed by a fluent speaker before being added to `SUPPORTED_LOCALES`.

GraphQL user-visible failures use `extensions.code`, `params`, and `correlationId`. UI code must translate the stable code and must not display the server diagnostic message. Unknown codes show a generic localized message while the correlation ID links the UI report to server logs.
### **Moji is a self-hosted service for automatically downloading JAV and cooperating with [Stash](https://github.com/stashapp/stash).**
- Moji integrate with Stash to fetch and manage your JAV collection.
- Moji helps you to build a personal JAV library from scratch.
- Moji traces JAV actresses and releases, downloading new content as it becomes available.
- Automatic generate subtitles for videos.
- User-friendly web interface for easy management.

## Installing Moji
**Moji now only support installation via Docker.**
### Prerequisites
- Docker and Docker Compose installed on your server.
- FFmpeg and ffprobe are included in the Docker image. Native runs need both in PATH.
- A running instance of [Stash](https://github.com/stashapp/stash).
- [jackett](https://github.com/Jackett/Jackett) and [qBittorrent](https://www.qbittorrent.org/) for torrent searching and downloading.
### Docker Compose
Clone the repository. The release version comes from [VERSION](VERSION). Prepare persistent
directories and copy the example configuration:
```shell
mkdir -p docker/data docker/downloads docker/library
cp docker/config.example.yaml docker/data/config.yaml
printf 'MOJI_UID=%s\nMOJI_GID=%s\n' "$(id -u)" "$(id -g)" > docker/.env
```
Fill in connection settings in `docker/data/config.yaml`, especially the required
Jackett URL/API key, and configure the media mounts for qBittorrent and Stash.
Then build and start:

```shell
./docker/compose.sh up -d --build
./docker/compose.sh exec moji moji --version
```

The UI is available at <http://localhost:10000>. The image runs as a non-root
user and stores config, databases, logs and caches under `/data`.
See [Docker packaging](docs/docker.md) for permissions, path mapping, version
labels, upgrades and image smoke tests. This builds a local `moji:<VERSION>` image;
no public registry image is required.
