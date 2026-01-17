# Moji
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
- optional: [PostgreSQL](https://www.postgresql.org/) instance and [r18dev](https://r18.dev/) database deployed if you need to build from scratch. ([how to deploy r18dev database](./docs/deploy_r18dev_database.md))
### Docker Compose
Get the [`docker-compose.yml`](./docker/docker-compose.yml) file from repository.

Once you have that file where you want it, modify the settings as you please, and then run:
```shell
docker compose up -d
```
Moji will by default be binded to port 10000. Web UI is available in your web browser locally at http://localhost:10000 or on your network as http://YOUR-LOCAL-IP:10000

