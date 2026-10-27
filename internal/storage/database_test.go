package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func testDatabase(t *testing.T) *Database {
	t.Helper()
	d, err := Open(context.Background(), Options{Path: filepath.Join(t.TempDir(), "core ?# space.sqlite")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

var testMigrations = []Migration{{1, "base", `CREATE TABLE parent(id INTEGER PRIMARY KEY,value TEXT NOT NULL); CREATE TABLE child(id INTEGER PRIMARY KEY,parent_id INTEGER NOT NULL REFERENCES parent(id));`}}

func seedDatabase(t *testing.T, d *Database) {
	t.Helper()
	if err := d.Migrate(context.Background(), testMigrations); err != nil {
		t.Fatal(err)
	}
}

func TestEngineAndPerConnectionProtection(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	seedDatabase(t, d)
	var version string
	if err := d.Reader.GetContext(ctx, &version, "SELECT sqlite_version()"); err != nil {
		t.Fatal(err)
	}
	if version != "3.51.3" {
		t.Fatalf("verify engine when upgrading dependency: got %s", version)
	}
	// Keep four leases simultaneously, proving settings apply to every connection.
	var leases []*sqlx.Conn
	for i := 0; i < 4; i++ {
		conn, err := d.Reader.Connx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		leases = append(leases, conn)
		var readonly int
		if err := conn.GetContext(ctx, &readonly, "PRAGMA query_only"); err != nil || readonly != 1 {
			t.Fatalf("read connection not protected: %d %v", readonly, err)
		}
	}
	for _, conn := range leases {
		if _, err := conn.ExecContext(ctx, "INSERT INTO parent VALUES(1,'bad')"); err == nil {
			t.Fatal("reader accepted write")
		}
		conn.Close()
	}
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.ExecContext(ctx, "INSERT INTO child VALUES(1,999)"); return err }); err == nil {
		t.Fatal("writer ignored foreign key constraint")
	}
}
func TestMigrationRollbackAndUnknownVersionPreserveData(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	seedDatabase(t, d)
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO parent VALUES(1,'important')")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	failed := append(append([]Migration{}, testMigrations...), Migration{2, "broken", "CREATE TABLE transient(id INTEGER); INSERT INTO missing_table VALUES(1)"})
	if err := d.Migrate(ctx, failed); err == nil {
		t.Fatal("invalid migration succeeded")
	}
	var count int
	d.Reader.GetContext(ctx, &count, "SELECT count(*) FROM sqlite_master WHERE name='transient'")
	if count != 0 {
		t.Fatal("DDL escaped migration rollback")
	}
	altered := []Migration{{1, "changed", "CREATE TABLE other(id INTEGER)"}}
	if err := d.Migrate(ctx, altered); !errors.Is(err, ErrMigrationChanged) {
		t.Fatalf("changed migration accepted: %v", err)
	}
	if err := d.Migrate(ctx, nil); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("newer schema accepted: %v", err)
	}
	var value string
	if err := d.Reader.GetContext(ctx, &value, "SELECT value FROM parent WHERE id=1"); err != nil || value != "important" {
		t.Fatal("important record changed", err)
	}
}
func TestUnversionedDatabaseIsNotAdoptedSilently(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, "CREATE TABLE legacy(id INTEGER); INSERT INTO legacy VALUES(42)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.Migrate(ctx, testMigrations); !errors.Is(err, ErrUnrecognizedSchema) {
		t.Fatalf("unrecognized database was initialized: %v", err)
	}
	var id int
	if err := d.Reader.GetContext(ctx, &id, "SELECT id FROM legacy"); err != nil || id != 42 {
		t.Fatal("legacy data modified", err)
	}
}
func TestReadersProceedDuringWriteAndAdmissionIsBounded(t *testing.T) {
	d, err := Open(context.Background(), Options{Path: filepath.Join(t.TempDir(), "busy.db"), WriteQueue: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	seedDatabase(t, d)
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- d.WithWrite(context.Background(), func(tx *sqlx.Tx) error {
			_, err := tx.Exec("INSERT INTO parent VALUES(1,'uncommitted')")
			close(entered)
			<-release
			return err
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var count int
	if err := d.Reader.GetContext(ctx, &count, "SELECT count(*) FROM parent"); err != nil || count != 0 {
		t.Fatalf("reader blocked or saw uncommitted value: %d %v", count, err)
	}
	err = d.WithWrite(ctx, func(*sqlx.Tx) error { return nil })
	if !errors.Is(err, ErrWriteQueueFull) {
		t.Fatalf("queue not bounded: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if d.Metrics().Writes != 2 {
		t.Fatalf("write metrics wrong: %+v", d.Metrics())
	}
}
func TestConcurrentUpdatesAreSerialized(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	seedDatabase(t, d)
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error { _, e := tx.Exec("INSERT INTO parent VALUES(1,'0')"); return e }); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 20; n++ {
				if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
					var n int
					if e := tx.Get(&n, "SELECT value FROM parent WHERE id=1"); e != nil {
						return e
					}
					_, e := tx.Exec("UPDATE parent SET value=? WHERE id=1", n+1)
					return e
				}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	var value int
	if err := d.Reader.Get(&value, "SELECT value FROM parent WHERE id=1"); err != nil || value != 240 {
		t.Fatalf("lost update: %d %v", value, err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.WithWrite(ctx, func(*sqlx.Tx) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("write after close: %v", err)
	}
}
func TestBackupIncludesCommittedWALAndDetectsTampering(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	seedDatabase(t, d)
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error { _, e := tx.Exec("INSERT INTO parent VALUES(1,'recovery')"); return e }); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	manifest, err := d.Backup(ctx, dir, "before-migration", true)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Tables["parent"] != 1 || !manifest.Protected || manifest.Engine != "3.51.3" {
		t.Fatalf("wrong backup manifest: %+v", manifest)
	}
	path := filepath.Join(dir, manifest.Database+".json")
	if _, err := VerifyBackup(ctx, path); err != nil {
		t.Fatal(err)
	}
	// A source mutation must not alter an already verified standalone backup.
	d.WithWrite(ctx, func(tx *sqlx.Tx) error { _, e := tx.Exec("DELETE FROM parent"); return e })
	if _, err := VerifyBackup(ctx, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), manifest.SHA256, strings.Repeat("0", 64), 1))
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackup(ctx, path); err == nil {
		t.Fatal("tampered checksum accepted")
	}
}
