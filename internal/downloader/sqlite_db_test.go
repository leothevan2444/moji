package downloader

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteDatabaseMigratesExistingTasksTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "moji.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	legacySchema := `
CREATE TABLE IF NOT EXISTS task_store_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  query TEXT NOT NULL,
  status TEXT NOT NULL,
  torrent_url TEXT NOT NULL DEFAULT '',
  save_path TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',
  torrent_hash TEXT NOT NULL DEFAULT '',
  torrent_name TEXT NOT NULL DEFAULT '',
  progress REAL NOT NULL DEFAULT 0,
  qbittorrent_state TEXT NOT NULL DEFAULT '',
  content_path TEXT NOT NULL DEFAULT '',
  completed_at TEXT,
  stash_job_id TEXT NOT NULL DEFAULT '',
  stash_scan_status TEXT NOT NULL DEFAULT '',
  stash_scan_error TEXT NOT NULL DEFAULT '',
  stash_scan_started_at TEXT,
  error TEXT NOT NULL DEFAULT '',
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
) STRICT;
INSERT INTO task_store_meta (key, value) VALUES ('schema_version', '1');
`
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	_ = db.Close()

	opened, err := OpenSQLiteDatabase(path)
	if err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	defer opened.Close()

	for _, column := range []string{
		"stash_mode",
		"stash_source_path",
		"stash_transfer_action",
		"stash_transfer_path",
		"stash_transfer_status",
		"stash_transfer_error",
		"stash_scan_path",
		"stash_scan_hint",
	} {
		exists, err := sqliteColumnExists(opened, "tasks", column)
		if err != nil {
			t.Fatalf("check column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("expected migrated column %s to exist", column)
		}
	}

	version, err := readSQLiteSchemaVersion(opened)
	if err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != "2" {
		t.Fatalf("expected schema version 2, got %q", version)
	}
}

func TestOpenSQLiteDatabaseInitializesNewSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	db, err := OpenSQLiteDatabase(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	version, err := readSQLiteSchemaVersion(db)
	if err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != "2" {
		t.Fatalf("expected schema version 2, got %q", version)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected sqlite file to exist: %v", err)
	}
}
