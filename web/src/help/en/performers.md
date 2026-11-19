# Performers

Performers are read from Stash. Moji saves subscription markers in Stash custom fields without changing Stash favorites. A heart means the performer is a Stash favorite; a bookmark means Moji is monitoring that performer.

## Browse and subscribe

Search matches Stash performer names and aliases. Open a performer to see Stash details, the preferred StashBox match, subscription health, release counts, and a deduplicated scene list built from Stash and StashBox.

Filter the complete list by subscribed performers, Stash favorites, pending reviews, or subscribed performers with failed latest checks. Sort by name, scene count, or most recent check. Filters and sorting are saved in the URL. The pending-review filter also includes performers who are no longer subscribed.

Enter selection mode to select performers with checkboxes. Selections persist across pages, searches, and filters; the action bar reports selections outside the current page. Exiting selection mode clears them. Performer names, subscription buttons, and check buttons support independent keyboard operation.

Subscribing saves a marker in Stash; Moji retains check and release-processing records. **Check for updates** checks one performer; **Check all for updates** checks every subscription. Newly found scenes are handled according to subscription rules. Activity levels 1–5 use automatic check intervals of 30, 14, 7, 3, and 1 day; an unavailable rating uses 48 hours. Enable or disable automatic checks in **Settings > Tasks & subscriptions**; intervals are determined automatically. Saved deadlines survive restarts. Checking needs a matching performer identity on a configured StashBox. Expand check errors on a card or in the detail view to read the cause. Subscriptions that have never been checked are labeled separately.

**Activity level** shows five flame icons and a separate numeric activity value. **Checks** shows the next check, last check, and status. Open **About activity levels and values** for the detailed rules. The icons show the 1–5 level and the value uses two decimal places. All icons remain unlit until a reliable rating is available. Hover for the rating evidence or the reason it is unavailable. Retry a failed read with the icon beside the rating. A task count prefixed with ≥ covers loaded scenes only; its tooltip explains the limit. **Latest release** uses a valid release date from the loaded preferred StashBox catalog, never the time the release was discovered.

Filter works by All, Single or Multiple, together with source, library status and search. Single means one performer in the metadata; Multiple means two or more. Unknown cast counts appear only under All. This filter does not distinguish co-starring releases from compilations. Changing the filter returns to the first page, reloading retains the filter, and existing selections survive filter changes.

## Review and queue scenes

The detail view can filter scenes by text, source, and library state. Scenes appearing in both sources are merged. **In library** means Moji matched the scene to Stash; an existing task is shown with its current stage.

Queue one scene or select up to 100 eligible scenes across pages for a batch; submission includes every selected scene. Items already in the library, already assigned a task, missing a usable code, or duplicated are skipped or failed individually; the batch result reports queued, skipped, and failed counts.

Subscription release policy classifies a release as solo, group, compilation-like, or unknown. **Download** creates a task, **Review** records it for review, and **Block** ignores it. Unknown dates and dates outside the automatic-download range are moved to review instead of downloaded automatically.

## Release review

Open Release review from the Performers page, or use a performer’s review count to filter by that performer. Filter by performer, code or title, VR and status. Selection persists across pages, up to 100 releases.

Only Download release creates a task. Do not download saves a persistent decision; Reconsider explicitly returns rejected or policy-skipped releases to review. Batch results are reported individually and failed or unchanged selections are retained. Task creation failures have their own view and retry action. Existing tasks and library items are not duplicated.

Performer details load basic information directly from Stash, independently of scene statistics and the scene list. Existing StashBox data appears immediately while expired snapshots are refreshed in the background.

Unsubscribed performers can be checked manually after confirmation. A manual check does not enable automatic checks. Scheduled runs only admit performers who are subscribed when the run is queued; admitted performers finish their current check even if they are later unsubscribed.

The activity value uses release dates from the past year, with contributions halving every 60 days. Each release day contributes at most 1; explicit compilations contribute 0.2 and insufficient type metadata contributes 0.5. Five flame icons indicate the activity level; all remain unlit while no rating is available. Queues or failures can delay the estimated next check.

Unavailable ratings distinguish loading, unassessed, waiting or active assessment, incomplete release coverage, invalid dates, unverified ownership, and unsuccessful checks. Hover over the value or icons for an explanation of the StashBox evidence.
