# Performers

Performers are read from Stash. Moji adds its own subscription state without changing Stash favorites. A heart means the performer is a Stash favorite; a bookmark means Moji is monitoring that performer.

## Browse and subscribe

Search matches Stash performer names and aliases. Open a performer to see Stash details, the preferred StashBox match, subscription health, release counts, and a deduplicated scene list built from Stash and StashBox.

Filter the complete list by subscribed performers, Stash favorites, pending reviews, or failed checks. Sort by name, scene count, or most recent check. Filters and sorting are saved in the URL. The pending-review filter also includes performers who are no longer subscribed.

Enter selection mode to select performers with checkboxes. Selections persist across pages, searches, and filters; the action bar reports selections outside the current page. Exiting selection mode clears them. Performer names, subscription buttons, and check buttons support independent keyboard operation.

Subscribing records the performer in Moji. **Check now** refreshes one performer; **Check all** checks every subscription. Scheduled checks use the polling interval in **Settings > Automation**. A subscription needs a matching performer identity on a configured StashBox. Expand check errors on a card or in the detail view to read the cause. Subscriptions that have never been checked are labeled separately.

## Review and queue scenes

The detail view can filter scenes by text, source, and library state. Scenes appearing in both sources are merged. **In library** means Moji matched the scene to Stash; an existing task is shown with its current stage.

Queue one scene or select eligible scenes on the current page for a batch. Items already in the library, already assigned a task, missing a usable code, or duplicated are skipped or failed individually; the batch result reports queued, skipped, and failed counts.

Subscription release policy classifies a release as solo, group, compilation-like, or unknown. **Download** creates a task, **Review** records it for review, and **Block** ignores it. Unknown dates and dates outside the automatic-download range are moved to review instead of downloaded automatically.

## Release review

Open Release review from the Performers page, or use a performer’s review count to filter by that performer. Filter by performer, code or title, VR and status. Selection persists across pages, up to 100 releases.

Only Download release creates a task. Do not download saves a persistent decision; Reconsider explicitly returns rejected or policy-skipped releases to review. Batch results are reported individually and failed or unchanged selections are retained. Task creation failures have their own view and retry action. Existing tasks and library items are not duplicated.
