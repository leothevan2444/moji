package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInterruptedPruningResumesFromSyncedIntent(t *testing.T) {
	d := testDatabase(t)
	seedDatabase(t, d)
	ctx := context.Background()
	directory := t.TempDir()
	old, err := d.Backup(ctx, directory, "old", false)
	if err != nil {
		t.Fatal(err)
	}
	newest, err := d.Backup(ctx, directory, "new", false)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(directory, old.Database+".json")
	if err := os.Rename(manifest, manifest+".pruning"); err != nil {
		t.Fatal(err)
	}
	// Simulate termination after the main snapshot was removed, before manifest.
	if err := os.Remove(filepath.Join(directory, old.Database)); err != nil {
		t.Fatal(err)
	}
	if err := PruneBackups(directory, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(manifest + ".pruning"); !os.IsNotExist(err) {
		t.Fatal("prune intent not completed", err)
	}
	if _, err := VerifyBackup(ctx, filepath.Join(directory, newest.Database+".json")); err != nil {
		t.Fatal("newest backup damaged", err)
	}
}
