package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestRepairDerivedProtectsDamagedStateAndKeepsCoreBytes(t *testing.T) {
	for _, role := range []string{"runtime", "cache"} {
		t.Run(role, func(t *testing.T) {
			ctx := context.Background()
			root, _ := closedInstance(t)
			m, err := OpenManager(ctx, DefaultManagerOptions(root))
			if err != nil {
				t.Fatal(err)
			}
			if err = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
				_, e := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('keep','MANUAL','KEEP-123','DOWNLOADING','RUNNING','now','now')`)
				return e
			}); err != nil {
				t.Fatal(err)
			}
			if _, err = RepairDerived(ctx, root, role); !errors.Is(err, ErrDirectoryInUse) {
				t.Fatalf("repair allowed with live owner: %v", err)
			}
			target := m.Runtime.Path
			if role == "cache" {
				target = m.Cache.Path
			}
			m.Close()
			before, _, err := fileDigest(m.Core.Path)
			if err != nil {
				t.Fatal(err)
			}
			damaged := []byte("damaged derived database")
			if err = os.WriteFile(target, damaged, 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(target+"-wal", []byte("damaged WAL"), 0600); err != nil {
				t.Fatal(err)
			}
			saved, err := RepairDerived(ctx, root, role)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(saved, "derived", role))
			if err != nil || string(raw) != string(damaged) {
				t.Fatal("derived original not protected", err)
			}
			after, _, err := fileDigest(m.Core.Path)
			if err != nil || before != after {
				t.Fatal("repair changed core", err)
			}
			m, err = OpenManager(ctx, DefaultManagerOptions(root))
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			var code string
			if err = m.Core.Reader.Get(&code, "SELECT code FROM tasks WHERE id='keep'"); err != nil || code != "KEEP-123" {
				t.Fatal("lost durable data", err)
			}
		})
	}
}

func TestRepairDerivedRejectsUnsafePathsAndRoles(t *testing.T) {
	root, active := closedInstance(t)
	if _, err := RepairDerived(context.Background(), root, "core"); err == nil {
		t.Fatal("core marked disposable")
	}
	if err := os.RemoveAll(filepath.Join(active, "cache")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(active, "cache")); err != nil {
		t.Fatal(err)
	}
	if _, err := RepairDerived(context.Background(), root, "cache"); err == nil {
		t.Fatal("repair followed symlink")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("repair touched outside generation", err)
	}
}
