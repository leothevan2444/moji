package downloader

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenSQLiteDatabaseResetsLegacyTasksTableToNewSchema(t *testing.T) {
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
		"code",
		"delivery_mode",
		"moji_source_path",
		"transfer_action",
		"moji_transfer_path",
		"transfer_error",
		"stash_scan_path",
		"stash_scan_hint",
		"torrent_identity_hash",
		"torrent_identity_magnet",
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
	if version != "7" {
		t.Fatalf("expected schema version 7, got %q", version)
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
	if version != "7" {
		t.Fatalf("expected schema version 7, got %q", version)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected sqlite file to exist: %v", err)
	}
}

func TestSQLiteTaskStoreRejectsDuplicateBusinessKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteTaskStore failed: %v", err)
	}

	now := time.Unix(100, 0).UTC()
	baseTask := &Task{
		ID:                    "task-1",
		Code:                  "SONE-000",
		Stage:                 TaskStageDownloading,
		StageStatus:           TaskStageStatusRunning,
		TorrentIdentityHash:   "HASH-ONE",
		TorrentIdentityMagnet: "magnet:?xt=urn:btih:HASH-ONE",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := store.Create(context.Background(), baseTask); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.Create(context.Background(), &Task{
		ID:                    "task-2",
		Code:                  "SONE-000",
		Stage:                 TaskStageDownloading,
		StageStatus:           TaskStageStatusRunning,
		TorrentIdentityHash:   "HASH-TWO",
		TorrentIdentityMagnet: "magnet:?xt=urn:btih:HASH-TWO",
		CreatedAt:             now.Add(time.Second),
		UpdatedAt:             now.Add(time.Second),
	})
	if !errors.Is(err, ErrDuplicateCodeTask) {
		t.Fatalf("expected duplicate code error, got %v", err)
	}

	err = store.Create(context.Background(), &Task{
		ID:                    "task-3",
		Code:                  "SONE-001",
		Stage:                 TaskStageDownloading,
		StageStatus:           TaskStageStatusRunning,
		TorrentIdentityHash:   "HASH-ONE",
		TorrentIdentityMagnet: "magnet:?xt=urn:btih:HASH-ONE",
		CreatedAt:             now.Add(2 * time.Second),
		UpdatedAt:             now.Add(2 * time.Second),
	})
	if !errors.Is(err, ErrDuplicateTorrentTask) {
		t.Fatalf("expected duplicate torrent error, got %v", err)
	}
}

func TestOpenSQLiteDatabaseResetsLegacyRowsOnSchemaMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nullable-migration.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	legacySchema := `
CREATE TABLE task_store_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;
CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT 'MANUAL',
  query TEXT NOT NULL,
  code TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  torrent_url TEXT NOT NULL DEFAULT '',
  save_path TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',
  torrent_identity_hash TEXT NOT NULL DEFAULT '',
  torrent_identity_magnet TEXT NOT NULL DEFAULT '',
  torrent_hash TEXT NOT NULL DEFAULT '',
  torrent_name TEXT NOT NULL DEFAULT '',
  progress REAL NOT NULL DEFAULT 0,
  qbittorrent_state TEXT NOT NULL DEFAULT '',
  content_path TEXT NOT NULL DEFAULT '',
  completed_at TEXT,
  stash_mode TEXT NOT NULL DEFAULT '',
  stash_source_path TEXT NOT NULL DEFAULT '',
  stash_transfer_action TEXT NOT NULL DEFAULT '',
  stash_transfer_path TEXT NOT NULL DEFAULT '',
  stash_transfer_status TEXT NOT NULL DEFAULT '',
  stash_transfer_error TEXT NOT NULL DEFAULT '',
  stash_job_id TEXT NOT NULL DEFAULT '',
  stash_scan_path TEXT NOT NULL DEFAULT '',
  stash_scan_status TEXT NOT NULL DEFAULT '',
  stash_scan_error TEXT NOT NULL DEFAULT '',
  stash_scan_hint TEXT NOT NULL DEFAULT '',
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
CREATE TABLE subscription_performer_state (
  performer_id TEXT PRIMARY KEY,
  last_checked_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE subscription_release_entities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  release_key TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  source TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  code TEXT NOT NULL DEFAULT '',
  release_date TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  query TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  performer_count INTEGER NOT NULL DEFAULT 0,
  performer_names TEXT NOT NULL DEFAULT '[]',
  classification TEXT NOT NULL DEFAULT 'UNKNOWN',
  decision TEXT NOT NULL DEFAULT 'QUEUED',
  decision_reason TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  seen_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE subscription_performer_releases (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  performer_id TEXT NOT NULL,
  release_id INTEGER NOT NULL,
  linked_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;
INSERT INTO task_store_meta (key, value) VALUES ('schema_version', '3');
`
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	now := time.Unix(200, 0).UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
INSERT INTO tasks (
  id, source, query, code, status, torrent_url, save_path, category, tags,
  torrent_identity_hash, torrent_identity_magnet, torrent_hash, torrent_name, progress,
  qbittorrent_state, content_path, stash_mode, stash_source_path, stash_transfer_action,
  stash_transfer_path, stash_transfer_status, stash_transfer_error, stash_job_id, stash_scan_path,
  stash_scan_status, stash_scan_error, stash_scan_hint, error, created_at, updated_at
) VALUES (?, 'SEARCH', ?, ?, 'pending', ?, '', '', '', '', '', '', '', 0, '', '', '', '', '', '', '', '', '', '', '', '', '', '', ?, ?)`,
		"task-nullable",
		"ABCD-123",
		"ABCD-123",
		"magnet:?xt=urn:btih:null-task",
		now,
		now,
	); err != nil {
		t.Fatalf("insert legacy task: %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO subscription_performer_state (performer_id, last_checked_at, last_error, created_at, updated_at)
VALUES ('performer-1', NULL, '', ?, ?)`, now, now); err != nil {
		t.Fatalf("insert legacy performer state: %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, release_date, url, query, task_id,
  performer_count, performer_names, classification, decision, decision_reason, last_error, seen_at, created_at, updated_at
) VALUES ('release-1', 'pending', 'stash-box:test', 'Title', 'ABCD-123', '', '', 'ABCD-123', '', 1, '[]', 'UNKNOWN', 'QUEUED', '', '', ?, ?, ?)`,
		now, now, now,
	); err != nil {
		t.Fatalf("insert legacy release entity: %v", err)
	}

	_ = db.Close()

	opened, err := OpenSQLiteDatabase(path)
	if err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	defer opened.Close()

	var taskCount int
	if err := opened.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&taskCount); err != nil {
		t.Fatalf("count tasks after reset: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected legacy tasks to be dropped during schema reset, got %d", taskCount)
	}
	var releaseCount int
	if err := opened.QueryRow(`SELECT COUNT(*) FROM subscription_release_entities`).Scan(&releaseCount); err != nil {
		t.Fatalf("count subscription releases after reset: %v", err)
	}
	if releaseCount != 0 {
		t.Fatalf("expected legacy subscription releases to be dropped during schema reset, got %d", releaseCount)
	}
}

func TestSubscriptionPerformerReleasesUsesCompositePrimaryKey(t *testing.T) {
	db, err := OpenSQLiteDatabase(filepath.Join(t.TempDir(), "subscription-links.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	now := time.Unix(300, 0).UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO subscription_performer_state (performer_id, created_at, updated_at) VALUES (?, ?, ?)`, "performer-1", now, now); err != nil {
		t.Fatalf("insert performer state: %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, performer_count, performer_names, classification, decision, decision_reason, seen_at, created_at, updated_at
) VALUES (?, 'pending', 'stash-box:test', 'Title', 'ABCD-123', 1, '[]', 'UNKNOWN', 'QUEUED', '', ?, ?, ?)`,
		"release-1", now, now, now,
	); err != nil {
		t.Fatalf("insert release entity: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at) VALUES ('performer-1', 1, ?)`, now); err != nil {
		t.Fatalf("insert first performer release link: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at) VALUES ('performer-1', 1, ?)`, now); err == nil {
		t.Fatal("expected duplicate composite key insert to fail")
	}
}
