package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func closedInstance(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	m, err := OpenManager(context.Background(), DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	active := filepath.Dir(m.Core.Path)
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	return root, active
}

func TestFreshLayoutUsesPlainSiblingPointer(t *testing.T) {
	root, active := closedInstance(t)
	raw, err := os.ReadFile(filepath.Join(root, layoutFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != filepath.Base(active)+"\n" || filepath.Dir(active) != root {
		t.Fatalf("invalid layout %q %s", raw, active)
	}
	if _, err = os.Stat(filepath.Join(root, "moji.db")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created a flat database", err)
	}
	if _, err = os.Stat(filepath.Join(root, ".moji-storage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created old JSON pointer", err)
	}
	for _, ending := range []string{"", "\n", "\r\n"} {
		if err = os.WriteFile(filepath.Join(root, layoutFile), []byte(filepath.Base(active)+ending), 0600); err != nil {
			t.Fatal(err)
		}
		m, err := OpenManager(context.Background(), DefaultManagerOptions(root))
		if err != nil {
			t.Fatal(err)
		}
		m.Close()
	}
}

func TestExistingIncompleteLayoutsNeverInitialize(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string, string) error
	}{
		{"missing pointer", func(root, active string) error { return os.Remove(filepath.Join(root, layoutFile)) }},
		{"missing generation", func(root, active string) error {
			return os.WriteFile(filepath.Join(root, layoutFile), []byte("absent"), 0600)
		}},
		{"old JSON pointer", func(root, active string) error {
			return os.WriteFile(filepath.Join(root, layoutFile), []byte(`{"format":1,"generation":"old"}`), 0600)
		}},
		{"missing core", func(root, active string) error { return os.Remove(filepath.Join(active, "moji.db")) }},
		{"missing runtime", func(root, active string) error { return os.Remove(filepath.Join(active, "runtime.db")) }},
		{"missing cache", func(root, active string) error {
			return os.Remove(filepath.Join(active, "cache", "stashbox", "cache.db"))
		}},
		{"empty runtime", func(root, active string) error { return os.Truncate(filepath.Join(active, "runtime.db"), 0) }},
		{"symlink WAL", func(root, active string) error {
			return os.Symlink(filepath.Join(active, "runtime.db"), filepath.Join(active, "moji.db-wal"))
		}},
		{"symlink core", func(root, active string) error {
			p := filepath.Join(active, "moji.db")
			if err := os.Rename(p, p+".original"); err != nil {
				return err
			}
			return os.Symlink(p+".original", p)
		}},
		{"symlink generation", func(root, active string) error {
			if err := os.Rename(active, active+"-original"); err != nil {
				return err
			}
			return os.Symlink(active+"-original", active)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, active := closedInstance(t)
			if err := tc.mutate(root, active); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(filepath.Join(root, layoutFile))
			for i := 0; i < 2; i++ {
				m, err := OpenManager(context.Background(), DefaultManagerOptions(root))
				if err == nil {
					m.Close()
					t.Fatal("silently repaired incomplete layout")
				}
				var layout *LayoutError
				if !errors.As(err, &layout) {
					t.Fatalf("missing recovery classification: %v", err)
				}
			}
			after, _ := os.ReadFile(filepath.Join(root, layoutFile))
			if string(before) != string(after) {
				t.Fatal("startup changed pointer")
			}
			entries, _ := os.ReadDir(root)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), "initial-") && filepath.Join(root, e.Name()) != active && filepath.Join(root, e.Name()) != active+"-original" {
					t.Fatal("created replacement generation")
				}
			}
		})
	}
}

func TestInterruptedInitializationAndFlatLayoutRequireRecovery(t *testing.T) {
	for _, name := range []string{"stale-lock", "flat-database", "old-json"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			switch name {
			case "stale-lock":
				lock, err := AcquireDirectory(root)
				if err != nil {
					t.Fatal(err)
				}
				lock.Close()
			case "flat-database":
				if err := os.WriteFile(filepath.Join(root, "moji.db"), []byte("original data"), 0600); err != nil {
					t.Fatal(err)
				}
			case "old-json":
				if err := os.WriteFile(filepath.Join(root, ".moji-storage.json"), []byte(`{"format":1}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			m, err := OpenManager(context.Background(), DefaultManagerOptions(root))
			if err == nil {
				m.Close()
				t.Fatal("adopted incomplete/legacy root")
			}
			if _, err := os.Stat(filepath.Join(root, layoutFile)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("created a pointer", err)
			}
		})
	}
}

func TestPointerRejectsUnsafeNames(t *testing.T) {
	root, _ := closedInstance(t)
	for _, name := range []string{"", ".", "..", "../elsewhere", "a/b", "a\\b", "foo\nbar", "initial\n\n", " initial", "initial ", "backups", "CURRENT", ".moji-storage.lock", "{bad-json}", "x\x00y"} {
		if err := os.WriteFile(filepath.Join(root, layoutFile), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ResolveActiveDirectory(root); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestRestoreAndActivateRecoverBrokenPointers(t *testing.T) {
	for _, broken := range []string{"missing", "invalid", "absent-target"} {
		t.Run(broken, func(t *testing.T) {
			ctx := context.Background()
			root, active := closedInstance(t)
			m, err := OpenManager(ctx, DefaultManagerOptions(root))
			if err != nil {
				t.Fatal(err)
			}
			saved, err := m.Backup(ctx, "test", true)
			if err != nil {
				t.Fatal(err)
			}
			m.Close()
			pointer := filepath.Join(root, layoutFile)
			var raw []byte
			switch broken {
			case "missing":
				err = os.Remove(pointer)
			case "invalid":
				raw = []byte("{broken}")
				err = os.WriteFile(pointer, raw, 0600)
			case "absent-target":
				raw = []byte("absent")
				err = os.WriteFile(pointer, raw, 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			restored, err := RestoreCore(ctx, root, filepath.Join(root, "backups", saved.Database+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) > 0 {
				preserved, e := os.ReadFile(filepath.Join(restored.PreviousRawBackup, layoutFile))
				if e != nil || string(preserved) != string(raw) {
					t.Fatal("lost invalid pointer", e)
				}
			}
			if _, err = os.Stat(filepath.Join(active, "moji.db")); err != nil {
				t.Fatal("lost unselected original generation", err)
			}
			if _, err = ActivateGeneration(ctx, root, filepath.Base(active)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMissingHistoryAndUnknownSchemaBlockStartup(t *testing.T) {
	for _, kind := range []string{"archive", "empty-ledger", "newer-schema", "checksum", "empty-schema"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			root, active := closedInstance(t)
			d, err := Open(ctx, Options{Path: filepath.Join(active, "moji.db"), Durability: Durable})
			if err != nil {
				t.Fatal(err)
			}
			err = d.WithWrite(ctx, func(tx *sqlx.Tx) error {
				statement := map[string]string{
					"archive":      `INSERT INTO history_archives(id,path,sha256,source_rows,retained_rows,created_at) VALUES('missing','missing.gz','bad',1,0,'now')`,
					"empty-ledger": `DELETE FROM schema_migrations`,
					"newer-schema": `INSERT INTO schema_migrations VALUES(999,'future','future','now')`,
					"checksum":     `UPDATE schema_migrations SET checksum='bad' WHERE version=1`,
					"empty-schema": `DROP TABLE schema_migrations`,
				}[kind]
				_, e := tx.Exec(statement)
				return e
			})
			if err != nil {
				t.Fatal(err)
			}
			d.Close()
			m, err := OpenManager(ctx, DefaultManagerOptions(root))
			if err == nil {
				m.Close()
				t.Fatal("accepted incomplete or incompatible core")
			}
		})
	}
}

func TestFailedInitializationRequiresExplicitRecovery(t *testing.T) {
	root := t.TempDir()
	o := DefaultManagerOptions(root)
	o.RuntimeMigrations = []Migration{{Version: 1, Name: "broken", SQL: "INVALID SQL"}}
	if m, err := OpenManager(context.Background(), o); err == nil {
		m.Close()
		t.Fatal("invalid initialization succeeded")
	}
	if m, err := OpenManager(context.Background(), DefaultManagerOptions(root)); err == nil {
		m.Close()
		t.Fatal("reinitialized a failed instance")
	}
	if _, err := os.Stat(filepath.Join(root, layoutFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("published incomplete initialization", err)
	}
}
