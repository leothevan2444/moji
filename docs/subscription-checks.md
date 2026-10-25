# Subscription check runs

Release checks now have a persistent lifecycle distinct from the download tasks they may create. Both the scheduled worker and the Web UI use the same service, reservations, and history. Home shows the latest scheduled run, the worker's next deadline, active progress, and recent results, using the existing Home panel and drawer patterns.

`startSubscriptionCheck(ids: [ID!])` reserves and persists a run before returning its ID. Omit `ids` to check all subscriptions; an explicit empty list is rejected. Selected checks accept at most 100 distinct actors. Work runs under the application's lifetime with a 30-minute deadline, independently of the HTTP request. Closing or refreshing the browser does not cancel it.

`homeSubscriptionChecks` returns the schedule, full active count, at most three active and three recent run summaries, and the latest scheduled run. Each summary includes at most three currently checked actors. Its domain projection lives in `internal/subscription/home_checks.go`, its resolver in `home.resolvers.go`, and its frontend query in `home-page.graphql`. It reads local run state without upstream catalogue calls.

`subscriptionCheck(id: ID!)` loads full actor outcomes only when a Home drawer (`/?check=ID`) is opened. Actor controls use `subscriptionCheckActivity(ids: [ID!]!)`, which returns only reservations for visible and selected actors, the active count, and any all-actor reservation. It returns no history or full result records.

A run records its trigger (`MANUAL` or `SCHEDULED`), scope, timestamps, status, and individual actor outcomes. Fetching, persistence, and downstream release-processing failures are reflected in the result even if an older synchronous endpoint returned no top-level error. Zero actors is a successful empty check once enumeration completes.

Reservations cover the complete selected input for the lifetime of a run. An all-subscription run reserves every actor before it starts enumerating subscriptions. A repeated background request fully covered by an active run returns that existing run. Partially overlapping requests fail with `SUBSCRIPTION_CHECK_IN_PROGRESS`; disjoint selections can run concurrently. Older synchronous refresh mutations remain supported and participate in the same tracking and conflict checks.

SQLite retains the latest 100 runs, every active run, and the most recent scheduled run. Startup marks formerly running records and their unfinished actor items `INTERRUPTED`, with reason `SERVICE_RESTARTED`. Checks do not resume automatically after a restart; they can be triggered again or run at the next scheduled deadline. Ordinary shutdown and the overall deadline are also reported as interruption.

The scheduler reconciles the configured interval every five seconds, including while disabled. Interval changes reset the next deadline. A timer tick covered by an existing all-subscription check reuses it; partial overlap with a selected check defers that dispatch and retries within 30 seconds, with the retry deadline exposed in the overview. No second copy of the same actor check is queued.

Home and actor routes fetch their independent snapshots on mount and focus, every two seconds while checks are active, and every ten seconds while idle. Hidden tabs pause polling. Check buttons use the returned reservations; loading or failed status reads temporarily prevent a new submission. Active actor status links open that run on Home. Server-side reservations remain authoritative across tabs and concurrent requests. Existing performer subscription events continue to update actor results.
