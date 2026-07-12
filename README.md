# Moji

## Internationalization

The web UI uses stable, language-neutral domain values and `i18next` resources for user-facing copy. The current locales are Simplified Chinese (`zh-CN`) and English (`en`); the saved preference may also be `auto` to follow the browser.

Run `npm run i18n:audit` in `web/` to report possible hard-coded CJK UI strings. New UI copy should use semantic translation keys grouped by feature rather than source text as keys. Run `npm run i18n:audit:strict` once the reported migration backlog reaches zero.
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
- A running instance of [Stash](https://github.com/stashapp/stash).
- [jackett](https://github.com/Jackett/Jackett) and [qBittorrent](https://www.qbittorrent.org/) for torrent searching and downloading.
### Docker Compose
Get the [`docker-compose.yml`](./docker/docker-compose.yml) file from repository.

Once you have that file where you want it, modify the settings as you please, and then run:
```shell
docker compose up -d
```
Moji will by default be binded to port 10000. Web UI is available in your web browser locally at http://localhost:10000 or on your network as http://YOUR-LOCAL-IP:10000
