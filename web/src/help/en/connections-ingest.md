# Connections and ingest

These settings define how Moji reaches its dependencies and how a completed download becomes visible to Stash. Save each form independently.

## Connections

Configure the base URL and credentials for Stash, Jackett, and qBittorrent. Moji derives the required API endpoints from those base URLs. A service is **configured** when its minimum fields are present and **ready** only after a recent probe succeeds.

qBittorrent's default save path, category, and tags are applied to new tasks unless an operation supplies an override. The Jackett dashboard password is used when torrent files require authenticated retrieval. Credentials are returned to the current settings UI, so access to Moji should be restricted.

## Ingest

Path mapping does not touch files. Moji removes the **qB download root** from qBittorrent's content path and appends the relative path to the **Stash library root**. Use it only when qBittorrent and Stash already see the same files through different prefixes.

File delivery maps that relative path onto the **Moji download root**, then copies, moves, or symlinks it under the **Moji library root**. The same relative path is appended to the **Stash library root** for scanning. Existing targets are not overwritten.

Every root must use the named service's filesystem view. Container paths must match that container's mounts. `Copy` preserves the source, `Move` can interrupt seeding, and `Symlink` requires the source to remain reachable from Moji.
