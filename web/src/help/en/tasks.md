# Tasks and ingest

A task is Moji's durable record for one release. It keeps the selected torrent, qBittorrent state, delivery paths, Stash scan job, errors, and timestamps. Removing a task follows the deletion policy in **Settings > System**.

## Task lifecycle

1. **Sourcing** — Moji searches Jackett and ranks downloadable candidates, unless you selected a Jackett result manually.
2. **Downloading** — the torrent has been submitted to qBittorrent. Moji periodically synchronizes its name, progress, state, and content path.
3. **Pending ingest** — the download is complete and ready for path planning.
4. **Transferring** — only in File delivery mode; Moji copies, moves, or symlinks the content.
5. **Scanning** — Moji starts a Stash metadata scan for the resolved library path and monitors the Stash job.
6. **Completed** — Stash reported that scan job complete.

A stage can be pending, running, blocked, or done. “Blocked” is actionable and preserves the error. Retry resumes from the current stage; it does not restart a completed download.

## Working with tasks

Search includes codes, titles, trackers, stages, torrent identities, and paths. Filter by running, completed, blocked, or pending scan, then sort by creation time, update time, or progress. **Refresh** reloads stored state. **Sync progress** asks qBittorrent for current transfer state. **Trigger scans** processes eligible completed downloads. **Retry blocked tasks** retries all blocked items.

When sourcing finds no acceptable torrent, open **Resolve** to search candidates and choose one manually. Before submitting, confirm the candidate belongs to the requested release. Deleting a task may also delete its qBittorrent item and files, depending on the system policy; the confirmation dialog states the active effect.

## Ingest modes

Path mapping does not touch files. Moji removes the configured **qB download root** from qBittorrent's content path, appends that relative path to the **Stash library root**, and asks Stash to scan the result. Use it only when both services already access the same content through different path prefixes.

File delivery first maps that same relative path onto the **Moji download root**. It then copies, moves, or symlinks the source to the relative location under the **Moji library root**. Finally, it maps the relative path onto the **Stash library root** for scanning. Existing targets cause the transfer to fail; Moji does not overwrite them.

`Copy` keeps the download source. `Move` removes it after a successful move and can affect qBittorrent seeding. `Symlink` requires the link target to remain reachable from the Moji host; it is usually unsuitable across container-only paths or hosts.
