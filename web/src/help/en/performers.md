# Performers

Performers are read from Stash. Moji adds its own subscription state without changing Stash favorites. A heart means the performer is a Stash favorite; a bookmark means Moji is monitoring that performer.

## Browse and subscribe

Search matches Stash performer names and aliases. Open a performer to see Stash details, the preferred StashBox match, subscription health, release counts, and a deduplicated scene list built from Stash and StashBox.

Subscribing records the performer in Moji. **Check now** refreshes one performer; **Refresh all** checks every subscription. Scheduled checks use the polling interval in **Settings > Automation**. A subscription needs a matching performer identity on a configured StashBox; errors remain visible on the performer.

## Review and queue scenes

The detail view can filter scenes by text, source, and library state. Scenes appearing in both sources are merged. **In library** means Moji matched the scene to Stash; an existing task is shown with its current stage.

Queue one scene or select eligible scenes on the current page for a batch. Items already in the library, already assigned a task, missing a usable code, or duplicated are skipped or failed individually; the batch result reports queued, skipped, and failed counts.

Subscription release policy classifies a release as solo, group, compilation-like, or unknown. **Download** creates a task, **Review** records it for review, and **Block** ignores it. Unknown dates and dates outside the automatic-download range are moved to review instead of downloaded automatically.
