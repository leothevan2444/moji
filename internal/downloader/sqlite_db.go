package downloader

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed sqlite_schema.sql
var sqliteSchema string

func OpenSQLiteDatabase(path string) (*sql.DB, error) {
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

	db, err := sql.Open("sqlite", trimmed)
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

func configureSQLiteDatabase(db *sql.DB) error {
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

	if _, err := db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("downloader: initialize sqlite schema: %w", err)
	}
	if err := migrateSQLiteDatabase(db); err != nil {
		return err
	}

	return nil
}

type sqliteColumnMigration struct {
	name       string
	definition string
}

func migrateSQLiteDatabase(db *sql.DB) error {
	taskMigrations := []sqliteColumnMigration{
		{name: "stash_mode", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_source_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_transfer_action", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_transfer_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_transfer_status", definition: "TEXT NOT NULL DEFAULT '' CHECK (stash_transfer_status IN ('', 'started', 'completed', 'failed'))"},
		{name: "stash_transfer_error", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_scan_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "stash_scan_hint", definition: "TEXT NOT NULL DEFAULT ''"},
	}
	for _, migration := range taskMigrations {
		if err := ensureSQLiteColumn(db, "tasks", migration); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`INSERT INTO task_store_meta (key, value) VALUES ('schema_version', '2')
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`); err != nil {
		return fmt.Errorf("downloader: update sqlite schema version: %w", err)
	}
	return nil
}

func ensureSQLiteColumn(db *sql.DB, table string, migration sqliteColumnMigration) error {
	exists, err := sqliteColumnExists(db, table, migration.name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, migration.name, migration.definition)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("downloader: add sqlite column %s.%s: %w", table, migration.name, err)
	}
	return nil
}

func sqliteColumnExists(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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

func readSQLiteSchemaVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow(`SELECT value FROM task_store_meta WHERE key = 'schema_version'`).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
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
