# Connections and ingest

These settings define how Moji reaches its dependencies and how a completed download becomes visible to Stash. Save each form independently.

## Connections

Configure the base URL and credentials for Stash, Jackett, and qBittorrent. Moji derives the required API endpoints from those base URLs. A service is **configured** when its minimum fields are present and **ready** only after a recent probe succeeds.

qBittorrent's default save path, category, and tags are applied to new tasks unless an operation supplies an override. The Jackett dashboard password is used when torrent files require authenticated retrieval. Credentials are returned to the current settings UI, so access to Moji should be restricted.

## Ingest

Moji makes the task save directory relative to the **qB download root**, then appends it to the **Moji download root** to read selected files. It filters, deduplicates, and merges the movie into `code/code.mkv` under the **Moji library root**, together with retained subtitles.

The **Stash library root** must expose the same files as the Moji library root. Moji appends the movie code directory to this root and asks Stash to scan the delivered movie.

All four roots use the named service's filesystem perspective and must match container mounts. **Copy** retains downloaded sources. **Move** stops the qB task and deletes selected sources only after the entire movie and subtitles are published. Existing movies are not overwritten. The Moji runtime requires `ffmpeg` and `ffprobe`.
