package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestCooldownMigrationPreservesExistingRuntimeState(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, Options{Path: filepath.Join(t.TempDir(), "runtime.db"), Durability: Rebuildable})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx, RuntimeMigrations()[:5]); err != nil {
		t.Fatal(err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.ExecContext(ctx, `INSERT INTO indexer_resources(id,instance_id,payload_version,payload_json,expires_at,created_at) VALUES('kept','site',1,'{}','2099-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
		return e
	}); err != nil {
		t.Fatal(err)
	}

	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO torrent_search_runs(id,payload_version,state,query_json,correlation_id,created_at,updated_at,expires_at) VALUES('kept-search',1,'COMPLETED','{}','synthetic-correlation','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','2099-01-01T00:00:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Migrate(ctx, RuntimeMigrations()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Reader.GetContext(ctx, &count, `SELECT COUNT(*) FROM indexer_resources WHERE id='kept'`); err != nil || count != 1 {
		t.Fatal("migration lost resource state", err)
	}
	if err = db.Reader.GetContext(ctx, &count, `SELECT COUNT(*) FROM torrent_search_runs WHERE id='kept-search' AND state='COMPLETED'`); err != nil || count != 1 {
		t.Fatal("migration lost existing search", err)
	}
	if err = db.Reader.GetContext(ctx, &count, `SELECT COUNT(*) FROM indexer_network_cooldowns`); err != nil || count != 0 {
		t.Fatal("cooldown table not initialized", err)
	}
	if err = db.Migrate(ctx, RuntimeMigrations()); err != nil {
		t.Fatal("repeat migration", err)
	}
}
