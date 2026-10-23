# Automation

Automation controls background polling, StashBox priority, subscription decisions, and automatic torrent ranking. Changes affect subsequent operations.

## Scheduling and release policy

The task processing switch controls download progress synchronization and automatic ingest. Task check intervals are 30 seconds, 1 minute, 2 minutes or 5 minutes. Subscription checks run independently: zero uses the default 1-hour interval, and a negative value disables scheduled subscription checks. Manual checks remain available.

The new-release download policy applies to scheduled subscription checks and manual checks of one, selected or all performers, including historical releases found during the first check. Subscribing only marks a performer as subscribed. Direct downloads from performer lists or search results, manual task creation and existing task retries do not apply this policy.

- **Download** allows a download task to be created; resource search follows the normal task workflow.
- **Manual review** has the same behavior as the previous “Record for review” option: save a release record without creating a task or submitting a download. Use Release review on the Performers page to approve or reject downloads.
- **Skip** records the skipped decision without creating a task or submitting a download.

Compilation classification takes priority over performer count. Other releases use the single-performer or multiple-performer setting. All non-compilation releases with two or more performers share one setting; the previous performer-count threshold did not affect download decisions and has been removed from the form.

**A release is VR only when its code contains `VR`, ignoring case.** Titles, tags, studios, torrent titles and filenames are not used for VR detection. VR defaults to manual review. The VR and classification policies apply together: skip takes priority over manual review, and automatic download requires every applicable policy to allow it. Allowing VR downloads cannot override a compilation skip policy.

The automatic download date range applies only to single- and multiple-performer releases that would otherwise download. Older releases or unknown dates are recorded for review. Compilations are exempt from this date range. Uncertain classifications are also recorded for review.

Saving settings affects newly discovered releases without reprocessing existing records. Records without tasks currently include releases awaiting review, policy skips and failed task creation; this total is not a review count. Durable review records are independent of performer snapshot limits and survive unsubscribing.

## Sources and torrent rules

StashBox endpoints are loaded from Stash. Discovery stops at the first endpoint with results; subscription lookups try endpoints in saved order. Endpoints omitted from the custom order are appended in Stash order.

Torrent selection is a lexicographic rule chain: the first rule that distinguishes candidates wins, and later rules break ties. Fast rules use indexer, title, date, similarity, seeders, or size. File rules inspect only the configured number of leading candidates and can prefer a single video or internal filename patterns. `PREFER` moves matches up, `AVOID` moves them down, and filename `LOCK` supplies the strongest preference. Invalid regular expressions do not match.
