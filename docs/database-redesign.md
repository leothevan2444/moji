# Database redesign implementation and acceptance

Accepted objective: replace the fragmented database infrastructure with the reviewed durable-core / runtime-observation / rebuildable-cache design, complete non-destructive upgrades, migrate the real existing database without losing workflows, implement retention, backups and bounded database performance, and resume the workflow against the migrated data.

Baseline commit: 89c5f25. The user has authorized implementation and migration and is stopping the running instance. Do not migrate the source database until the application is confirmed stopped, a verified durable backup exists, and the full migration has passed on a copy. Do not use the real qB/Stash endpoints in isolated restore tests.

## Required acceptance evidence

- [x] Verified untouched raw originals and consistent SQLite backups, checksums, full integrity/FK validation and an off-main-database recovery path.
- [x] SQLite engine version includes the WAL-reset fix, verified by a SQL version query and regression tests.
- [x] One storage owner per database, bounded write admission, small query-only read pools, consistent per-connection settings, shutdown and single-instance lifecycle.
- [x] Durable core normalized task state, cold details/attachments, file choices, checkpoints, decisions, external operation intentions and audit.
- [x] Separate runtime database for replaceable qB observations/progress and diagnostics; no important deletion intention or policy moved into a disposable snapshot.
- [x] Separate StashBox cache with safe migrations, delta writes, TTL/generation/capacity retention.
- [x] No business auto-reset/drop-on-version-mismatch; ordered transactional migrations with checksums and refusal of unknown/newer schemas.
- [x] Versioned durable checkpoints and conflict-safe task updates; important decisions and dependent events share a transaction.
- [x] Typed indexed review/check data; independent check items and differential subscription relations; important decisions survive unsubscribe/task history pruning.
- [x] No full task scan per task event; bounded queue/query operations and SQL aggregate stats; observed update performance tests.
- [x] Online backup API, retention policy, manifests, restore validation/CLI, migration-preflight backup and safe failure paths.
- [x] Historical event archival with verified counts/digests, bounded purge, controlled WAL/checkpoint/compaction; no deletion based only on unchanged stage.
- [x] Durable configuration writes and backup scope covering required non-SQL state.
- [x] Crash/restart, concurrent update/read, busy/disk error, migration failure, unknown schema and restore tests; full project validation.
- [x] Rehearsal against a copy of the actual database: compare every important decision, identity, selection and checkpoint; validate FK/integrity and retained history archives.
- [x] Actual stopped-source migration with original backup retained, workflows and pending commands reconciled; controlled startup and runtime checks.

## Progress

2026-10-27: confirmed the process and :10000 listener are absent. Original source remains untouched. Protected raw originals and standalone SQLite snapshots live in `backups/before-storage-v2-20261027T030816Z` (integrity/FK/hash validated). Source has 277 tasks, 2,592,577 task events, 435 reviews, 291 release entities, 337 links, 11 performer states and 8 check runs.

Implementation foundations (production acceptance is recorded below):

- SQLite driver v1.46.2 / engine 3.51.3, Go 1.25 minimum; SQL engine version test passed.
- `internal/storage`: one writer, bounded admission, query-only reader pools, checksummed migrations, OS directory lock, coalesced runtime observations, online backup, artifact manifests, backup retention, atomic generation pointer and core restore.
- Normalized relational task repository preserves all task fields; CAS revisions reject stale writes; core task/audit commit together; progress-only updates never reserve the core writer. Cold attachment upserts and unchanged file selections avoid churn. SQL event statistics avoid full task loading.
- Managed subscription repository shares the core owner, updates links/items differentially and preserves decisions after unsubscribe. Typed review/check tables retain raw review provenance.
- Managed downloader stores policy and removal commands in core, observation snapshots in runtime. Ambiguous removal responses remain UNCERTAIN; losing runtime does not discard intent, policy or confirmed provenance. Persistence failure prevents the qB mutation.
- Managed StashBox cache uses the shared owner, delta snapshots and active/latest-inactive generation retention. Unknown versions require explicit rebuild, no silent reset.
- Production main receives the manager and joins HTTP/background workers before closing pools; media/check/cleanup jobs have shutdown joins. The selected generation and production lifecycle were subsequently verified below.
- Config writes use fsync plus atomic rename and 0600. Scheduled backups include configured YAML and available delivery receipts. Referenced legacy history is copied into backup assets and verified independently of live files.
- Offline CLI `go run ./cmd/moji-storage prepare|backup|verify-backup|restore` never creates external clients. Restore tests protect previous state, reject an active owner and recover a missing selected core.

Validation evidence:

- `go test ./...` passed before application wiring; later focused main tests passed.
- Race tests for storage/taskruntime/subscription/downloader/StashBox/main passed after application wiring (`/tmp/moji-storage-lifecycle-race.log`). Subsequent storage restore/artifact race tests passed.
- Two real-source-copy rehearsals: `/tmp/moji-storage-rehearsal-20261027a` and `...27b`. Second rehearsal also compared every domain task field, review record and check item. Counts match; full history archive is 28 MiB, 2,592,577 rows, SHA256 `fd7d60c11bf2864fc20c83a0b8781f638b242bd339c20d6e64fcd35f21d290e8`.
- Neither temporary rehearsal was published. The final prepared generation was separately activated as recorded below; original root database/WAL and source backups remain preserved.

## Final implementation evidence (before production publication)

- Core schema v2 persists resolved download locations as ingest inputs. Discarding runtime before pending ingest is tested; restored ingest proceeds through generate/identify to completion without replaying scan.
- Queue/monitor/ingest worker queries use 128-item keyset batches and narrow headers. Idle completed tasks skip detailed loads. Home overview and event stats are bounded/SQL aggregates. Subscription List uses two snapshot queries instead of N+1; review task matching reads only task IDs/codes.
- File selections now commit the chosen order/policy as unapplied durable intent before any pause/priority/resume request. Persistence failure prevents qB mutation. Restart preserves CAS revision and advances generation.
- Task add/delete/restart deletion record durable external intentions; successful complete qB listings reconcile outcomes without mutation replay. Checkpoint payload guards prevent future-format data from triggering writes/actions.
- Runtime diagnostics have 14-day/100000-ID retention in <=1000-row batches. Cache has a configurable 512-MiB live-page budget including indexes/link data, bounded cleanup and generation retention. Important audit remains durable rather than TTL-deleted.
- Offline status/activate/compact/restore tools implemented. Protected pre-upgrade/restore/compaction backups, immutable history/config/receipt assets, corruption checks and crash-resumable backup pruning are tested. `-backup-dir` supports another filesystem target; no off-host destination has been configured in this environment.
- Native SQLite busy handling was measured delaying a 100-ms context by 4.8 seconds. Short native waits plus Go-side retry budgeting now pass the deadline/recovery test.
- Actual SQLITE_FULL rollback, SIGKILL/reopen with committed choices and incomplete WAL changes, damaged-core recovery, unknown/future schema/payload refusal, and HTTP shutdown with an open GraphQL WebSocket all pass.
- Config atomic write failure now restores the last valid persisted state in memory. Configuration writes and backup artifacts remain private.
- Synthetic 1000-task/8-KiB attachment workload: event stats ~1.13 ms, 128 monitor headers ~3.0 ms, progress updates ~0.88 ms with zero core writes (`/tmp/moji-database-benchmarks.log`). These are local measurements, not an unlimited-load guarantee.
- Real-source rehearsal C (`/tmp/moji-storage-rehearsal-20261027c`) passed strict legacy payload validation, every task/review/check comparison, and full primary-key-ordered hashes of all copied subscription/cache tables. Core source SHA256: `6001f3f255dece2ef0cd6355ca60416bab72dbd7643d93d84171b65c069a0bd3`; complete 2,592,577-event archive retains the prior verified digest.
- Original core/WAL/config/cache-main hashes remain unchanged. SHM is a rebuildable runtime index and its reader bookkeeping differs; raw originals and verified standalone backups are preserved.
- Full backend suite passed (`/tmp/moji-database-final-suite.log`); the last batched-read changes passed focused validation, including the main process under the required localhost-test permissions.

## Production acceptance

Implementation commits: `c1fd5a4` (storage redesign), `a28bbeb` (protected manual backups), `13ab389` (split-table duplicate identity errors), `f69b976` (atomic cache snapshot reads). The final runtime is built from `f69b976`.

- Final prepared generation: `/home/wangqi/services/moji/.storage/generations/initial-v2-20261027`; selected atomically through `.moji-storage.json`.
- Preserved original files and executable: `backups/before-storage-v2-20261027T030816Z`. A further verified physical pre-activation copy is in `backups/before-restore-raw-20261027T072847Z-43136806-cbc1-40d1-b862-da00ed0646d3`. The old root database files remain recovery originals, not active application databases.
- Protected migrated snapshot: `backups/20261027T072310.293334678Z-fae72ce9f49e7542.sqlite.json`. Core is 4,030,464 bytes; 116 verified assets include configuration, 114 delivery receipts and the complete legacy event archive. Original task/review/check/entity/cache values were verified before publication.
- This snapshot was restored into `/tmp/moji-final-restore-20261027`. Every stored row across all 21 core tables matched the prepared source; integrity/FK checks and all assets passed. Evidence is retained in `.storage/verification/offline-restoration.json`.
- Application startup and graceful restart passed against real migrated data. Current runtime PID is recorded in `.storage/runtime.json` (last verified PID 3139315); startup arguments retain the original config and port 10000. All open database descriptors point into the selected generation, verified from the live process.
- Final API acceptance: 277 original task IDs retained, 11 performer states available, qB connected with a fresh advancing observation. During observation, completed tasks advanced from 110 to 116 and existing transfer/scan jobs continued. One newly blocked scan requires complete Stash Identify metadata; no database failure caused it.
- All 258 original plan headers and 225 nonempty original file-choice lists were rechecked live and retained. Three previously empty plans gained observed torrent metadata normally. Reviews/check history remain 435/8.
- The actual running service created a scheduled backup with SQLite 3.51.3; its manifest and assets passed independent verification. A further protected backup was taken before the final graceful runtime restart.
- Final live integrity and FK checks passed for core/runtime/cache. Fresh-runtime logs contained no SQLITE_BUSY/FULL, locked-database, schema/column/checkpoint, constraint or storage-maintenance errors.
- Full backend suite passed (`/tmp/moji-database-final-suite.log`). Final race suite passed for storage, taskruntime, subscription, downloader, cache, config, taskflow and main (`/tmp/moji-final-acceptance-race.log`). The last cache snapshot/eviction concurrency regression passed separately (`/tmp/moji-cache-consistency-final.log`).
- Detailed local acceptance artifacts are under `.storage/verification/`; operational commands and recovery scope are documented in `docs/database-operations.md`.

The requested database implementation, source-preserving migration, cleanup, backup/restore and performance work has been deployed and verified. Automatic backups currently use the local backup directory; an independent off-host destination has not been configured. `-backup-dir` supports an independently mounted backup target without introducing a database service.
