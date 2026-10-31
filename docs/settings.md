# Settings and configuration migration

Settings are grouped into **Indexers**, **Downloader**, **Stash**, **Tasks & subscriptions**, **System**, **Logs**, and **About**. Service connections are edited in their business tab. Existing `/settings/connections`, `/settings/ingest`, and `/settings/automation` bookmarks redirect to Indexers, Stash, and Tasks & subscriptions respectively.

Settings initialize from fresh GraphQL queries and save automatically after 500 ms without further edits. Only changed fields are sent through the typed `patchSettings` mutation. Objects merge recursively; lists replace. Pending edits survive tab changes and edits made during a request are submitted afterwards. A failed save retains pending edits and offers retry. Reopening settings reads fresh server data. Download speed limits remain qBittorrent settings and cleanup policies remain in Moji's core database.

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
| `LOCK` | Lock on any match (OR across title and file clauses); no matches leave normal selection unchanged. |
| `EXCLUDE` | Any matching exclusion rejects the candidate. |
| `PREFER` | Each match improves its rank within that rule. |
| `DEMOTE` | Each match worsens its rank within that rule. |

Exclusions take precedence over locks and preferences. After exclusions, if any candidate matches a lock, selection is restricted to the matching candidates, with the other rules deciding among them. Identity, duplicate and torrent validation remain mandatory. Fast rules compare candidates in configured order; later rules break ties. File inspection reranks the leading candidates within the configured budget. File exclusions reject failed, unavailable or uninspected candidates, including magnets that have no inspectable torrent file. File locks only match successfully inspected paths; inspection failure or no match does not prevent normal fallback. A single-video preference remains a preference and does not forbid multipart movies. Invalid regular expressions are rejected when loading or saving configuration.

Disabling resource rules preserves upstream result order while retaining identity and duplicate checks. Old `AVOID` is normalized to `DEMOTE`. Code, GraphQL and canonical configuration use `LOCK` for lock on match. Indexer preference becomes actual sequential query order, so a later site's better-ranked resource no longer displaces an acceptable resource from an earlier site.

StashBox source priority is configured under Stash. Sources omitted from the priority list are still appended in Stash order; priority is not an enable/disable list.

## Language and task deletion

Task deletion policy is edited and saved independently under Tasks & subscriptions. Its YAML field is `tasks.task_delete_policy`; legacy `system.task_delete_policy` is read and moved on the next save, with mixed duplicate fields rejected. Compatibility GraphQL reads retain `settings.system.taskDeletePolicy`.

Interface language is edited under System and persisted as `system.language`: `auto` (default), `zh-CN`, or `en`. It applies after a successful automatic save and loads on refresh/focus. Browser storage is only the initial display fallback while the saved configuration loads. Choosing `auto` follows each browser's language preferences. Failed saves preserve the draft and active language. System saves omit task deletion policy and preserve the current policy under the configuration lock.

## Compatibility names

Current UI components, query operations and translations use business names: service connections, media delivery, task/subscription settings, metadata sources and resource rules. Loading and failure states use these same current titles. The removed Connections, Ingest and Automation tabs have no active translation entries.

The following names are intentional compatibility contracts: legacy `/settings/connections`, `/settings/ingest` and `/settings/automation` redirects; old YAML fields consumed by the migration adapter; existing GraphQL `Settings.automation`, `Settings.ingest`, `Settings.system.taskDeletePolicy` and the deprecated `updateAutomationSettings` aggregate mutation and enum. New code uses `updateSettingsSection` with `SettingsSection`. These protocol names do not imply an old UI tab still exists. Task ingest and automatic processing remain real workflow concepts.


## Incremental automatic saving

`patchSettings(input: SettingsPatchInput!)` accepts optional fields grouped by the canonical v2 ownership (`indexers`, `downloader`, `stash`, `tasks`, `system`). Omitted or null fields remain unchanged; explicit empty strings clear strings, empty lists replace lists with an empty list, and `false` remains an explicit value. Nested objects merge; lists are atomic ordered values. The previous full-section mutations remain available for compatibility.

The settings provider retains each form's local value, acknowledged baseline and pending delta across tab changes. A 500 ms quiet period merges rapid edits. Requests for the same form are serialized; an acknowledgement clears only the submitted changes, so edits made during a slow request remain pending. Saving failures retain pending state and offer retry without a request loop. Queues continue when navigating within the application; closing or reloading the browser while work is pending invokes the browser's unsaved-change prompt. Pending values and credentials are not copied into browser storage.

The Go configuration store locks the full read/merge/validate/write operation, using the latest saved document and preserving unrelated and unknown fields. Validation failures leave the current document unchanged; persistence failures are surfaced to the caller. Configuration values become visible in memory after durable writing succeeds. As with existing atomic file writes, a directory-sync error after rename is reported as uncertain durability; retry is safe because patches set values rather than replaying external actions.

Downloader cleanup uses a separate partial policy mutation and serializes read/merge/write under its policy lock. Speed-limit edits use the existing single-field qBittorrent command. Cache clearing, source refresh and other explicit operations remain buttons; they are not configuration saves.

## Log files and live viewer

Logs are always stored beside the configuration in `logs/moji.log`; no file
path setting is required or supported. The bootstrap logger, configured logger
and current-file download use the same fixed location. In Docker this is
`/data/logs/moji.log` inside the persistent mount. The retired
`logging.file_path` field is ignored when reading old configurations and removed
on the next successful save. Old log files and backups are not moved or deleted
automatically; archive them separately when upgrading.

The active file appends across restarts and rotates by size (default 10 MiB).
Backups stay in the same directory as `moji.log.1` through `moji.log.5` by default;
`.1` is the newest backup. `logging.max_file_size_bytes` and
`logging.max_file_backups` control these limits. A single log entry is not split
across files. Retention applies to this file set, not files left at an old path.

The logs tab shows up to 200 recent matching in-memory entries, oldest first,
and follows new entries at the bottom. Scrolling upward or choosing Pause
following freezes the visible list while the bounded live cache keeps updating.
Resume following returns to the latest list. Changing the minimum level resets
following; reconnects and sequence gaps refresh the snapshot. The viewer is not
a reader for rotated files or logs from previous processes. Copy uses the visible
list in display order; Download retrieves only the active file, with all recorded
levels, via `/api/logs/current`. Rotated backups can be read from the log directory.
