# Task workspace snapshots and live updates

The task route loads a server-filtered, sorted page of `TaskSummary` records
(24 by default; the UI offers 12/24/48, and the API caps requests at 100).
Files, candidate data, transfer paths and execution details load only when a task
is opened. No virtual scrolling is used.

`TaskSummary` and full `Task` records have separate cache entities. The summary
includes a durable revision, a sortable version combining revision and update
time, lifecycle fields and progress. Error messages are capped at 512 runes;
the detail retains the full diagnostic. A single Stash scene ID preserves the
Home page's recent-ingest link.

## Backend reads

- The default created-time page uses SQL `LIMIT/OFFSET` with an ID tie-breaker.
  Counts and summaries do not load attachments, file plans or checkpoint JSON.
- Progress/name observations are maintained in memory separately from durable
  task data. Search and progress/updated-time sorts therefore filter and sort
  lightweight server-side header projections with the live observation overlay,
  then return a page. These operations still scan matching headers; they do not
  deserialize full task histories. This preserves freshness without flushing
  observations to SQLite for each UI read.
- Home reads three attention summaries, five active summaries and three recent
  completions, plus aggregate counts.
- The workbench starts an independent, oldest-first attention session from IDs
  and small identity/status records. It fetches one task's full detail at a time.
  New arrivals are left for the next session. Its session ID list scales with
  the number of attention tasks, independently of the visible list page/filter.
- Cross-page selection stores up to 100 IDs independently of visible rows.
  Selected summaries are queried separately, so retry eligibility remains valid
  after changing pages or filters. Selection is never pruned against one page.

## Snapshot and subscription boundaries

The eventing store serializes page, Home, detail and workbench snapshot boundaries
with commits and publication. Each snapshot carries `streamId` and `sequence`.
The stream ID changes on process restart.

`taskEvents` first sends `READY`, registered atomically with the event bus. Normal
events include a small summary and counts; full task data remains available to
legacy subscribers but is not materialized for summary-only subscriptions.
`READY` has no counts, so it cannot overwrite an already loaded aggregate with
zeroes. Event publication is serialized so concurrent publishers cannot deliver
sequence numbers out of order.

The browser checks sequence gaps before batching, coalesces updates for each task
over 100ms, then delivers the surviving events in sequence order. The per-task
coalescer preserves membership-change signals. Thus one task's progress does not
cause all task details or list pages to be fetched again.

A bounded 512-task transport journal overlays newer versions onto late HTTP
summary responses, protects counts, and removes known deleted rows. Snapshots
older than the journal floor require a bounded reread. Durable revision changes
reload only the open detail after any in-flight request finishes; progress patches
leave full diagnostic text and files intact.

Only mounted consumers recover their own page, selected IDs, workbench or detail.
The global bridge never fetches task history. A snapshot already at the ready
boundary avoids a duplicate initial read. If a slow protocol subscriber overflows,
the server replaces its backlog with `READY`; losing the final event therefore
does not leave a client stale indefinitely while waiting for another event.

## List stability and explicit actions

Ordinary progress changes update cards without moving them. Membership changes
are refreshed after a short coalescing delay when the created-time list is idle.
During selection, detail/workbench interaction, or progress/updated-time ordering,
the page exposes an explicit refresh action. URL search, filter, sort and page
state survive opening and closing details.

Explicit retry/delete/resolve actions also refresh the affected page/selection,
so their confirmed result is visible even when the socket is disconnected.
The manual refresh action remains available. The list omits continuously animated
progress sheen while retaining transitions for actual progress changes.

## Validation

Backend tests cover bounded header reads even with unsupported cold checkpoint
payloads, current observations in search/sorting, counts, page clamping, snapshot
boundaries, missing/deleted details, independent queues and overflow recovery.
Frontend tests cover late-response protection, batching order, detail revisions,
and preservation of full files/diagnostics during progress updates. Browser tests
use 277 synthetic tasks to exercise pagination, cross-page selection, stable
updates, delayed snapshots, stream restart, details and workbench navigation.
