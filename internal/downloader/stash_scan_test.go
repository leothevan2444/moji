package downloader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

func TestTriggerStashScansStartsScanForCompletedTask(t *testing.T) {
	store := NewMemoryTaskStore()
	completedAt := time.Unix(200, 0).UTC()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-completed",
		Status:      TaskStatusCompleted,
		ContentPath: "/downloads/ABCD-123.mp4",
		CompletedAt: &completedAt,
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanner := &fakeStashScanner{jobID: "job-1"}
	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithClock(func() time.Time { return time.Unix(300, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	tasks, err := service.TriggerStashScans(context.Background(), scanner)
	if err != nil {
		t.Fatalf("TriggerStashScans failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.StashJobID != "job-1" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
	if task.StashScanStartedAt == nil || !task.StashScanStartedAt.Equal(time.Unix(300, 0).UTC()) {
		t.Fatalf("unexpected stash scan started at: %v", task.StashScanStartedAt)
	}
	if len(scanner.requests) != 1 || len(scanner.requests[0].Paths) != 1 || scanner.requests[0].Paths[0] != "/downloads/ABCD-123.mp4" {
		t.Fatalf("unexpected stash scan request: %+v", scanner.requests)
	}
}

func TestTriggerStashScansSkipsAlreadyStartedTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:              "task-started",
		Status:          TaskStatusCompleted,
		StashJobID:      "job-existing",
		StashScanStatus: StashScanStatusStarted,
		CreatedAt:       time.Unix(100, 0).UTC(),
		UpdatedAt:       time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanner := &fakeStashScanner{jobID: "job-new"}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	if _, err := service.TriggerStashScans(context.Background(), scanner); err != nil {
		t.Fatalf("TriggerStashScans failed: %v", err)
	}
	if len(scanner.requests) != 0 {
		t.Fatalf("expected no scan requests, got %+v", scanner.requests)
	}
}

func TestTriggerStashScansRecordsFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:        "task-failed-scan",
		Status:    TaskStatusCompleted,
		CreatedAt: time.Unix(100, 0).UTC(),
		UpdatedAt: time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanErr := errors.New("stash rejected scan")
	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithClock(func() time.Time { return time.Unix(300, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	tasks, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{err: scanErr})
	if err == nil {
		t.Fatal("expected scan error")
	}
	task := tasks[0]
	if task.StashScanStatus != StashScanStatusFailed || task.StashScanError != scanErr.Error() {
		t.Fatalf("unexpected failed scan task: %+v", task)
	}
}

type fakeStashScanner struct {
	jobID    string
	err      error
	requests []stashsync.ScanRequest
}

func (f *fakeStashScanner) MetadataScan(_ context.Context, req stashsync.ScanRequest) (string, error) {
	f.requests = append(f.requests, req)
	return f.jobID, f.err
}
