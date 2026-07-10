package subscription

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/taskruntime"
	_ "modernc.org/sqlite"
)

func TestNewSQLiteStoreInitializesSubscriptionTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscription-only.db")

	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.db.Close()

	for _, table := range []string{
		"task_store_meta",
		"tasks",
		"task_events",
		"subscription_performer_state",
		"subscription_release_entities",
		"subscription_performer_releases",
	} {
		exists, err := sqliteTableExists(store.db, table)
		if err != nil {
			t.Fatalf("check %s existence: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected %s to exist", table)
		}
	}
}

func TestNewSQLiteStoreUpgradesLegacySubscriptionSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-subscription.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Unix(400, 0).UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
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
INSERT INTO subscription_performer_state (performer_id, created_at, updated_at) VALUES ('performer-1', ?, ?);
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, release_date, url, query, task_id,
  performer_count, performer_names, classification, decision, decision_reason, last_error, seen_at, created_at, updated_at
) VALUES ('release-1', 'pending', 'stash-box:test', 'Title', 'ABCD-123', '', '', 'ABCD-123', '', 1, '[]', 'UNKNOWN', 'QUEUED', '', '', ?, ?, ?);
INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at, created_at, updated_at)
VALUES ('performer-1', 1, ?, ?, ?);
`, now, now, now, now, now, now, now); err != nil {
		t.Fatalf("seed legacy subscription schema: %v", err)
	}
	_ = db.Close()

	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore upgrade failed: %v", err)
	}
	defer store.db.Close()

	hasQuery, err := sqliteColumnExists(store.db, "subscription_release_entities", "query")
	if err != nil {
		t.Fatalf("check legacy query column: %v", err)
	}
	if hasQuery {
		t.Fatal("expected legacy query column to be removed")
	}

	hasID, err := sqliteColumnExists(store.db, "subscription_performer_releases", "id")
	if err != nil {
		t.Fatalf("check legacy performer_release id column: %v", err)
	}
	if hasID {
		t.Fatal("expected legacy performer_release id column to be removed")
	}

	var releaseCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM subscription_release_entities`).Scan(&releaseCount); err != nil {
		t.Fatalf("count release entities: %v", err)
	}
	if releaseCount != 0 {
		t.Fatalf("expected legacy subscription rows to be reset, got %d", releaseCount)
	}

	if _, err := store.db.Exec(`INSERT INTO subscription_performer_state (performer_id, created_at, updated_at) VALUES ('performer-1', ?, ?)`, now, now); err != nil {
		t.Fatalf("insert upgraded performer state: %v", err)
	}
	if _, err := store.db.Exec(`
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, performer_count, performer_names, classification, decision, decision_reason, seen_at, created_at, updated_at
) VALUES ('release-1', 'pending', 'stash-box:test', 'Title', 'ABCD-123', 1, '[]', '', '', '', ?, ?, ?)`, now, now, now); err != nil {
		t.Fatalf("insert upgraded release entity: %v", err)
	}
	if _, err := store.db.Exec(`INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at) VALUES ('performer-1', 1, ?)`, now); err != nil {
		t.Fatalf("insert upgraded performer-release link: %v", err)
	}
	if _, err := store.db.Exec(`INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at) VALUES ('performer-1', 1, ?)`, now); err == nil {
		t.Fatal("expected composite primary key to reject duplicate performer-release link")
	}
}

func TestNewSQLiteStoreClearsDanglingTaskReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dangling-links.db")

	db, err := taskruntime.OpenSQLiteDatabase(path)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase failed: %v", err)
	}
	defer db.Close()

	now := time.Unix(500, 0).UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	if _, err := db.Exec(`
CREATE TABLE subscription_performer_state (
  performer_id TEXT PRIMARY KEY,
  last_checked_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE subscription_release_entities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  release_key TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'discovered',
  source TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  code TEXT NOT NULL DEFAULT '',
  release_date TEXT,
  url TEXT,
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
) STRICT;
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, task_id, performer_count, performer_names, classification, decision, decision_reason, seen_at, created_at, updated_at
) VALUES ('release-1', 'pending', 'stash-box:test', 'Title', 'ABCD-123', 'missing-task', 1, '[]', '', '', '', ?, ?, ?)`, now, now, now); err != nil {
		t.Fatalf("seed dangling release task reference: %v", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.db.Close()

	var taskID sql.NullString
	if err := store.db.QueryRow(`SELECT task_id FROM subscription_release_entities WHERE release_key = 'release-1'`).Scan(&taskID); err != nil {
		t.Fatalf("query release task_id: %v", err)
	}
	if taskID.Valid {
		t.Fatalf("expected dangling task_id to be cleared, got %q", taskID.String)
	}
}

func TestSQLiteStorePutWorksWithoutTasksTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscription-no-tasks.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlite")
	if err := ensureSubscriptionSQLiteSchema(sqlxDB); err != nil {
		t.Fatalf("ensureSubscriptionSQLiteSchema failed: %v", err)
	}
	_ = sqlxDB.Close()

	store := &SQLiteStore{db: sqlx.MustConnect("sqlite", path)}
	defer store.db.Close()

	now := time.Unix(600, 0).UTC()
	state := &PerformerState{
		PerformerID: "performer-1",
		PendingReleases: []RecordedRelease{
			{
				Key:            "release-1",
				Source:         "stash-box:test",
				Title:          "Title",
				Code:           "ABCD-123",
				TaskID:         "missing-task",
				SeenAt:         now,
				PerformerCount: 1,
				PerformerNames: []string{"Name"},
			},
		},
	}
	if err := store.Put(context.Background(), state); err != nil {
		t.Fatalf("Put failed without tasks table: %v", err)
	}

	loaded, err := store.Get(context.Background(), "performer-1")
	if err != nil {
		t.Fatalf("Get failed without tasks table: %v", err)
	}
	if loaded == nil || len(loaded.PendingReleases) != 1 {
		t.Fatalf("expected one pending release, got %#v", loaded)
	}
	if loaded.PendingReleases[0].TaskID != "" {
		t.Fatalf("expected missing task reference to be dropped, got %q", loaded.PendingReleases[0].TaskID)
	}
}

func TestNewSQLiteStoreAndTaskRuntimeCanInitializeInAnyOrder(t *testing.T) {
	t.Run("subscription then task runtime", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "subscription-first.db")

		store, err := NewSQLiteStore(path)
		if err != nil {
			t.Fatalf("NewSQLiteStore failed: %v", err)
		}
		_ = store.db.Close()

		db, err := taskruntime.OpenSQLiteDatabase(path)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase failed: %v", err)
		}
		defer db.Close()

		for _, table := range []string{"tasks", "subscription_release_entities"} {
			exists, err := sqliteTableExists(db, table)
			if err != nil {
				t.Fatalf("check %s existence: %v", table, err)
			}
			if !exists {
				t.Fatalf("expected %s to exist", table)
			}
		}
	})

	t.Run("task runtime then subscription", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "taskruntime-first.db")

		db, err := taskruntime.OpenSQLiteDatabase(path)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase failed: %v", err)
		}
		_ = db.Close()

		store, err := NewSQLiteStore(path)
		if err != nil {
			t.Fatalf("NewSQLiteStore failed: %v", err)
		}
		defer store.db.Close()

		for _, table := range []string{"tasks", "subscription_release_entities"} {
			exists, err := sqliteTableExists(store.db, table)
			if err != nil {
				t.Fatalf("check %s existence: %v", table, err)
			}
			if !exists {
				t.Fatalf("expected %s to exist", table)
			}
		}
	})
}

func TestSQLiteStoreDeleteOrphanReleaseEntities(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store-roundtrip.db")

	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.db.Close()

	now := time.Unix(700, 0).UTC()
	state := &PerformerState{
		PerformerID: "performer-1",
		PendingReleases: []RecordedRelease{
			{
				Key:            "release-1",
				Source:         "stash-box:test",
				Title:          "Title",
				Code:           "ABCD-123",
				SeenAt:         now,
				PerformerCount: 1,
				PerformerNames: []string{"Name"},
			},
		},
	}
	if err := store.Put(context.Background(), state); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := store.Delete(context.Background(), "performer-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	loaded, err := store.Get(context.Background(), "performer-1")
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected performer state to be deleted, got %#v", loaded)
	}

	var releaseCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM subscription_release_entities`).Scan(&releaseCount); err != nil {
		t.Fatalf("count release entities after delete: %v", err)
	}
	if releaseCount != 0 {
		t.Fatalf("expected orphan release entities to be removed, got %d", releaseCount)
	}
}

func TestSQLiteStoreGetMissingReturnsNil(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "missing.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.db.Close()

	state, err := store.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state for missing performer, got %#v", state)
	}
}
