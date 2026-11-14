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

## Subscriptions and Stash

The subscription summary shows the subscribed total and first checks still pending, with a management entry. Release checks show the schedule, progress, latest result and retained statistics separately. Reading Home does not trigger subscription checks.

The Stash panel shows connection status, version, scene count, and Moji tasks awaiting a Stash scan. It provides connection settings, a status refresh, and a link to open Stash. Unknown measurements appear as a dash; failed status reads retain the last observations without presenting the connection as healthy. Delivery paths remain in Stash settings.

Task processing starts automatically with Moji and has no global pause switch. Home hides global connection details, the global update time, and a page-wide refresh during normal operation. Service panels retain their own refresh actions. An interrupted connection shows a notice and retry action until recovery. Home refreshes snapshots every 30 seconds. Background tabs pause polling and refresh when visible again. A failed module leaves other modules usable and retains available data with a retry notice.
