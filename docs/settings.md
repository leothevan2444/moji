# Settings and configuration

Settings are grouped into **Indexers**, **Downloader**, **Stash**, **Tasks & subscriptions**, **System**, **Logs**, and **About**. Service connections are edited in their business tab. Existing `/settings/connections`, `/settings/ingest`, and `/settings/automation` bookmarks redirect to Indexers, Stash, and Tasks & subscriptions respectively.

Settings initialize from fresh GraphQL queries and save automatically after 500 ms without further edits. Changed fields are sent through typed settings mutations. Objects merge recursively; lists replace. Pending edits survive tab changes and edits made during a request are submitted afterwards. A failed save retains pending edits and offers retry. Reopening settings reads fresh server data.

## Where settings are saved

| Settings or data | Storage |
| --- | --- |
| Indexer connections, sites, ordering, network, cache policy and definition update policy | `config.yaml`, under `indexers` |
| Moji downloader connection/defaults and automatic cleanup policy | `config.yaml`, under `downloader.qbittorrent` and `downloader.cleanup` |
| Stash connection, delivery paths, metadata priority and data cache policy | `config.yaml`, under `stash` |
| Task and subscription policies, deletion policy, system preferences and logging configuration | `config.yaml`, under `tasks`, `system` and `logging` |
| qBittorrent speed limits, seeding limits and torrent queueing | qBittorrent itself |
| Task choices and accepted source snapshots, execution/recovery state, observations and cached data | Moji databases |
| Theme, search history and remembered search controls | Browser storage |

Interface language is saved in `system.language`; its browser copy is a display fallback. Database records containing a task's accepted configuration preserve that task's execution and recovery context; changing global settings does not turn these records into global configuration.

Automatic cleanup has no database configuration fallback or automatic configuration conversion. See [Downloader](downloader.md#自动清理) for its YAML fields and when changes take effect. The configuration storage audit on 2026-11-13 found no additional application settings stored in databases that require relocation.

## Current configuration

The example is [docker/config.example.yaml](../docker/config.example.yaml). `config_version: 4` is the current layout, grouped by the business ownership listed above. Unsupported versions, mixed ownership layouts and retired fields are rejected. Settings saves validate and atomically persist the current configuration while preserving unrelated fields; failed saves retain the previous effective settings.

## Resource selection and indexer order

See [Resource selection](resource-selection.md) for the reusable rule library, rule groups, preview, search budgets and YAML. Automatic sourcing searches sites in saved order, retains candidates across sites, stops on a verified first-tier resource, and otherwise chooses the best acceptable resource at the end of the round. Interactive searches collect site results for manual choice. Rule groups never bypass identity or duplicate checks.

StashBox source priority is configured under Stash. Sources omitted from the priority list are still appended in Stash order; priority is not an enable/disable list.

## Language and task deletion

Task deletion policy is edited and saved independently under Tasks & subscriptions. Its YAML field is `tasks.task_delete_policy`; legacy `system.task_delete_policy` is read and moved on the next save, with mixed duplicate fields rejected. Compatibility GraphQL reads retain `settings.system.taskDeletePolicy`.

Interface language is edited under System and persisted as `system.language`: `auto` (default), `zh-CN`, or `en`. It applies after a successful automatic save and loads on refresh/focus. Browser storage is only the initial display fallback while the saved configuration loads. Choosing `auto` follows each browser's language preferences. Failed saves preserve the draft and active language. System saves omit task deletion policy and preserve the current policy under the configuration lock.

## Compatibility names

Current UI components, query operations and translations use business names: service connections, media delivery, task/subscription settings, metadata sources and resource rules. Loading and failure states use these same current titles. The removed Connections, Ingest and Automation tabs have no active translation entries.

The following names are intentional compatibility contracts: legacy `/settings/connections`, `/settings/ingest` and `/settings/automation` redirects; old YAML fields consumed by the migration adapter; existing GraphQL `Settings.automation`, `Settings.ingest`, `Settings.system.taskDeletePolicy` and the deprecated `updateAutomationSettings` aggregate mutation and enum. New code uses `updateSettingsSection` with `SettingsSection`. These protocol names do not imply an old UI tab still exists. Task ingest and automatic processing remain real workflow concepts. Task processing starts automatically with Moji and cannot be disabled through settings. Task processing frequency has three tiers; subscribed-performer checks always run automatically on their independent activity-based schedules.


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

## Shared builtin indexer network

`indexers.builtin.network.proxy` contains `enabled`, `url`, `username`, and `password`. `indexers.builtin.network.flaresolverr` contains `enabled`, `url`, and `max_timeout_ms` (default 90000, range 1000–300000). This is the sole FlareSolverr timeout: it bounds queueing and the full call, with remaining milliseconds sent as protocol `maxTimeout`; a shorter operation deadline still applies. Both apply to all builtin sites; they do not configure the external Jackett service. Disabling retains saved values; password edits are write-only, blank retains the saved password, and clearing is explicit. Request and site timeouts remain shared values. Request timeout bounds one built-in HTTP request or Moji request to Jackett (connection, redirects and response reading). Site timeout bounds one complete site search, including login, pagination and solver use, and also bounds shared torrent acquisition. These settings do not change Jackett’s own timeouts. The network form uses the existing settings autosave queue: proxy, solver and timeout edits are persisted through one atomic updateIndexerNetwork patch. There is no manual save button. Empty usernames clear the saved username; empty passwords are omitted from patches and retain the saved password.

The separate `challenge_timeout_seconds` field has been removed; configure `network.flaresolverr.max_timeout_ms` instead. Removed keys are rejected, with no automatic conversion.

The profile registry, per-site `network_profile`, and profile CRUD API have been removed. There is no automatic profile migration. Replace these removed keys manually before loading an existing configuration, retaining a private backup. The schema rejects the removed keys instead of silently ignoring them. Existing accepted task resource records retain their frozen transport parameters.

## Task processing frequency and timezone

Under **Tasks & subscriptions > Scheduling**, task processing frequency is High, Medium (default), or Low. `tasks.task_processing_frequency` accepts `HIGH`, `MEDIUM`, or `LOW`, corresponding to checks every 30 seconds, 1 minute, or 5 minutes after the previous pass. A pass confirms active task progress and triggers eligible Stash scans. High reduces the wait between passes at the cost of more background checks and downloader/Stash requests; Low reduces this overhead. Actual waits also depend on processing time and upstream availability. Download speed, media-processing concurrency, resource-search retry intervals and activity-based performer checks are separate.

Subscribed-performer checks, upcoming-release handoff and release retries run automatically while Moji is running. Existing per-performer deadlines and failure backoff remain persisted across restart. Subscription membership and normal eligibility still apply.

**System > General > Timezone** provides Follow system, UTC, and IANA geographic timezones. `system.timezone` defaults to `auto`; this follows the Moji process/container timezone, not the browser. The same setting controls calendar decisions, activity scoring, upcoming release boundaries, displayed timestamps and new log-file timestamp offsets. Instants stay stored in UTC; durations, relative elapsed times, remote protocol dates and film calendar dates do not move when the timezone changes. A successful save updates displays without discarding page state; a failed save keeps the active timezone and the pending draft.
