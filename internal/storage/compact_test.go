package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestCompactionReclaimsSpaceWithoutLosingRows(t *testing.T) {
	ctx := context.Background()
	d := testDatabase(t)
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TABLE retained(id INTEGER PRIMARY KEY,data BLOB); INSERT INTO retained VALUES(1,zeroblob(1048576)),(2,zeroblob(1048576)); DELETE FROM retained WHERE id=2`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := d.Backup(ctx, t.TempDir(), "before-compaction", true)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Protected {
		t.Fatal("missing protection")
	}
	result, err := d.Compact(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.AfterBytes >= result.BeforeBytes {
		t.Fatalf("space not reclaimed: %+v", result)
	}
	var size int
	if err := d.Reader.Get(&size, "SELECT length(data) FROM retained WHERE id=1"); err != nil || size != 1048576 {
		t.Fatal("retained row changed", err)
	}
}
func TestRestoreUnderstandsOlderKnownSchemaPrefix(t *testing.T) {
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	future := append(CoreMigrations(), Migration{Version: len(CoreMigrations()) + 1, Name: "future known change", SQL: "CREATE TABLE future(id TEXT)"})
	if err := validateCoreBundle(ctx, m.Core.Path, future, true); err != nil {
		t.Fatal("known older schema rejected", err)
	}
	if err := validateCoreBundle(ctx, m.Core.Path, future, false); err == nil {
		t.Fatal("published a generation before required upgrades")
	}
	if err := validateCoreBundle(ctx, filepath.Join(filepath.Dir(m.Core.Path), "runtime.db"), CoreMigrations(), true); err == nil {
		t.Fatal("runtime accepted as core")
	}
}
