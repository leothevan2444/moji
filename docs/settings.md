# Settings and configuration migration

Settings are grouped into **Indexers**, **Downloader**, **Stash**, **Tasks & subscriptions**, **System**, **Logs**, and **About**. Service connections are edited in their business tab. Existing `/settings/connections`, `/settings/ingest`, and `/settings/automation` bookmarks redirect to Indexers, Stash, and Tasks & subscriptions respectively.

Each form saves its own section. Unsaved drafts survive tab changes; a failed save retains the draft. Reopening settings reads fresh server data. Download speed limits remain qBittorrent settings and cleanup policies remain in Moji's core database.

## Configuration v2

The example is [docker/config.example.yaml](../docker/config.example.yaml). `config_version: 2` identifies this layout:

| Old field | New field |
| --- | --- |
| `connection.jackett` | `indexers.jackett` |
| `automation.torrent_selection.fast_rules.indexer_preference.tracker_ids` | `indexers.query_order` |
| `connection.qbittorrent` | `downloader.qbittorrent` |
| `connection.stash` | `stash.connection` |
| `ingest` | `stash.ingest` |
| `automation.selected_stash_box_endpoints` | `stash.metadata_priority` |
| `system.stash_box_data_cache` | `stash.data_cache` |
| Other `automation` fields | `tasks` |
| `system.task_delete_policy` | `tasks.task_delete_policy` |
| `system`, `logging` | Same ownership, except StashBox cache and deletion policy above |

Unknown top-level fields and fields within moved sections survive normal saves. Unmodeled fields directly inside old `connection` are retained under `legacy_connection`. Runtime Go provider fields and legacy GraphQL reads retain compatibility names; v2 is the persisted YAML format. Mixed v2/legacy fields and unsupported config versions are rejected. Startup reads legacy configuration without rewriting it; a successful settings save writes v2 atomically. Failed writes restore the last persisted state.

Do not let an old running binary save a v2 file. Stop it before replacing the actual configuration and start the new binary afterwards. To roll back the binary, restore the original configuration as well.

The offline tool never opens the databases or contacts external services:

```sh
go run ./cmd/moji-config -config config.yaml
go run ./cmd/moji-config -config config.yaml -output /tmp/moji-config-v2.yaml
```

The output path must be new and different from the source. The tool compares effective values after migration and does not print credentials. Keep a protected original, validate a copy, stop the old process, then replace the actual file with the validated copy. Configurations and backups contain secrets; do not commit them.

## Indexer order

Automatic sourcing queries one enabled Jackett site at a time in saved order. Other configured sites are appended in Jackett order; missing or disabled sites are skipped. Request-specific site filters restrict this eligible set. A site only stops the search after a candidate passes identity, hard rules, metadata verification and duplicate checks. Empty or rejected candidates, timeouts, rate limits and site errors continue to the next site. Cancellation stops the operation. Site requests have a 30-second timeout. If every request fails, the task records search failure instead of treating it as an empty successful search.

Manual searches collect results sequentially from eligible sites for user choice. They share request ordering, without stopping at an automatically chosen resource. Candidate preview applies whichever rule stages the user requests; actual manual submission enforces identity and duplicate checks without forcing automatic preferences.

## Resource effects

Title and internal filename clauses support:

| Effect | Meaning |
| --- | --- |
| `REQUIRE` | Every required clause must match. |
| `EXCLUDE` | Any matching exclusion rejects the candidate. |
| `PREFER` | Each match improves its rank within that rule. |
| `DEMOTE` | Each match worsens its rank within that rule. |

Exclusions take precedence over preferences. Fast rules compare candidates in configured order; later rules break ties. File inspection reranks the leading candidates within the configured budget. Hard file clauses reject failed, unavailable or uninspected candidates, including magnets that have no inspectable torrent file. A single-video preference remains a preference and does not forbid multipart movies. Invalid regular expressions are rejected when loading or saving configuration.

Disabling resource rules preserves upstream result order while retaining identity and duplicate checks. Old `AVOID` is normalized to `DEMOTE`; old filename `LOCK` becomes `REQUIRE`. This strengthens LOCK from a ranking preference to a mandatory condition. Indexer preference becomes actual sequential query order, so a later site's better-ranked resource no longer displaces an acceptable resource from an earlier site.

StashBox source priority is configured under Stash. Sources omitted from the priority list are still appended in Stash order; priority is not an enable/disable list.

## Language and task deletion

Task deletion policy is edited and saved independently under Tasks & subscriptions. Its YAML field is `tasks.task_delete_policy`; legacy `system.task_delete_policy` is read and moved on the next save, with mixed duplicate fields rejected. Compatibility GraphQL reads retain `settings.system.taskDeletePolicy`.

Interface language is edited under System and persisted as `system.language`: `auto` (default), `zh-CN`, or `en`. It applies after a successful save and loads on refresh/focus. Browser storage is only the initial display fallback while the saved configuration loads. Choosing `auto` follows each browser's language preferences. Failed saves preserve the draft and active language. System saves omit task deletion policy and preserve the current policy under the configuration lock.

## Compatibility names

Current UI components, query operations and translations use business names: service connections, media delivery, task/subscription settings, metadata sources and resource rules. Loading and failure states use these same current titles. The removed Connections, Ingest and Automation tabs have no active translation entries.

The following names are intentional compatibility contracts: legacy `/settings/connections`, `/settings/ingest` and `/settings/automation` redirects; old YAML fields consumed by the migration adapter; existing GraphQL `Settings.automation`, `Settings.ingest`, `Settings.system.taskDeletePolicy` and the deprecated `updateAutomationSettings` aggregate mutation and enum. New code uses `updateSettingsSection` with `SettingsSection`. These protocol names do not imply an old UI tab still exists. Task ingest and automatic processing remain real workflow concepts.
