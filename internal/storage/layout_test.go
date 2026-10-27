package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestGenerationPublicationPreservesOldUntilValidated(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	old, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('original','MANUAL','','DOWNLOADING','RUNNING','2026-10-27','2026-10-27')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	if err := PublishGeneration(ctx, root, "incomplete"); err == nil {
		t.Fatal("published missing generation")
	}
	if active, err := ResolveActiveDirectory(root); err != nil || active != root {
		t.Fatalf("failed preparation changed old layout: %s %v", active, err)
	}
	staged := filepath.Join(root, ".storage", "generations", "new")
	next, err := OpenManager(ctx, DefaultManagerOptions(staged))
	if err != nil {
		t.Fatal(err)
	}
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishGeneration(ctx, root, "new"); err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	if opened.Core.Path != filepath.Join(staged, "moji.db") {
		t.Fatal("not reading published generation")
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "moji.db")); err != nil {
		t.Fatal("old core was removed")
	}
	if err := os.Remove(filepath.Join(staged, "moji.db")); err != nil {
		t.Fatal(err)
	}
	if m, err := OpenManager(ctx, DefaultManagerOptions(root)); err == nil {
		m.Close()
		t.Fatal("missing active core silently replaced by empty database")
	}
}
