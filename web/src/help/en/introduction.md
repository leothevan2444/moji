# Introduction

Moji is a self-hosted companion for Stash. It coordinates release discovery, torrent selection, qBittorrent downloads, delivery into a media library, and the final Stash scan.

## What Moji manages

Moji stores its own download tasks and performer subscriptions. It searches metadata through the StashBox endpoints configured in Stash and searches torrents through Jackett. Completed downloads are mapped or delivered into a Stash library before Moji requests a Stash scan.

Stash remains the source of truth for performers, library scenes, media paths, and presentation. qBittorrent remains the source of truth for torrent transfer state. Jackett remains responsible for indexer configuration and search availability.

## Main workflow

1. Find a scene in **Discover** or under a performer.
2. Create a task from structured metadata or a selected torrent.
3. Let Moji source and monitor the qBittorrent download.
4. Map or deliver the completed content to the library.
5. Let Stash scan the resolved path.

Moji marks the task complete only after the associated Stash scan job completes.
