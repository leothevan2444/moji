package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestBackupIncludesAndVerifiesReferencedHistory(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	archive := filepath.Join(root, "history.gz")
	if err := os.WriteFile(archive, []byte("immutable historical events"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, _, err := fileDigest(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO history_archives(id,path,sha256,source_rows,retained_rows,created_at) VALUES('history','history.gz',?,100,0,'2026-10-27')`, digest)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	saved, err := m.Core.Backup(ctx, m.BackupDirectory, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Artifacts) != 1 {
		t.Fatal("referenced history missing from backup")
	}
	manifest := filepath.Join(m.BackupDirectory, saved.Database+".json")
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackup(ctx, manifest); err != nil {
		t.Fatal("backup still depends on live history", err)
	}
	artifact := filepath.Join(m.BackupDirectory, saved.Artifacts[0].Path)
	if err := os.WriteFile(artifact, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackup(ctx, manifest); err == nil {
		t.Fatal("corrupt referenced history accepted")
	}
}
