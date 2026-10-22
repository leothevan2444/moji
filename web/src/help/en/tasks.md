# Tasks and ingest

A task is Moji's durable record for one release. It keeps the selected torrent, qBittorrent state, delivery paths, Stash scan job, errors, and timestamps. Removing a task follows the deletion policy in **Settings > System**.

## Task lifecycle

1. **Sourcing** — Moji searches Jackett and ranks downloadable candidates, unless you selected a Jackett result manually.
2. **Downloading** — the torrent has been submitted to qBittorrent. Moji periodically synchronizes its name, progress, state, and content path.
3. **Pending ingest** — the download is complete and ready for path planning.
4. **Transferring** — Moji filters and deduplicates files, losslessly merges movie parts in order, and delivers the movie with retained matching subtitles. Single files keep their original format at `CODE/CODE.ext`; merged parts use `CODE/CODE.mkv`.
5. **Stash ingest** — Moji schedules scan, generation, and identification in order and monitors each job. The UI shows the active step. Generation prepares screenshot covers, previews, sprites, and pHash for the target scenes before identification. Full video transcoding is off by default. Enable **Settings → Ingest → Generate video transcodes** to let Stash generate transcodes when needed.
6. **Completed** — after all jobs succeed, Moji reads the target scenes to verify media files and matching code, title, date, studio, and performers. Missing or mismatched metadata keeps the task blocked.

Configure and save sources and field policies under **Stash > Settings > Tasks > Identify** first. Moji uses Stash's saved source order, global options, and source-specific options. It does not write scene metadata using `sceneUpdate`. If sources are missing or matching is incomplete, adjust settings or review the result in Stash, then retry in Moji. Generation and identification are scoped to the scenes collected for this task, never the entire library.

A stage can be pending, running, blocked, or done. “Blocked” is actionable and preserves the error. Retry resumes from the blocked step without repeating a completed download, successful transfer, scan, or generation.


Before generation, identification, and final verification, Moji resolves the delivered path to current Stash scenes. Deleted or rebuilt records restart generation with refreshed IDs without repeating downloads or delivery. Each run permits one automatic recovery, persisted across restarts. Repeated changes or missing scenes block the task with a specific explanation. Manual retry starts a new recovery attempt.

## Working with tasks

Search includes codes, titles, trackers, stages, torrent identities, and paths. Filter by running, completed, blocked, or pending scan, then sort by creation time, update time, or progress. **Refresh** reloads stored state. **Sync progress** asks qBittorrent for current transfer state. **Trigger scans** processes eligible completed downloads. **Retry blocked tasks** retries all blocked items.

When sourcing finds no acceptable torrent, open **Resolve** to search candidates and choose one manually. Before submitting, confirm the candidate belongs to the requested release. Deleting a task may also delete its qBittorrent item and files, depending on the system policy; the confirmation dialog states the active effect.

## File preparation and delivery

Files have four classes: movie/parts, subtitles, known advertisements/samples/promotional text/screenshots, and videos requiring review. Known extras are skipped for download and delivery. **Settings > Ingest > Keep subtitles** controls subtitles in every language; when off, external subtitles are skipped. Single movies are delivered unchanged, including embedded subtitles. Merging multiple parts also applies this setting to embedded subtitles.

The task code anchors filename matching. Advertising domains are ignored, and matching supports case, separators, zero-padding variants, and A/B or -1/-2 part suffixes. Original download filenames stay unchanged. Unknown videos, ambiguous parts, or gaps pause downloading. Select movie files and arrange their merge order in task details, then confirm to continue.

Use **File delivery** and configure qB download, Moji download, Moji library, and Stash library roots. The Moji runtime requires `ffmpeg` and `ffprobe`. Choose Copy or Move for delivery.

Copy retains sources. Move stops the qBittorrent task and removes selected source files only after the entire movie and its subtitles have been published. Both preserve the original format for single files and produce MKV for merged parts, excluding extras from the library.

Preparation checks file sizes, removes byte-identical videos using SHA-256, then copies a single remaining movie unchanged and verifies its hash, without calling FFmpeg or ffprobe. Multiple distinct parts are checked with ffprobe, concatenated with FFmpeg, and checked for output duration. Incompatible parts remain blocked instead of being silently transcoded at lower quality. Part text subtitles are offset by preceding video durations and combined as SRT; styling may be simplified. Unsupported part bitmap subtitles block preparation and can be omitted by disabling subtitle retention before retrying.

Preparation runs in the background independently of web requests. Stash is notified only after the final movie directory is ready. File selections and ordering persist across restarts. Task-owned receipts and output hashes allow retries to reuse an already published movie without overwriting unrelated targets. Stash scan, generation, identification, and existing completion checks remain unchanged.

## Reselect torrent

For a wrong torrent or mistaken file confirmation, open **Task details > Actions > Reselect torrent** to start over. Confirmation deletes the old qBittorrent task and its downloaded files, delivered files and the delivery receipt, and the corresponding Stash scenes and generated assets. Files are deleted directly without backups. This happens regardless of the ordinary task deletion policy, including when that policy keeps downloaded files. Deletion cannot be undone.

The original Moji task entry and code remain. The old torrent, file selection, download progress, completion timestamps, and ingest references are cleared. Manual torrent selection opens only after all cleanup succeeds. Choosing a resource then starts file selection, downloading, and the full ingest workflow from the beginning.

Completed tasks and tasks actively delivering or processing in Stash cannot restart. Unowned or modified outputs and scenes linked to outside files stop cleanup. If cleanup is interrupted, select **Reselect torrent** again or retry to remove the remaining artifacts. A new download cannot start until cleanup finishes, and the old torrent is never selected automatically.
