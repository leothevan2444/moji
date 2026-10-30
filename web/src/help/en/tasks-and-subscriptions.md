# Tasks and subscriptions

Tasks & subscriptions controls background scheduling, release decisions, and resource selection. StashBox priority and data cache are under Stash. Changes affect subsequent operations.

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

StashBox endpoints and their priority are managed under Settings > Stash. Discovery stops at the first endpoint with results; subscription lookups try endpoints in saved order. Endpoints omitted from the custom order are appended in Stash order.

Automatic sourcing queries enabled Jackett sites sequentially in the order saved under Indexers. It continues after empty, rejected, timed-out or failed sites, and stops only after a verified eligible resource. Other enabled sites are appended. Manual search collects results sequentially for user choice.

Resource clauses support **Lock on match (LOCK)**, **Exclude**, **Prefer**, and **Demote**. Any title or file path lock match restricts selection to eligible matching torrents, ranked by the other rules. With no lock matches, selection continues normally. Any exclusion rejects the candidate; locks cannot bypass exclusions, identity, duplicate or torrent validation. Preferences and demotions adjust rank, with later rules breaking ties. File checks apply only within the inspection budget. Failed or unchecked files cannot match file locks but remain eligible for normal fallback. With file exclusions enabled, candidates without successful inspection are still rejected. Disabling rules keeps upstream order and still enforces identity and duplicate checks. Invalid regular expressions cannot be saved. Code, configuration and API use LOCK. AVOID becomes DEMOTE. Manual selection retains identity checks without forcing automatic preferences.

## Task deletion

The policy can delete only the Moji task record, also remove the corresponding qBittorrent item, or remove both the torrent item and its downloaded files. It is evaluated when a task is deleted and does not apply retroactively. The delete confirmation shows the current effect.
