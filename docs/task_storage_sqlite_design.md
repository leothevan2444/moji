# SQLite Task Store Design

## Goal

- Replace the temporary single-JSON task store with a SQLite-backed store that can handle:
  - large task volumes;
  - filtered task listing;
  - future pagination;
  - task event history;
  - safer concurrent reads/writes.

## Design Summary

- Use a single local SQLite database file for task persistence.
- Keep the main task snapshot in one wide `tasks` table.
- Keep operator/audit history in a separate `task_events` table.
- Do not normalize candidate fields into a separate table yet; current task data is a single embedded object and is always read together with the task.

## Table: `tasks`

- `tasks` stores the latest state of each Moji task.
- It maps directly to `internal/taskruntime.Task` and `Candidate`.
- Candidate fields are flattened into the same row to keep reads simple and avoid unnecessary joins in the first implementation.

### Key Choices

- `id` remains the application-generated task ID.
- Timestamps are stored as RFC3339 UTC text so the database remains easy to inspect manually.
- `stage` and `stage_status` are constrained with `CHECK` rules to match current runtime enums.
- `progress` is constrained to the `[0, 1]` range.
- Optional text remains nullable in SQLite where "missing" and "empty" are meaningfully different in the runtime model.

### Important Indexes

- `(stage, created_at DESC)` for task-center stage views.
- `(stage_status, created_at DESC)` for blocked/pending/running views.
- `(updated_at DESC)` for default newest/recent activity views.
- `(download_completed_at DESC)` for completed-download history.
- `code`, `torrent_identity_hash`, and `torrent_identity_magnet` support strict task dedupe.
- Partial `scan_queue` index for `PENDING_INGEST + PENDING` tasks waiting for Stash scan triggering.

## Table: `task_events`

- `task_events` stores append-only operational history for each task.
- This is not required to replace the JSON store, but it is a good foundational table for:
  - status transition history;
  - manual operator actions;
  - failure trail;
  - future per-task timeline UI.

### Event Model

- `event_type` is intentionally open text in v1 so the application can evolve event vocabulary without immediate schema churn.
- `level` supports UI tone and troubleshooting.
- `old_stage` and `new_stage` make product-stage transitions easy to query.

## Why Not More Tables Yet

- No separate `candidates` table:
  - each task currently has one chosen candidate snapshot, not a reusable candidate entity.
- No separate `stash_scan_runs` table:
  - current runtime only keeps latest scan state on the task itself.
- No migration table beyond `task_store_meta`:
  - the app is still pre-production, so a lightweight schema version key is enough for now.

## Recommended Runtime Settings

- SQLite file example: `moji.db`
- Enable `WAL`
- Enable `foreign_keys`
- Set a practical `busy_timeout`

## Subscription In The Same Database

- Moji Subscription should live in the same SQLite database as tasks.
- Stash `custom_fields` remains the source of truth for whether a performer is subscribed.
- SQLite stores only Moji-local runtime state:
  - last refresh/check time;
  - last refresh error;
  - global release entities discovered from the source;
  - performer-to-release linkage state;
  - global release processing state and task linkage.

### Table: `subscription_performer_state`

- One row per subscribed performer that Moji has local state for.
- This table does not duplicate performer profile data from Stash.
- It only stores Moji-owned state:
  - `performer_id`;
  - `last_checked_at`;
  - `last_error`;
  - `created_at`;
  - `updated_at`.

### Table: `subscription_release_entities`

- Stores the global release entity discovered from the external source.
- This is the shared film/release record, not a performer-specific view.
- `release_key` is globally unique and remains the dedupe anchor.
- A single release entity can be linked to multiple subscribed performers.
- Because one global release should produce one global Moji task, release-level processing fields also live here:
  - `status`;
  - `task_id`;

### Table: `subscription_performer_releases`

- Stores the pure linkage between a subscribed performer and a release entity.
- `linked_at` records when Moji associated that performer with the release.
- This table allows one release entity to be associated with multiple subscribed performers without duplicating the release snapshot itself.

### Key Choices For Subscription

- Keep release entities and performer links as rows, not JSON blobs:
  - easier dedupe;
  - easier cleanup;
  - future pagination becomes straightforward;
  - per-performer timeline queries stay cheap.
- Separate the global release entity from the performer link:
  - one film can have multiple performers;
  - multiple subscribed performers can point at the same release;
  - global release dedupe is preserved while performer-local state stays independent.
- Keep processing state on the release entity:
  - one release should create one global task;
  - `pending` / `processed` / `failed` therefore belong to the shared release, not to individual performer links.
- Use `ON DELETE CASCADE` from performer state to performer-release rows:
  - unsubscribing a performer should fully remove only that performer's Moji-local linkage state.
- Use `ON DELETE CASCADE` from release entity to performer-release rows:
  - if a release entity is ever pruned, all performer links disappear with it.
- Use `ON DELETE SET NULL` from release `task_id` to `tasks.id`:
  - if a task is ever pruned in the future, the release record can remain while dropping the task link.

## Next Implementation Steps

1. Add a `SQLiteTaskStore` implementation behind the existing `TaskStore` interface.
2. Map `Create`, `Update`, `Find`, and `List` onto the `tasks` table.
3. Record task lifecycle changes into `task_events`.
4. Add a `SQLiteStore` for Subscription using `subscription_performer_state`, `subscription_release_entities`, and `subscription_performer_releases`.
5. Switch `tasks.store` and `subscription.store` defaults from `json` to `sqlite`.
6. Expand both store interfaces later for filtered list and retention cleanup operations.
