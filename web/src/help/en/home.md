# Home

Home is the daily workspace: see what needs attention, follow task progress, or find your next movie.

## Search and overview

Choose **Find movies** (StashBox) or **Find resources** (Jackett). Submitting opens Discover with the query and source preserved. Press `/` to focus the search field.

Each metric opens its corresponding task list:

- **Needs attention**: blocked tasks requiring source selection, file confirmation, or recovery.
- **In progress**: sourcing, downloading, queued ingest, file preparation, scanning, and identification.
- **Waiting for resources**: tasks with another automatic resource search scheduled. These do not require intervention.
- **Added in last 24h**: tasks that passed the complete ingest workflow during the past 24 hours. The linked list uses the same completion window.

Downloading to 100% is not completion. File preparation and Stash scanning, generation, identification, and verification must finish successfully.

## Review tasks

Needs attention shows the three oldest blocked tasks with a direct next action. Open a release code to inspect its details in a drawer without losing your home position. Detail URLs can be shared.

Review queue opens the task center workbench for all attention items. Unsubmitted choices remain available while switching within the session. Successful actions show a result before you continue to the next item. Closing returns to Home.

In progress shows up to five tasks. Download percentages use real progress; other stages show their current step. Recently added shows the latest three tasks with recorded ingest completion times. Linked scenes can open in Stash in a new tab.

## Subscriptions and services

Subscription health comes from persisted check results. It includes subscribed performers, failed checks, first checks still pending, and links to affected performers. Reading Home does not trigger subscription checks.

Healthy services use a compact summary. Expand Connection & ingest details for paths and diagnostics. **Configured** means required fields exist; **Ready** means a recent probe succeeded. First-run guidance links to connection and delivery settings.

Paused automatic processing remains visible. Home hides connection details, the global update time, and manual refresh during normal operation. An interrupted connection shows a notice and retry action until recovery. Home refreshes snapshots every 30 seconds. Background tabs pause polling and refresh when visible again. A failed module leaves other modules usable and retains available data with a retry notice.
