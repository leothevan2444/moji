# Clean movie delivery

Moji owns download file selection and the delivery artifact. Stash owns library
metadata, identification, generated playback resources, and the existing task
completion checks.

## File plan

`internal/mediafiles` classifies torrent entries into four kinds: `MAIN`,
`SUBTITLE`, `DISCARD`, and `UNKNOWN_VIDEO`. A task's trusted release code anchors
matching, including advertising-domain removal, case/separator variants,
zero-padding aliases, and part suffixes. The trusted spelling remains the output
name; aliases do not globally rewrite release identifiers.

`tasks.file_plan_json` stores original qB file indices, original relative paths,
sizes, classification reasons, subtitle policy, ordered parts, manual decisions,
and priority-application state. Schema version 9 upgrades versions 7 and 8
additively. File selection and ordering survive process restarts.

Production construction enables file preparation in `configureTaskRuntime`.
qBittorrent receives `stopCondition=MetadataReceived`; non-magnet submissions are
also stopped on add. Once metadata exists, Moji stops the torrent, applies file
priorities using original qB indices, and starts only the selected download.
Unknown videos or ambiguous/missing part order remain paused for manual review.
The optional initial paused flag is respected. Download completion is based on
selected file progress, never completion of excluded extras.

`resolveTaskFiles` accepts video indices in playback order for a blocked download.
It validates against the server-owned file plan and cannot select arbitrary
paths. The subtitle toggle controls every language; subtitles explicitly naming
another release are excluded. No subtitle language detector is used.

## Preparation and publication

Ingest always prepares a movie and delivers it using COPY or MOVE. Configure all
four roots: qB download, Moji download, Moji library, and Stash library. Single
movies are published as `CODE/CODE.ext`, retaining the source extension
and bytes without requiring media tools. Multiple distinct parts require `ffmpeg`
and `ffprobe` in PATH and are published as `CODE/CODE.mkv`. Stash scans the
corresponding `Stash library root/CODE` directory.

The persisted preparation queue uses `TRANSFERRING/PENDING`. At most one task
holds the processing slot. Active processing uses `TRANSFERRING/RUNNING` and a
lifecycle context independent of the HTTP request and progress polling timeout.
A process shutdown cancels tools and leaves a recoverable checkpoint. Deleting a
task cancels its preparation. A failed preparation stays blocked before Stash.

The preparer validates source containment and declared video sizes and hashes
files to remove byte-identical videos. If one distinct movie remains, it copies
the file unchanged and checks the copy against the source hash. Embedded tracks,
metadata, and chapters remain intact, regardless of the subtitle setting. No
`ffprobe` or `ffmpeg` call is made for single-file delivery, including retained
external subtitles. This verifies file integrity, not media decodability.

For multiple distinct parts, it probes stream compatibility, losslessly
concatenates into `CODE/CODE.mkv`, and checks output duration before publication.
It does not silently re-encode incompatible parts. During merging, embedded
subtitle streams follow the subtitle retention setting. Selected part text
subtitles are normalized to SRT and shifted using preceding video durations; whole-movie
subtitles retain their format. Conflicting tracks or unsupported part bitmap
subtitles block publication instead of producing misleading captions.

Work stays in a private temporary directory with a `.stashignore` file. After
all artifacts are ready, the movie directory is renamed into place on the same
filesystem. Only the final directory is submitted to Stash. A separate
`.moji-receipts` directory contains task-owned signatures and output hashes for
recovery, including the output video filename; legacy receipts without that
filename still recover their MKV output. These bookkeeping files are not in the
submitted movie directory.
An existing unrelated or modified target is never reused or overwritten.

COPY retains downloaded files. MOVE first stops qBittorrent, publishes the full
artifact set, and then removes only selected sources; recovery tolerates already
removed sources. Completed delivery checkpoints proceed through the existing
Stash scan/generate/identify workflow without repeating preparation.

## Validation

Tool-independent tests cover original-format single-file COPY/MOVE, subtitles,
deduplication to one file, size checks, cancellation, and legacy receipt recovery.
Media integration tests create real FFmpeg clips and check ordered assembly,
byte-level deduplication, subtitle offsets, subtitle exclusion, cancellation,
output ownership, and refusal to reuse modified outputs. Runtime tests cover
file priorities, manual ordering, persisted plans, independent background work,
bounded preparation, failures, retries, and shutdown recovery. GraphQL and UI
tests cover the subtitle switch and ordered file selection.

Run `make test-media` with FFmpeg installed to require and execute the media tests.
