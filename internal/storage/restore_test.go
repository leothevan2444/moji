package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestRestoreGenerationKeepsPreviousAndRebuildsDerivedData(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('saved','MANUAL','TEST-123','DOWNLOADING','RUNNING','2026-10-27','2026-10-27')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	saved, err := m.Core.Backup(ctx, m.BackupDirectory, "restore-test", true)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(m.BackupDirectory, saved.Database+".json")
	if _, err := RestoreCore(ctx, root, manifest); err == nil {
		t.Fatal("restored over active instance")
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec("UPDATE tasks SET code='CHANGED-456'"); return err }); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreCore(ctx, root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if restored.PreviousBackup == nil {
		t.Fatal("old state was not protected")
	}
	opened, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	var code string
	if err := opened.Core.Reader.Get(&code, "SELECT code FROM tasks WHERE id='saved'"); err != nil || code != "TEST-123" {
		t.Fatalf("restored code=%s err=%v", code, err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	previous, err := VerifyBackup(ctx, filepath.Join(root, "backups", restored.PreviousBackup.Database+".json"))
	if err != nil || !previous.Protected {
		t.Fatal("old backup invalid", err)
	}
	// A missing active core is recoverable explicitly, but normal startup refuses it.
	if err := os.Remove(filepath.Join(restored.Directory, "moji.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreCore(ctx, root, manifest); err != nil {
		t.Fatal("could not recover missing active core", err)
	}
}

func TestRestorePreservesDamagedCoreBeforeRecovery(t *testing.T) {
	ctx := context.Background()
	source := t.TempDir()
	m, err := OpenManager(ctx, DefaultManagerOptions(source))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := m.Core.Backup(ctx, m.BackupDirectory, "recovery", true)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(m.BackupDirectory, saved.Database+".json")
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	damaged := t.TempDir()
	original := []byte("damaged SQLite header with irreplaceable original bytes")
	if err := os.Mkdir(filepath.Join(damaged, "broken"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(damaged, layoutFile), []byte("broken\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(damaged, "broken", "moji.db"), original, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreCore(ctx, damaged, manifest)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(restored.PreviousRawBackup, "moji.db"))
	if err != nil || string(raw) != string(original) {
		t.Fatal("damaged original not preserved", err)
	}
	healthy, err := OpenManager(ctx, DefaultManagerOptions(damaged))
	if err != nil {
		t.Fatal(err)
	}
	healthy.Close()
}
