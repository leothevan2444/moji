package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestResourceDecisionMigrationPreservesAcceptedTasksWithoutInventingReports(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, Options{Path: filepath.Join(t.TempDir(), "core.db"), Durability: Durable})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx, CoreMigrations()[:9]); err != nil {
		t.Fatal(err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('kept','SEARCH','ABCD-123','DOWNLOADING','RUNNING','2026-01-01','2026-01-01')`)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO task_checkpoints(task_id,sourcing_request_json) VALUES('kept','{"version":1}')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Migrate(ctx, CoreMigrations()); err != nil {
		t.Fatal(err)
	}
	var report sql.NullString
	var request string
	if err = db.Reader.QueryRowContext(ctx, `SELECT selection_report_json,sourcing_request_json FROM task_checkpoints WHERE task_id='kept'`).Scan(&report, &request); err != nil || report.Valid || request != `{"version":1}` {
		t.Fatal("migration lost request or invented an old decision", err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE task_checkpoints SET selection_report_json=? WHERE task_id='kept'`, `{"version":1,"selected":"`+strings.Repeat("x", 262144)+`"}`)
		return err
	}); err == nil {
		t.Fatal("oversized persisted report accepted")
	}
}

func TestRemoveMetadataPreviewJobsPreservesSearchRuns(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, Options{Path: filepath.Join(t.TempDir(), "runtime.db"), Durability: Durable})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx, RuntimeMigrations()[:8]); err != nil {
		t.Fatal(err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO torrent_search_runs(id,payload_version,state,query_json,correlation_id,created_at,updated_at,expires_at) VALUES('kept-run',1,'COMPLETED','{}','kept','2026-01-01','2026-01-01','2027-01-01')`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO torrent_selection_previews(id,search_id,state,request_json,created_at,expires_at) VALUES('removed-job','kept-run','QUEUED','{}','2026-01-01','2027-01-01')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var before int
	if err = db.Reader.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='torrent_selection_previews'").Scan(&before); err != nil || before != 1 {
		t.Fatal("old preview table missing", err)
	}
	if err = db.Migrate(ctx, RuntimeMigrations()); err != nil {
		t.Fatal(err)
	}
	var remaining, searchRuns int
	if err = db.Reader.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='torrent_selection_previews'").Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("obsolete preview table retained", err)
	}
	if err = db.Reader.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='torrent_search_runs'").Scan(&searchRuns); err != nil || searchRuns != 1 {
		t.Fatal("search table removed", err)
	}
	var preserved string
	if err = db.Reader.QueryRowContext(ctx, `SELECT state FROM torrent_search_runs WHERE id='kept-run'`).Scan(&preserved); err != nil || preserved != "COMPLETED" {
		t.Fatal("search snapshot lost", err)
	}

}
