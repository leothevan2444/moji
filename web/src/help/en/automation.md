# Automation

Automation controls background polling, StashBox priority, subscription decisions, and automatic torrent ranking. Changes affect subsequent operations.

## Scheduling and release policy

The task progress interval controls qBittorrent synchronization; the subscription interval controls scheduled performer checks. Zero uses the built-in default of 60 seconds or 1 hour respectively; a negative value disables the corresponding loop.

Release policy separately controls solo, group, and compilation-like scenes. **Download** creates a task, **Review** records the release without downloading it, and **Block** ignores it. Compilation signals take priority over performer-count classification. A missing date or date outside the configured automatic range changes an otherwise automatic download to review.

## Sources and torrent rules

StashBox endpoints are loaded from Stash. Discovery stops at the first endpoint with results; subscription lookups try endpoints in saved order. Endpoints omitted from the custom order are appended in Stash order.

Torrent selection is a lexicographic rule chain: the first rule that distinguishes candidates wins, and later rules break ties. Fast rules use indexer, title, date, similarity, seeders, or size. File rules inspect only the configured number of leading candidates and can prefer a single video or internal filename patterns. `PREFER` moves matches up, `AVOID` moves them down, and filename `LOCK` supplies the strongest preference. Invalid regular expressions do not match.
