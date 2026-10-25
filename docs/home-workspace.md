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
