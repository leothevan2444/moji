# Home workspace

Home prioritizes operator attention, task progress, and discovery. Detailed
service configuration is available from independent Indexers, Downloader and
Stash panels. Reading Home does not start checks, searches or rule updates.

## Data boundaries

`homeOverview` returns mutually exclusive counts for attention, active work,
and scheduled resource waits. Completed tasks are separate. Only tasks whose
recorded `ingestCompletedAt` falls within the previous 24 hours contribute to
`completedLast24Hours`; download completion and polling timestamps do not.

SQLite computes counts in an aggregate query and reads bounded task samples in
the same transaction: three oldest blocked tasks, five oldest active tasks, and
three most recently completed tasks with recorded ingest timestamps. The Home
query does not materialize the full task table. JSON and memory stores retain
compatible development behavior.

`homeSubscriptionSummary` counts performers currently subscribed in Stash. It
uses a narrow ID/custom-field membership query and local latest-check outcomes,
without loading full actor metadata, releases or StashBox catalogues. Each actor
counts once. Failed/interrupted latest checks are issues; an active check retains
the previous completed outcome. Historical task-creation retry failures are
separate. A failed membership read preserves prior UI values with an error.

The My subscriptions card keeps a large subscription total, a smaller latest-issue
count and the heading's management link. The issue count opens
`/performers?filter=error`; pending first checks appear when applicable. A neutral
History action sits at the right of the check-status row. Its left side shows
scheduling and the next relative deadline while idle, or actor/progress details
while active; narrow screens wrap that left content and retain the history entry.

`homeSubscriptionChecks` independently projects local run state: full active
count, at most three active and three recent summaries, the latest scheduled run
and at most three current actors per summary. The history drawer loads the
latest finished scheduled result, statistics and additional history on demand;
full outcomes use `subscriptionCheck(id)`. Counts and check state load and fail
independently inside the same card.

`latestFinishedScheduled` is separate from `latestScheduled`, which can be active.
`statistics` counts successful, failed (including interrupted), and skipped actor
attempts across up to 100 most recently started retained finished runs, including
manual and scheduled checks. Repeated actors count separately; active runs and
pending items are excluded. `runCount` reports the actual number of included runs.
This is a retained-history summary, not a complete time-window metric. The UI uses
these backend aggregates rather than the three recent-run samples. Empty history
has zero counts; a failed read retains the previous snapshot with its error notice.

## Homepage integration

The external Homepage dashboard reads the same four counts through
`GET /api/v1/widget/homepage`. Its counts-only query shares the aggregate with
Home, without loading task samples or contacting upstream services. Home still
reads its counts and bounded samples in one transaction. See
[Homepage widget setup](homepage-widget.md) for configuration.

## Navigation and updates

Check details use the `check` URL parameter and the same drawer and focus pattern
as task details. Their independent errors leave task modules usable. Home and
actor controls share visibility-aware polling, every two seconds while checks
are active and every ten seconds while idle. Actor controls query scoped
reservations rather than Home history.

Home task details use the `task` URL parameter, with `resolve=1` for manual source
selection and `files=1` for direct file confirmation. The drawer keeps Home
mounted. The task-center `workbench=1` entry
waits for its task snapshot before constructing a review session and pauses on
successful results until the user continues. Closing a Home-origin review
returns to Home. The completion metric links to the task list with
`status=completed&completedWithin=24h`.

Task events coalesce bounded Home refreshes. Service and subscription events
refresh their independent snapshots. Home also refreshes every 30 seconds while
visible and immediately upon returning to the tab. Normal connections have no status indicator, global timestamp, or refresh
control. Interruptions show a notice and retry action until reconnection succeeds.
Failed modules remain visible; available content is retained during refresh.
The global task bridge calibrates Home with its bounded query.

## Presentation

Desktop uses independent main and secondary columns; mobile uses one ordered
column. The interface supports Chinese, English, light/dark themes, keyboard
navigation, and reduced motion. Task downloads animate only between reported
progress values. Other stages show phase labels rather than simulated progress.
