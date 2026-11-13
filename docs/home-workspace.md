# Home workspace

Home prioritizes operator attention, task progress, and discovery. Detailed
service configuration remains available under an expandable status panel.

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

`homeSubscriptionSummary` separately reads persisted subscription check health.
Its counts and up to three affected performer IDs require no Stash catalogue
or release-history read. A failed summary does not hide the rest of Home.

`homeSubscriptionChecks` is a separate bounded local projection of release-check
runs: the full active count, three active and three recent run summaries, and the
latest scheduled run. Each run includes at most three current actor names.
`internal/subscription/home_checks.go` builds this projection independently of
check execution; `home.resolvers.go` only maps it to the GraphQL contract.
Full actor outcomes are fetched through `subscriptionCheck(id)` on demand.

Home retains the large subscription total and the heading's management link,
without showing subscription issues. First-check counts remain visible.
The check panel shows automatic scheduling and the next deadline; its History
button uses a success or warning icon and color for the latest retained finished
scheduled run. Its title and accessible name include the result and timestamp.
An active run does not replace that result. With no finished scheduled run, the
button stays neutral. Active runs retain progress and actor names. Recent runs
are available in the History drawer rather than repeated on Home.
On narrow screens, release checks follow the subscription summary.

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
