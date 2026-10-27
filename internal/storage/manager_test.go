package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestNormalizedSchemasKeepDecisionsDurableAndObservationsDisposable(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	m, err := OpenManager(ctx, DefaultManagerOptions(dir))
	if err != nil {
		t.Fatal(err)
	}
	core := `INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('task','MANUAL','ABCD-123','DOWNLOADING','RUNNING','2026-10-27T00:00:00.000000000Z','2026-10-27T00:00:00.000000000Z');
 INSERT INTO task_file_plans(task_id,confirmed,keep_subtitles,pause_requested,applied) VALUES('task',1,0,0,1);
 INSERT INTO task_files(task_id,file_index,position,path,size,kind,part,selected,reason) VALUES('task',7,0,'ABCD-123.mp4',100,'MAIN',0,1,'USER_SELECTED');
 INSERT INTO external_commands(id,task_id,target_hash,kind,state,payload_json,created_at,updated_at) VALUES('delete','task','exact-hash','REMOVE','PENDING','{"delete_files":false}','now','now');
 INSERT INTO subscription_release_reviews(release_key,status,code,seen_at,record_json) VALUES('release','REJECTED','ABCD-999','now','{}');`
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error { _, e := tx.Exec(core); return e }); err != nil {
		m.Close()
		t.Fatal(err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`INSERT INTO task_observations(task_id,progress,observed_at) VALUES('task',.42,'now')`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenManager(ctx, DefaultManagerOptions(dir)); !errors.Is(err, ErrDirectoryInUse) {
		t.Fatalf("second instance allowed: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"runtime.db", filepath.Join("cache", "stashbox", "cache.db")} {
		if err := os.Remove(filepath.Join(filepath.Dir(m.Core.Path), path)); err != nil {
			t.Fatal(err)
		}
	}
	if failed, e := OpenManager(ctx, DefaultManagerOptions(dir)); e == nil {
		failed.Close()
		t.Fatal("missing derived databases silently recreated")
	}
	for _, role := range []string{"runtime", "cache"} {
		if _, e := RepairDerived(ctx, dir, role); e != nil {
			t.Fatal(e)
		}
	}
	m, err = OpenManager(ctx, DefaultManagerOptions(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for _, table := range []string{"tasks", "task_files", "external_commands", "subscription_release_reviews"} {
		var count int
		if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM "+table); err != nil || count != 1 {
			t.Fatalf("lost durable %s: %d %v", table, count, err)
		}
	}
	var count int
	if err := m.Runtime.Reader.Get(&count, "SELECT count(*) FROM task_observations"); err != nil || count != 0 {
		t.Fatalf("observations were not recreated safely: %d %v", count, err)
	}
}
func TestManagerInitializationFailureReleasesLockAndPreservesSource(t *testing.T) {
	dir := t.TempDir()
	lock, err := AcquireDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	lock.Close()
	d, err := Open(context.Background(), Options{Path: filepath.Join(dir, "moji.db")})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.WithWrite(context.Background(), func(tx *sqlx.Tx) error {
		_, e := tx.Exec("CREATE TABLE legacy(id INTEGER); INSERT INTO legacy VALUES(42)")
		return e
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()
	if _, err := OpenManager(context.Background(), DefaultManagerOptions(dir)); err == nil {
		t.Fatalf("legacy database silently initialized: %v", err)
	}
	lock, err = AcquireDirectory(dir)
	if err != nil {
		t.Fatalf("failed initialization leaked instance lock: %v", err)
	}
	lock.Close()
}
func TestRetentionProtectsOriginalsAndRequiresMatchingChecksum(t *testing.T) {
	d := testDatabase(t)
	seedDatabase(t, d)
	ctx := context.Background()
	dir := t.TempDir()
	protected, err := d.Backup(ctx, dir, "before-migration", true)
	if err != nil {
		t.Fatal(err)
	}
	// Three backups in one hour; preserve protected, newest, and no older duplicates.
	older, err := d.Backup(ctx, dir, "scheduled", false)
	if err != nil {
		t.Fatal(err)
	}
	newest, err := d.Backup(ctx, dir, "scheduled", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := PruneBackups(dir, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, protected.Database)); err != nil {
		t.Fatal("protected original was pruned", err)
	}
	if _, err := os.Stat(filepath.Join(dir, newest.Database)); err != nil {
		t.Fatal("newest backup was pruned", err)
	}
	if _, err := os.Stat(filepath.Join(dir, older.Database)); !os.IsNotExist(err) {
		t.Fatal("duplicate hourly backup retained")
	}
}
