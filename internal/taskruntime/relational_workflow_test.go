package taskruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/storage"
)

func TestRelationalIngestResumesThroughBackupRestore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	store := NewRelationalTaskStore(m.Core, m.Observations)
	createIngestTask(t, store)
	files := &fakeFileOperator{}
	scanner := readyIngestScanner()
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	if err != nil {
		t.Fatal(err)
	}
	first := advanceIngest(t, service, scanner)
	if first.StashIngestPhase != StashIngestScan {
		t.Fatal("scan did not start")
	}
	second := advanceIngest(t, service, scanner)
	if second.StashGenerateJobID != "generate-1" {
		t.Fatal("generate checkpoint missing")
	}
	saved, err := m.Backup(ctx, "workflow-mid-ingest", true)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(m.BackupDirectory, saved.Database+".json")
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	// Restore into a different directory, deliberately without runtime/cache DBs.
	restoredRoot := t.TempDir()
	if _, err := storage.RestoreCore(ctx, restoredRoot, manifest); err != nil {
		t.Fatal(err)
	}
	restored, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(restoredRoot))
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	store = NewRelationalTaskStore(restored.Core, restored.Observations)
	completedAt := time.Now().UTC()
	service, err = NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files), WithClock(func() time.Time { return completedAt }))
	if err != nil {
		t.Fatal(err)
	}
	third := advanceIngest(t, service, scanner)
	if third.StashIdentifyJobID != "identify-1" {
		t.Fatal("generate work was not resumed")
	}
	done := advanceIngest(t, service, scanner)
	if done.Stage != TaskStageCompleted || done.IngestCompletedAt == nil || !done.IngestCompletedAt.Equal(completedAt) {
		t.Fatalf("restored workflow did not complete: %+v", done)
	}
	if len(scanner.requests) != 1 {
		t.Fatalf("scan was replayed after restore: %d", len(scanner.requests))
	}
	if err := service.TriggerActiveStashScans(ctx, scanner); err != nil {
		t.Fatal(err)
	}
	if len(scanner.requests) != 1 {
		t.Fatal("completed task was scanned again")
	}
}

func TestResolvedDownloadPathSurvivesRuntimeLossBeforeIngest(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	store := NewRelationalTaskStore(m.Core, m.Observations)
	createIngestTask(t, store)
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "runtime.db")); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	store = NewRelationalTaskStore(m.Core, m.Observations)
	task, err := store.Find(ctx, "ingest")
	if err != nil || task.ContentPath != "/downloads/ABCD-123.mp4" {
		t.Fatal("durable ingest input lost", task, err)
	}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	if err != nil {
		t.Fatal(err)
	}
	scanner := readyIngestScanner()
	if err := service.TriggerActiveStashScans(ctx, scanner); err != nil {
		t.Fatal(err)
	}
	task, err = store.Find(ctx, "ingest")
	if err != nil || task.StashScanJobID != "scan-1" {
		t.Fatal("workflow did not resume after runtime loss", err)
	}
}
