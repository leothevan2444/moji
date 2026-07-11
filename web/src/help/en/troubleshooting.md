# Troubleshooting

Start with **Home** for service readiness, open the affected task for its stage error and paths, then use **Settings > Logs** for backend detail. Preserve the task while diagnosing it; its stored paths and upstream identifiers are evidence.

## Connection and search problems

- **Configured but not ready:** the fields are present, but the latest probe failed or is stale. Verify the URL from the Moji server or container, credentials, TLS trust, and reverse-proxy path.
- **No StashBox results:** confirm that Stash has at least one StashBox endpoint, refresh the list in Automation, and check its API key. Moji tries endpoints in order and returns results from the first successful match.
- **No Jackett results:** confirm Jackett is ready, at least one indexer is configured, and active filters are not excluding it. Clear tracker filters and retry the exact release code.
- **No downloadable candidate:** results existed but did not expose a usable magnet or torrent URL, or selection rules ranked no usable item. Resolve the blocked task manually or adjust the rules.
- **Duplicate rejected:** Moji prevents a second task for the same code or torrent and checks whether the code already exists in Stash. Use the existing task or library item.

## Download, ingest, and scan problems

- **Progress is stale:** use **Sync progress**. Then verify the torrent still exists in the configured qBittorrent instance and that its identity matches the task.
- **Content path is outside qB root:** qBittorrent reported a path that is not below the configured qB download root. Correct the root from qBittorrent's filesystem view; do not use Moji's host path in that field.
- **Transfer failed:** verify the Moji download and library roots, permissions, free space, and that the target does not already exist. For symlinks, verify the source remains reachable.
- **Scan path is wrong:** compare the task's relative path with the Stash library root. The final path must use Stash's filesystem view and be inside a configured Stash library.
- **Stash scan failed or stalled:** verify Stash readiness and open the task for the scan job ID and error. Retry the task after the underlying Stash job or path problem is resolved.

Do not repeatedly retry a path or permission failure without changing its cause. Retry uses the stored task and current settings, so it will reproduce the same result when neither has changed.
