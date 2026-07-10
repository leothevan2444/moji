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

const sqliteSchemaVersion = "7"

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

	if !hadSchema || versionBeforeInit == sqliteSchemaVersion {
		if _, err := db.Exec(sqliteSchema); err != nil {
			return fmt.Errorf("downloader: initialize sqlite schema: %w", err)
		}
	} else {
		if err := resetSQLiteDatabase(db); err != nil {
			return err
		}
		if _, err := db.Exec(sqliteSchema); err != nil {
			return fmt.Errorf("downloader: reinitialize sqlite schema: %w", err)
		}
	}
	if err := ensureSQLiteRuntimeState(db); err != nil {
		return err
	}

	return nil
}

func resetSQLiteDatabase(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("downloader: begin sqlite schema reset: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		`PRAGMA foreign_keys = OFF`,
		`DROP TABLE IF EXISTS subscription_performer_releases`,
		`DROP TABLE IF EXISTS subscription_release_entities`,
		`DROP TABLE IF EXISTS subscription_performer_state`,
		`DROP TABLE IF EXISTS task_events`,
		`DROP TABLE IF EXISTS tasks`,
		`DROP TABLE IF EXISTS task_store_meta`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("downloader: reset sqlite schema with %q: %w", statement, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("downloader: commit sqlite schema reset: %w", err)
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

func recreateSQLiteIndexes(tx *sqlx.Tx) error {
	indexesByTable := map[string][]string{
		"tasks": {
			`CREATE INDEX IF NOT EXISTS idx_tasks_stage_created_at ON tasks (stage, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_stage_status_created_at ON tasks (stage_status, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks (updated_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_download_completed_at ON tasks (download_completed_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_torrent_hash ON tasks (torrent_hash)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_selected_info_hash ON tasks (selected_info_hash)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_stash_scan_job_id ON tasks (stash_scan_job_id)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_scan_queue ON tasks (updated_at DESC) WHERE stage = 'PENDING_INGEST' AND stage_status = 'PENDING' AND stash_scan_job_id IS NULL`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_code_unique ON tasks (code) WHERE code <> ''`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_torrent_identity_hash_unique ON tasks (torrent_identity_hash) WHERE torrent_identity_hash IS NOT NULL`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_torrent_identity_magnet_unique ON tasks (torrent_identity_magnet) WHERE torrent_identity_magnet IS NOT NULL`,
		},
		"task_events": {
			`CREATE INDEX IF NOT EXISTS idx_task_events_task_created_at ON task_events (task_id, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_task_events_created_at ON task_events (created_at DESC)`,
		},
		"subscription_performer_state": {
			`CREATE INDEX IF NOT EXISTS idx_subscription_state_last_checked_at ON subscription_performer_state (last_checked_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_state_updated_at ON subscription_performer_state (updated_at DESC)`,
		},
		"subscription_release_entities": {
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_release_entities_key ON subscription_release_entities (release_key)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_date ON subscription_release_entities (release_date DESC, seen_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_code ON subscription_release_entities (code)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_status_seen_at ON subscription_release_entities (status, seen_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_task_id ON subscription_release_entities (task_id)`,
		},
		"subscription_performer_releases": {
			`CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_performer_linked_at ON subscription_performer_releases (performer_id, linked_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_release_id ON subscription_performer_releases (release_id)`,
		},
	}

	for table, indexes := range indexesByTable {
		exists, err := sqliteTableExistsTx(tx, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		for _, query := range indexes {
			if _, err := tx.Exec(query); err != nil {
				return fmt.Errorf("downloader: create sqlite index: %w", err)
			}
		}
	}
	return nil
}

func clearDanglingSubscriptionTaskReferences(tx *sqlx.Tx) error {
	exists, err := sqliteTableExistsTx(tx, "subscription_release_entities")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
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
