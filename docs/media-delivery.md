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

The production clean-delivery path requires TRANSFER mode, COPY or MOVE, configured
Moji filesystem roots, and `ffmpeg`/`ffprobe` in PATH. Path mapping and symlinks
cannot produce a newly assembled video. The settings UI retains legacy values
for visibility but offers the supported modes for new selections.

The persisted preparation queue uses `TRANSFERRING/PENDING`. At most one task
holds the processing slot. Active processing uses `TRANSFERRING/RUNNING` and a
lifecycle context independent of the HTTP request and progress polling timeout.
A process shutdown cancels tools and leaves a recoverable checkpoint. Deleting a
task cancels its preparation. A failed preparation stays blocked before Stash.

The preparer validates source containment and declared video sizes, hashes files
to remove byte-identical videos, checks stream compatibility, and losslessly
remuxes/concatenates into `CODE/CODE.mkv`. It does not silently re-encode incompatible
parts. It validates output duration before publication. With subtitles disabled,
embedded subtitle streams are excluded as well. Selected part text subtitles are
normalized to SRT and shifted using preceding video durations; whole-movie
subtitles retain their format. Conflicting tracks or unsupported part bitmap
subtitles block publication instead of producing misleading captions.

Work stays in a private temporary directory with a `.stashignore` file. After
all artifacts are ready, the movie directory is renamed into place on the same
filesystem. Only the final directory is submitted to Stash. A separate
`.moji-receipts` directory contains task-owned signatures and output hashes for
recovery; these bookkeeping files are not in the submitted movie directory.
An existing unrelated or modified target is never reused or overwritten.

COPY retains downloaded files. MOVE first stops qBittorrent, publishes the full
artifact set, and then removes only selected sources; recovery tolerates already
removed sources. Completed delivery checkpoints proceed through the existing
Stash scan/generate/identify workflow without repeating preparation.

## Validation

Media integration tests create real FFmpeg clips and check ordered assembly,
byte-level deduplication, subtitle offsets, subtitle exclusion, cancellation,
output ownership, and refusal to reuse modified outputs. Runtime tests cover
file priorities, manual ordering, persisted plans, independent background work,
bounded preparation, failures, retries, and shutdown recovery. GraphQL and UI
tests cover the subtitle switch and ordered file selection.

Run `make test-media` with FFmpeg installed to require and execute the media tests.
