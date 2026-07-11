# Getting started

Moji is a companion for Stash. It finds releases, sends downloads to qBittorrent, delivers completed content to a Stash library, and asks Stash to scan it. Stash remains the system that catalogs and presents your media.

> Moji does not install or configure Stash, Jackett, qBittorrent, or StashBox for you.

## Before you begin

You need running instances of Stash, Jackett, and qBittorrent. Add at least one configured indexer in Jackett. For metadata discovery and performer subscriptions, configure at least one StashBox endpoint in Stash, including its API key when that endpoint requires one.

The Moji server must be able to reach all three services. If you use file delivery, it must also be able to read the download directory and write to the library directory.

## First setup

1. Open **Settings > Connections** and save the URL and credentials for each service.
2. Return to **Home**. A service is ready only after a recent probe succeeds; “configured” alone does not prove connectivity.
3. Open **Settings > Ingest**. Choose Path mapping when qBittorrent and Stash already see the same files, or File delivery when Moji must copy, move, or link them into the library.
4. Open **Settings > Automation**. Set progress and subscription intervals, refresh the StashBox list, and review the release and torrent-selection policies.
5. Search in **Discover**, or open **Performers** and queue a scene. Follow the resulting item in **Tasks** until Stash scanning completes.

## A safe first run

Use a release code with one unambiguous result. Keep file delivery on **Copy** until path mapping is verified. Confirm that the qBittorrent content path is under the configured qB download root, and that the final scan path is inside a Stash library. After the task completes, verify the scene in Stash before enabling broader subscription automation.
