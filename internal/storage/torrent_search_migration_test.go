package storage

import (
	"context"
	"github.com/jmoiron/sqlx"
	"path/filepath"
	"testing"
)

func TestInteractiveSearchMigrationPreservesRuntimeV4ResourcesAndObservations(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.db")
	db, err := Open(ctx, Options{Path: path, Durability: Rebuildable})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Migrate(ctx, RuntimeMigrations()[:4]); err != nil {
		t.Fatal(err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexer_resources(id,instance_id,payload_version,payload_json,expires_at,created_at) VALUES('preserved','source',1,'{}','2099-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO indexer_health(instance_id,config_revision,bundle_revision,operation,started_at,finished_at,error_code,correlation_id) VALUES('source','config','bundle','SEARCH','2026-01-01T00:00:00Z','2026-01-01T00:00:01Z','','correlation')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(ctx, Options{Path: path, Durability: Rebuildable})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx, RuntimeMigrations()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Reader.GetContext(ctx, &count, `SELECT count(*) FROM indexer_resources WHERE id='preserved'`); err != nil || count != 1 {
		t.Fatal("v4 reference lost during migration", err)
	}
	if err = db.Reader.GetContext(ctx, &count, `SELECT count(*) FROM indexer_health WHERE instance_id='source'`); err != nil || count != 1 {
		t.Fatal("v4 health observation lost", err)
	}
	if err = db.Reader.GetContext(ctx, &count, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('torrent_search_runs','torrent_search_sites','torrent_search_candidates','indexer_health')`); err != nil || count != 4 {
		t.Fatal("search/health tables missing after ordered migration", count, err)
	}
}
