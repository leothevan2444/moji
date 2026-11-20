# Getting started

Moji is a companion for Stash. It finds releases, sends downloads to qBittorrent, delivers completed content to a Stash library, and asks Stash to scan it. Stash remains the system that catalogs and presents your media.

> Moji does not install or configure Stash, Jackett, qBittorrent, or StashBox for you.

## Before you begin

Downloading and ingest need running Stash and qBittorrent instances. Configure at least one StashBox endpoint in Stash to confirm the film for each task, including an API key when required. Enable a usable built-in site or an external Jackett site for sourcing; Jackett is optional.

The Moji server must be able to reach the services and sites you use. If you use file delivery, it must also be able to read the download directory and write to the library directory.

## First setup

1. Download definitions and add built-in sites, or configure and enable optional Jackett, under **Settings > Indexers**; configure qBittorrent under **Settings > Downloader**, and Stash under **Settings > Stash**. Changes save automatically.
2. Return to **Home**. A service is ready only after a recent probe succeeds; “configured” alone does not prove connectivity.
3. Open **Settings > Stash**. Changes save automatically. Configure the qB download, Moji download, Moji library, and Stash library roots, then choose Copy or Move.
4. Open **Settings > Tasks & subscriptions**. Choose a task processing frequency and review release and resource policies. StashBox priority is under **Settings > Stash**. Changes save automatically.
5. Search in **Discover**, or open **Performers** and queue a scene. Follow the resulting item in **Tasks** until delivery, Stash scanning, generation, identification and verification complete.

## A safe first run

Search for and confirm a StashBox film first. Keep file delivery on **Copy** until path mapping is verified. Confirm that the qBittorrent content path is under the configured qB download root, and that the final scan path is inside a Stash library. After the task completes, verify the scene in Stash before enabling broader subscription automation.
