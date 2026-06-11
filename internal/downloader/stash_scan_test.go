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

func TestTriggerTaskStashScanStartsSingleTask(t *testing.T) {
	store := NewMemoryTaskStore()
	completedAt := time.Unix(200, 0).UTC()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-single",
		Status:      TaskStatusCompleted,
		SavePath:    "/downloads/task-single",
		CompletedAt: &completedAt,
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanner := &fakeStashScanner{jobID: "job-single"}
	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithClock(func() time.Time { return time.Unix(300, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-single", scanner)
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if task.StashJobID != "job-single" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected task after scan start: %+v", task)
	}
	if len(scanner.requests) != 1 || scanner.requests[0].Paths[0] != "/downloads/task-single" {
		t.Fatalf("unexpected stash scan request: %+v", scanner.requests)
	}
}

func TestTriggerTaskStashScanAllowsRetryAfterFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:              "task-retry",
		Status:          TaskStatusCompleted,
		ContentPath:     "/downloads/retry.mp4",
		StashScanStatus: StashScanStatusFailed,
		StashScanError:  "previous failure",
		CreatedAt:       time.Unix(100, 0).UTC(),
		UpdatedAt:       time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanner := &fakeStashScanner{jobID: "job-retry"}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-retry", scanner)
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if task.StashScanStatus != StashScanStatusStarted || task.StashScanError != "" {
		t.Fatalf("expected failed scan to be retried, got %+v", task)
	}
}

func TestTriggerTaskStashScanRejectsTaskNotReady(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:        "task-running",
		Status:    TaskStatusDownloading,
		CreatedAt: time.Unix(100, 0).UTC(),
		UpdatedAt: time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	_, err = service.TriggerTaskStashScan(context.Background(), "task-running", &fakeStashScanner{})
	if err == nil {
		t.Fatal("expected not-ready error")
	}
	if got := err.Error(); got != `downloader: task "task-running" is not ready for stash scan` {
		t.Fatalf("unexpected error: %q", got)
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
