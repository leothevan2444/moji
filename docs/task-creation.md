# Task creation

Actor-page creation resolves only the selected Stash-box scene IDs. Each scene
must belong to a Stash-box performer mapped to the actor and a configured
endpoint. This path does not paginate the actor's catalogue or register images.
The validated code and title are passed directly to taskflow, avoiding a second
lookup of the same scene. Local Stash selections remain skipped.

Code-based taskflow entrypoints persist a `SOURCING/PENDING` task and return it
before library checks, torrent search, inspection, or download submission.
Creation events are published immediately. Invalid codes and duplicate task
codes are still rejected before creation; downstream failures are reflected in
the persisted task rather than delaying the creation response.

The application starts two sourcing workers with its lifecycle context. The
queue is the task store, not an in-memory request list. SQLite schema 13 adds
`sourcing_request_json` without deleting existing tasks or events. Saved request
options include tracker filters, categories, limits, destination, tags and pause
preference. Workers scan on startup, creation notifications and a recovery timer.
Request cancellation does not cancel worker execution; application cancellation
leaves unfinished sourcing jobs available for the next startup. Blocked tasks
require user retry and are not automatically retried in a loop.

A validated torrent selection is checkpointed before submission. After an
interruption, the worker revalidates and resubmits the same selected torrent
instead of performing a fresh search. qBittorrent deduplicates identical torrent
submissions. The task enters `DOWNLOADING` only after successful submission.

The low-level synchronous `DownloadMediaContext` remains available for execution
and existing integrations; interactive code-based creation uses
`QueueDownloadContext`. Manual torrent creation keeps its existing path.

Tests cover immediate creation events with a blocked search, request cancellation,
request-option persistence, worker concurrency, failures and retry, selected-scene
membership checks, recovery after submission interruption, and schema upgrades.
