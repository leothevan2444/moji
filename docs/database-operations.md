# SQLite storage and recovery

Moji uses embedded SQLite (modernc driver, SQLite 3.51.3), with no database server to deploy. Build with Go 1.25 or newer. DuckDB is not used for transactional workflow state.

## Files and ownership

Start the application with `-data-dir /path/to/moji`. Only one application or maintenance process may own that directory; an OS lock, not the PID written in the lock file, enforces this. Stale PID text after a crash is harmless.

Fresh installations create databases directly in the data directory. Migrated/restored installations use `.moji-storage.json` to select **all** files from `.storage/generations/<generation>/`. Do not infer the active database from a leftover root `moji.db`; use the storage tool and the configured data directory.

| File | Contents | Durability / recovery |
| --- | --- | --- |
| `moji.db` | Tasks, resolved download paths, identities, ordered file choices, sourcing/restart/ingest checkpoints, review decisions, subscriptions/checks, important audit and external command intentions | `synchronous=FULL`; back up and restore |
| `runtime.db` | Latest progress/qB observations, downloader snapshot and disposable diagnostics | `synchronous=NORMAL`; reconstruct through observation |
| `cache/stashbox/cache.db` | Rebuildable upstream entity/page cache | `synchronous=NORMAL`; reconstruct from upstream |
| `legacy-task-events.jsonl.gz` | Complete pre-migration task history | Immutable; core manifests reference its digest; included in core backups |

Relations requiring atomic decisions stay in the core file. No business invariant assumes a transaction can commit atomically across these three files. There is no tenant/hash sharding: current workload benefits from isolating write frequency and recovery roles, not adding distributed transactions.

Each file has one bounded writer and four query-only reader connections. Transactions never span network/media operations. Busy handling uses short native waits and context-aware retries. Task revisions reject stale writes; restart generations fence old workers. File choices are committed as unapplied intentions before qB priorities/resume can change.

## Upgrade and initial migration

SQL migrations have ordered versions, names and immutable checksums in `schema_migrations`. Unknown/newer versions or changed checksums stop startup; there is no automatic reset. The core is currently version 2; runtime/cache are version 1. Independent checkpoint payload versions also reject unsupported data.

Before a supported nonempty upgrade, Moji creates a protected verified backup. DDL and its ledger entry commit in the same transaction. Add future migrations instead of editing an applied migration. Keep important relations in core, evolve typed/indexed fields there, and reserve JSON for versioned recovery payloads or upstream snapshots. Rewrite large data in resumable bounded batches when a future migration requires it.

For a legacy schema-16 installation, first stop Moji and preserve verified raw DB/WAL/config files plus standalone SQLite backups. Then prepare a new generation from those **consistent backups**, leaving originals unchanged:

```sh
go run ./cmd/moji-storage prepare \
  -source /safe/backup/core.sqlite \
  -cache /safe/backup/stashbox.sqlite \
  -destination /path/to/moji/.storage/generations/initial-v2
```

Preparation compares every task domain field, review and check item, hashes every copied entity/cache table in primary-key order, and streams/validates the entire old event archive. Failed preparation leaves an unpublished staging directory; inspect it, then choose a new destination for another attempt. Activation is separate and offline:

```sh
go run ./cmd/moji-storage activate -data-dir /path/to/moji -generation initial-v2
```

Activation protects the selected previous core physically and validates the staged schemas, integrity, closed WALs and synced files before atomically replacing one pointer. Existing originals/generations remain available for recovery. After new writes are accepted, restore/reconcile deliberately; pointing at a stale generation would discard those newer decisions from the active view.

## Backup and restore

The application takes an initial backup and then one per hour. `-backup-dir /backup-volume/moji` selects another filesystem location; default is `data-dir/backups`. Configure a separate disk/NAS/off-host copy when protection against host/disk loss is required; the default local directory only provides local recovery. Expected backup recovery-point loss is up to one hour after complete disk loss, while normal process crashes recover committed core WAL transactions.

Backups use SQLite's online API, not a copy of an open main file without its WAL. Each standalone snapshot is integrity/FK checked, hashed and published with a manifest. Referenced immutable history, the configured YAML and available `.moji-receipts/*.json` are copied into private backup assets and independently hashed. These non-SQL files are individually atomic, not part of a cross-file SQL transaction. Backup files are 0600 and directories 0700.

Retention keeps the newest regular snapshot and the newest in 24 hourly, 14 daily and 8 weekly buckets. Protected migration/compaction/restore snapshots and original recovery directories are never automatically pruned. A synced prune intention makes interrupted deletion resumable; changed surviving files stop pruning.

Offline manual operations:

```sh
go run ./cmd/moji-storage backup -data-dir /path/to/moji -config /path/to/config.yaml -protect -reason before-maintenance
go run ./cmd/moji-storage verify-backup -manifest /backup/path/snapshot.sqlite.json
go run ./cmd/moji-storage restore -data-dir /path/to/moji -manifest /backup/path/snapshot.sqlite.json
```

Restore verifies the snapshot and all referenced assets, preserves the existing core even if damaged, copies into a new generation, applies supported known upgrades, creates fresh runtime/cache files and atomically selects the result. It initializes no qB/Stash client and replays no commands. Configuration/receipts from the backup are available inside the returned generation under `configuration/` and `receipts/`; restore them to the intended configured locations before restarting if those files were lost. A normal database-only restore keeps existing configuration/media in place.

Video files, image caches, Stash's database/custom-field subscription markers and upstream services are not contained in a Moji core backup. Back up Stash/media independently. Once running, qB observations reconcile pending add/removal intentions; an unavailable upstream never proves a torrent is missing or a mutation failed.

## Retention and performance

Diagnostics retain 14 days and the latest 100000 IDs, removed in batches of at most 1000. Important audit/choices/command provenance are not TTL-deleted. Progress uses one latest observation per task and coalesces writes; no unbounded progress-event history is generated.

StashBox uses its configured TTL (default 24 hours), a 30-day stale window, active plus latest inactive generation, and a default 512 MiB live-page budget. `system.stash_box_data_cache.max_size_mb` adjusts the budget; TTL-only settings changes preserve it. Cleanup releases the writer between batches and includes indexes/link pages in the capacity check. Free pages are reused, so file length need not immediately shrink after deletion.

Background work uses indexed lifecycle queries and keyset batches of 128. Idle monitoring reads narrow headers, not torrent attachments/file plans. Event statistics are SQL aggregates; the home overview decodes only bounded samples. Public task-list pagination/response-size changes are outside this database migration.

Inspect or explicitly compact during a stopped maintenance window:

```sh
go run ./cmd/moji-storage status -data-dir /path/to/moji
go run ./cmd/moji-storage compact -data-dir /path/to/moji
```

Compaction first makes a protected backup, checks for held snapshots, then runs SQLite VACUUM and a truncating checkpoint. Online maintenance uses passive checkpoints; no code manually removes a live WAL. A reader may delay truncation, so investigate a sustained WAL rather than deleting it. Status reports engine/schema versions, page/free/WAL sizes, reader/writer pools, queue, busy retries and timing metrics.
