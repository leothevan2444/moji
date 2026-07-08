package downloader

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed sqlite_schema.sql
var sqliteSchema string

const sqliteSchemaVersion = "4"

func OpenSQLiteDatabase(path string) (*sqlx.DB, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, fmt.Errorf("downloader: sqlite db path is required")
	}

	dir := filepath.Dir(trimmed)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("downloader: create sqlite dir %q: %w", dir, err)
		}
	}

	db, err := sqlx.Open("sqlite", trimmed)
	if err != nil {
		return nil, fmt.Errorf("downloader: open sqlite db %q: %w", trimmed, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := configureSQLiteDatabase(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func configureSQLiteDatabase(db *sqlx.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("downloader: configure sqlite pragma %q: %w", pragma, err)
		}
	}

	versionBeforeInit, err := readSQLiteSchemaVersionSafe(db)
	if err != nil {
		return err
	}
	hadSchema, err := sqliteTableExists(db, "tasks")
	if err != nil {
		return err
	}
	if !hadSchema {
		hadSchema, err = sqliteTableExists(db, "subscription_release_entities")
		if err != nil {
			return err
		}
	}

	if _, err := db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("downloader: initialize sqlite schema: %w", err)
	}
	if err := migrateSQLiteDatabase(db, hadSchema, versionBeforeInit); err != nil {
		return err
	}
	if err := ensureSQLiteRuntimeState(db); err != nil {
		return err
	}

	return nil
}

func migrateSQLiteDatabase(db *sqlx.DB, hadSchema bool, version string) error {
	if !hadSchema && version == "" {
		return nil
	}
	if version == sqliteSchemaVersion {
		return nil
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("downloader: begin sqlite schema migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("downloader: disable sqlite foreign keys for migration: %w", err)
	}
	if err := ensureLegacySQLiteColumns(tx); err != nil {
		return err
	}

	if err := recreateTasksTable(tx); err != nil {
		return err
	}
	if err := recreateTaskEventsTable(tx); err != nil {
		return err
	}
	if err := recreateSubscriptionPerformerStateTable(tx); err != nil {
		return err
	}
	if err := recreateSubscriptionReleaseEntitiesTable(tx); err != nil {
		return err
	}
	if err := recreateSubscriptionPerformerReleasesTable(tx); err != nil {
		return err
	}
	if err := recreateSQLiteIndexes(tx); err != nil {
		return err
	}
	if err := clearDanglingSubscriptionTaskReferences(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("downloader: re-enable sqlite foreign keys after migration: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO task_store_meta (key, value) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, sqliteSchemaVersion); err != nil {
		return fmt.Errorf("downloader: update sqlite schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("downloader: commit sqlite schema migration: %w", err)
	}
	return nil
}

func ensureSQLiteRuntimeState(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("downloader: begin sqlite runtime state tx: %w", err)
	}
	defer tx.Rollback()

	if err := recreateSQLiteIndexes(tx); err != nil {
		return err
	}
	if err := clearDanglingSubscriptionTaskReferences(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO task_store_meta (key, value) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, sqliteSchemaVersion); err != nil {
		return fmt.Errorf("downloader: persist sqlite schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("downloader: commit sqlite runtime state tx: %w", err)
	}
	return nil
}

func ensureLegacySQLiteColumns(tx *sqlx.Tx) error {
	taskMigrations := map[string]string{
		"source":                  "TEXT NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'SEARCH', 'SUBSCRIPTION'))",
		"code":                    "TEXT NOT NULL DEFAULT ''",
		"torrent_identity_hash":   "TEXT NOT NULL DEFAULT ''",
		"torrent_identity_magnet": "TEXT NOT NULL DEFAULT ''",
		"stash_mode":              "TEXT NOT NULL DEFAULT ''",
		"stash_source_path":       "TEXT NOT NULL DEFAULT ''",
		"stash_transfer_action":   "TEXT NOT NULL DEFAULT ''",
		"stash_transfer_path":     "TEXT NOT NULL DEFAULT ''",
		"stash_transfer_status":   "TEXT NOT NULL DEFAULT '' CHECK (stash_transfer_status IN ('', 'started', 'completed', 'failed'))",
		"stash_transfer_error":    "TEXT NOT NULL DEFAULT ''",
		"stash_scan_path":         "TEXT NOT NULL DEFAULT ''",
		"stash_scan_hint":         "TEXT NOT NULL DEFAULT ''",
	}
	for name, definition := range taskMigrations {
		if err := ensureSQLiteColumnTx(tx, "tasks", name, definition); err != nil {
			return err
		}
	}

	subscriptionReleaseMigrations := map[string]string{
		"performer_count": "INTEGER NOT NULL DEFAULT 0",
		"performer_names": "TEXT NOT NULL DEFAULT '[]'",
		"classification":  "TEXT NOT NULL DEFAULT ''",
		"decision":        "TEXT NOT NULL DEFAULT ''",
		"decision_reason": "TEXT NOT NULL DEFAULT ''",
	}
	for name, definition := range subscriptionReleaseMigrations {
		if err := ensureSQLiteColumnTx(tx, "subscription_release_entities", name, definition); err != nil {
			return err
		}
	}

	return nil
}

func ensureSQLiteColumnTx(tx *sqlx.Tx, table string, column string, definition string) error {
	exists, err := sqliteColumnExistsTx(tx, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("downloader: add sqlite column %s.%s: %w", table, column, err)
	}
	return nil
}

func recreateTasksTable(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "tasks")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	statements := []string{
		`DROP TABLE IF EXISTS tasks_v4_new`,
		`CREATE TABLE tasks_v4_new (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'SEARCH', 'SUBSCRIPTION')),
			query TEXT NOT NULL,
			code TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL CHECK (status IN ('pending', 'added', 'downloading', 'completed', 'failed')),
			torrent_url TEXT NOT NULL DEFAULT '',
			save_path TEXT,
			category TEXT,
			tags TEXT,
			torrent_identity_hash TEXT,
			torrent_identity_magnet TEXT,
			torrent_hash TEXT,
			torrent_name TEXT,
			progress REAL NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 1),
			qbittorrent_state TEXT,
			content_path TEXT,
			completed_at TEXT,
			stash_mode TEXT,
			stash_source_path TEXT,
			stash_transfer_action TEXT,
			stash_transfer_path TEXT,
			stash_transfer_status TEXT CHECK (stash_transfer_status IN ('started', 'completed', 'failed')),
			stash_transfer_error TEXT,
			stash_job_id TEXT,
			stash_scan_path TEXT,
			stash_scan_status TEXT CHECK (stash_scan_status IN ('started', 'failed')),
			stash_scan_error TEXT,
			stash_scan_hint TEXT,
			stash_scan_started_at TEXT,
			error TEXT,
			candidate_title TEXT NOT NULL DEFAULT '',
			candidate_tracker TEXT NOT NULL DEFAULT '',
			candidate_info_hash TEXT NOT NULL DEFAULT '',
			candidate_link TEXT NOT NULL DEFAULT '',
			candidate_magnet_uri TEXT NOT NULL DEFAULT '',
			candidate_size INTEGER NOT NULL DEFAULT 0,
			candidate_seeders INTEGER NOT NULL DEFAULT 0,
			candidate_peers INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		) STRICT`,
		`INSERT INTO tasks_v4_new (
			id, source, query, code, status, torrent_url, save_path, category, tags,
			torrent_identity_hash, torrent_identity_magnet, torrent_hash, torrent_name, progress, qbittorrent_state, content_path,
			completed_at, stash_mode, stash_source_path, stash_transfer_action, stash_transfer_path,
			stash_transfer_status, stash_transfer_error, stash_job_id, stash_scan_path, stash_scan_status,
			stash_scan_error, stash_scan_hint, stash_scan_started_at, error,
			candidate_title, candidate_tracker, candidate_info_hash, candidate_link, candidate_magnet_uri,
			candidate_size, candidate_seeders, candidate_peers, created_at, updated_at
		)
		SELECT
			id,
			COALESCE(NULLIF(source, ''), 'MANUAL'),
			query,
			COALESCE(code, ''),
			status,
			torrent_url,
			NULLIF(save_path, ''),
			NULLIF(category, ''),
			NULLIF(tags, ''),
			NULLIF(torrent_identity_hash, ''),
			NULLIF(torrent_identity_magnet, ''),
			NULLIF(torrent_hash, ''),
			NULLIF(torrent_name, ''),
			progress,
			NULLIF(qbittorrent_state, ''),
			NULLIF(content_path, ''),
			completed_at,
			NULLIF(stash_mode, ''),
			NULLIF(stash_source_path, ''),
			NULLIF(stash_transfer_action, ''),
			NULLIF(stash_transfer_path, ''),
			NULLIF(stash_transfer_status, ''),
			NULLIF(stash_transfer_error, ''),
			NULLIF(stash_job_id, ''),
			NULLIF(stash_scan_path, ''),
			NULLIF(stash_scan_status, ''),
			NULLIF(stash_scan_error, ''),
			NULLIF(stash_scan_hint, ''),
			stash_scan_started_at,
			NULLIF(error, ''),
			COALESCE(candidate_title, ''),
			COALESCE(candidate_tracker, ''),
			COALESCE(candidate_info_hash, ''),
			COALESCE(candidate_link, ''),
			COALESCE(candidate_magnet_uri, ''),
			candidate_size,
			candidate_seeders,
			candidate_peers,
			created_at,
			updated_at
		FROM tasks`,
		`DROP TABLE tasks`,
		`ALTER TABLE tasks_v4_new RENAME TO tasks`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: migrate tasks table: %w", err)
		}
	}

	return nil
}

func recreateTaskEventsTable(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "task_events")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	statements := []string{
		`DROP TABLE IF EXISTS task_events_v4_new`,
		`CREATE TABLE task_events_v4_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
			message TEXT NOT NULL,
			old_status TEXT NOT NULL DEFAULT '',
			new_status TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		) STRICT`,
		`INSERT INTO task_events_v4_new (id, task_id, event_type, level, message, old_status, new_status, created_at)
		SELECT id, task_id, event_type, level, message, COALESCE(old_status, ''), COALESCE(new_status, ''), created_at
		FROM task_events`,
		`DROP TABLE task_events`,
		`ALTER TABLE task_events_v4_new RENAME TO task_events`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: migrate task_events table: %w", err)
		}
	}
	return nil
}

func recreateSubscriptionPerformerStateTable(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "subscription_performer_state")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	statements := []string{
		`DROP TABLE IF EXISTS subscription_performer_state_v4_new`,
		`CREATE TABLE subscription_performer_state_v4_new (
			performer_id TEXT PRIMARY KEY,
			last_checked_at TEXT,
			last_error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		) STRICT`,
		`INSERT INTO subscription_performer_state_v4_new (performer_id, last_checked_at, last_error, created_at, updated_at)
		SELECT performer_id, last_checked_at, NULLIF(last_error, ''), created_at, updated_at
		FROM subscription_performer_state`,
		`DROP TABLE subscription_performer_state`,
		`ALTER TABLE subscription_performer_state_v4_new RENAME TO subscription_performer_state`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: migrate subscription_performer_state table: %w", err)
		}
	}
	return nil
}

func recreateSubscriptionReleaseEntitiesTable(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "subscription_release_entities")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	statements := []string{
		`DROP TABLE IF EXISTS subscription_release_entities_v4_new`,
		`CREATE TABLE subscription_release_entities_v4_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			release_key TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered', 'pending', 'processed', 'failed')),
			source TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			code TEXT NOT NULL DEFAULT '',
			release_date TEXT,
			url TEXT,
			query TEXT NOT NULL DEFAULT '',
			task_id TEXT,
			performer_count INTEGER NOT NULL DEFAULT 0,
			performer_names TEXT NOT NULL DEFAULT '[]',
			classification TEXT NOT NULL DEFAULT '',
			decision TEXT NOT NULL DEFAULT '',
			decision_reason TEXT NOT NULL DEFAULT '',
			seen_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
		) STRICT`,
		`INSERT INTO subscription_release_entities_v4_new (
			id, release_key, status, source, title, code, release_date, url, query, task_id,
			performer_count, performer_names, classification, decision, decision_reason, seen_at, created_at, updated_at
		)
		SELECT
			id,
			release_key,
			status,
			COALESCE(source, ''),
			COALESCE(title, ''),
			COALESCE(code, ''),
			NULLIF(release_date, ''),
			NULLIF(url, ''),
			COALESCE(query, ''),
			NULLIF(task_id, ''),
			COALESCE(performer_count, 0),
			COALESCE(performer_names, '[]'),
			COALESCE(NULLIF(classification, ''), 'UNKNOWN'),
			COALESCE(NULLIF(decision, ''), 'QUEUED'),
			COALESCE(decision_reason, ''),
			seen_at,
			created_at,
			updated_at
		FROM subscription_release_entities`,
		`DROP TABLE subscription_release_entities`,
		`ALTER TABLE subscription_release_entities_v4_new RENAME TO subscription_release_entities`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: migrate subscription_release_entities table: %w", err)
		}
	}
	return nil
}

func recreateSubscriptionPerformerReleasesTable(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "subscription_performer_releases")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	statements := []string{
		`DROP TABLE IF EXISTS subscription_performer_releases_v4_new`,
		`CREATE TABLE subscription_performer_releases_v4_new (
			performer_id TEXT NOT NULL,
			release_id INTEGER NOT NULL,
			linked_at TEXT NOT NULL,
			PRIMARY KEY (performer_id, release_id),
			FOREIGN KEY (performer_id) REFERENCES subscription_performer_state(performer_id) ON DELETE CASCADE,
			FOREIGN KEY (release_id) REFERENCES subscription_release_entities(id) ON DELETE CASCADE
		) STRICT`,
		`INSERT INTO subscription_performer_releases_v4_new (performer_id, release_id, linked_at)
		SELECT performer_id, release_id, linked_at
		FROM subscription_performer_releases`,
		`DROP TABLE subscription_performer_releases`,
		`ALTER TABLE subscription_performer_releases_v4_new RENAME TO subscription_performer_releases`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: migrate subscription_performer_releases table: %w", err)
		}
	}
	return nil
}

func recreateSQLiteIndexes(tx *sqlx.Tx) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at ON tasks (status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks (updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_completed_at ON tasks (completed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_torrent_hash ON tasks (torrent_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_candidate_info_hash ON tasks (candidate_info_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_stash_job_id ON tasks (stash_job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_scan_queue ON tasks (updated_at DESC) WHERE status = 'completed' AND stash_job_id IS NULL AND stash_scan_status IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_code_unique ON tasks (code) WHERE code <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_torrent_identity_hash_unique ON tasks (torrent_identity_hash) WHERE torrent_identity_hash IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_torrent_identity_magnet_unique ON tasks (torrent_identity_magnet) WHERE torrent_identity_magnet IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_task_events_task_created_at ON task_events (task_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_task_events_created_at ON task_events (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_state_last_checked_at ON subscription_performer_state (last_checked_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_state_updated_at ON subscription_performer_state (updated_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_release_entities_key ON subscription_release_entities (release_key)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_date ON subscription_release_entities (release_date DESC, seen_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_code ON subscription_release_entities (code)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_status_seen_at ON subscription_release_entities (status, seen_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_task_id ON subscription_release_entities (task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_performer_linked_at ON subscription_performer_releases (performer_id, linked_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_release_id ON subscription_performer_releases (release_id)`,
	}

	for _, query := range indexes {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("downloader: create sqlite index: %w", err)
		}
	}
	return nil
}

func clearDanglingSubscriptionTaskReferences(tx *sqlx.Tx) error {
	if _, err := tx.Exec(`
UPDATE subscription_release_entities
SET task_id = NULL
WHERE task_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM tasks
    WHERE tasks.id = subscription_release_entities.task_id
  )`); err != nil {
		return fmt.Errorf("downloader: clear dangling subscription task references: %w", err)
	}
	return nil
}

func sqliteTableExists(db sqlx.ExtContext, table string) (bool, error) {
	var count int
	if err := sqlx.GetContext(context.Background(), db, &count, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table); err != nil {
		return false, fmt.Errorf("downloader: inspect sqlite table %s: %w", table, err)
	}
	return count > 0, nil
}

func sqliteColumnExists(db *sqlx.DB, table string, column string) (bool, error) {
	rows, err := db.Queryx(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("downloader: inspect sqlite table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("downloader: scan sqlite table info for %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("downloader: iterate sqlite table info for %s: %w", table, err)
	}
	return false, nil
}

func sqliteColumnExistsTx(tx *sqlx.Tx, table string, column string) (bool, error) {
	rows, err := tx.Queryx(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("downloader: inspect sqlite table %s in tx: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("downloader: scan sqlite table info for %s in tx: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("downloader: iterate sqlite table info for %s in tx: %w", table, err)
	}
	return false, nil
}

func sqliteTableExistsTx(tx *sqlx.Tx, table string) (bool, error) {
	var count int
	if err := tx.Get(&count, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table); err != nil {
		return false, fmt.Errorf("downloader: inspect sqlite table %s in tx: %w", table, err)
	}
	return count > 0, nil
}

func readSQLiteSchemaVersionSafe(db *sqlx.DB) (string, error) {
	exists, err := sqliteTableExists(db, "task_store_meta")
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return readSQLiteSchemaVersion(db)
}

func readSQLiteSchemaVersion(db *sqlx.DB) (string, error) {
	var version string
	err := db.Get(&version, `SELECT value FROM task_store_meta WHERE key = 'schema_version'`)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("downloader: read sqlite schema version: %w", err)
	}
	return version, nil
}

func formatSQLiteTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatOptionalSQLiteTimestamp(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatSQLiteTimestamp(*value)
}

func parseOptionalSQLiteTimestamp(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw.String)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func parseSQLiteTimestamp(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
