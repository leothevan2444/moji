package downloader

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

func TestTriggerStashScansStartsSharedStorageScanForCompletedTask(t *testing.T) {
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

	scanner := &fakeStashScanner{
		jobID: "job-1",
		config: stashsync.IntegrationConfig{
			Mode:                  stashsync.IntegrationModeSharedStorage,
			QBittorrentPathPrefix: "/downloads",
			StashPathPrefix:       "/library",
		},
	}
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
	task := tasks[0]
	if task.StashMode != string(stashsync.IntegrationModeSharedStorage) {
		t.Fatalf("unexpected stash mode: %+v", task)
	}
	if task.StashScanPath != "/library/ABCD-123.mp4" {
		t.Fatalf("unexpected scan path: %+v", task)
	}
	if task.StashJobID != "job-1" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
}

func TestTriggerStashScansStartsFileTransferCopyBeforeScan(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-transfer",
		Status:      TaskStatusCompleted,
		ContentPath: "/downloads/ABCD-123.mp4",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	fileOps := &fakeFileOperator{}
	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithClock(func() time.Time { return time.Unix(300, 0).UTC() }),
		WithFileOperator(fileOps),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	scanner := &fakeStashScanner{
		jobID: "job-transfer",
		config: stashsync.IntegrationConfig{
			Mode:               stashsync.IntegrationModeFileTransfer,
			TransferAction:     stashsync.TransferActionCopy,
			TransferTargetPath: "/stash-import",
		},
	}
	task, err := service.TriggerTaskStashScan(context.Background(), "task-transfer", scanner)
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if len(fileOps.calls) != 1 {
		t.Fatalf("expected a single file transfer, got %+v", fileOps.calls)
	}
	call := fileOps.calls[0]
	if call.sourcePath != "/downloads/ABCD-123.mp4" || call.targetPath != "/stash-import/ABCD-123.mp4" || call.action != stashsync.TransferActionCopy {
		t.Fatalf("unexpected file transfer call: %+v", call)
	}
	if task.StashTransferStatus != StashTransferStatusCompleted || task.StashScanPath != "/stash-import/ABCD-123.mp4" {
		t.Fatalf("unexpected task after transfer: %+v", task)
	}
	if task.StashJobID != "job-transfer" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected scan task: %+v", task)
	}
}

func TestTriggerTaskStashScanRecordsTransferFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-transfer-fail",
		Status:      TaskStatusCompleted,
		ContentPath: "/downloads/fail.mp4",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	transferErr := errors.New("target exists")
	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithFileOperator(&fakeFileOperator{err: transferErr}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-transfer-fail", &fakeStashScanner{
		config: stashsync.IntegrationConfig{
			Mode:               stashsync.IntegrationModeFileTransfer,
			TransferAction:     stashsync.TransferActionMove,
			TransferTargetPath: "/stash-import",
		},
	})
	if err == nil {
		t.Fatal("expected transfer error")
	}
	if task.StashTransferStatus != StashTransferStatusFailed || task.StashTransferError != transferErr.Error() {
		t.Fatalf("unexpected failed transfer task: %+v", task)
	}
	if task.StashScanStatus != StashScanStatusFailed || task.StashJobID != "" {
		t.Fatalf("unexpected scan state after failed transfer: %+v", task)
	}
}

func TestTriggerTaskStashScanUsesLibraryPathInLibraryScanMode(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:        "task-library",
		Status:    TaskStatusCompleted,
		CreatedAt: time.Unix(100, 0).UTC(),
		UpdatedAt: time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-library", &fakeStashScanner{
		jobID: "job-library",
		config: stashsync.IntegrationConfig{
			Mode:        stashsync.IntegrationModeLibraryScan,
			LibraryPath: "/data/library",
		},
	})
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if task.StashScanPath != "/data/library" {
		t.Fatalf("unexpected library scan path: %+v", task)
	}
	if task.StashScanHint == "" {
		t.Fatalf("expected library scan hint, got %+v", task)
	}
}

func TestTriggerTaskStashScanRejectsMissingSharedStorageMapping(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-mismatch",
		Status:      TaskStatusCompleted,
		ContentPath: "/different/file.mp4",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-mismatch", &fakeStashScanner{
		config: stashsync.IntegrationConfig{
			Mode:                  stashsync.IntegrationModeSharedStorage,
			QBittorrentPathPrefix: "/downloads",
			StashPathPrefix:       "/library",
		},
	})
	if err == nil {
		t.Fatal("expected mapping error")
	}
	if task.StashScanStatus != StashScanStatusFailed || task.StashScanError == "" {
		t.Fatalf("expected failed scan task, got %+v", task)
	}
}

func TestTriggerTaskStashScanAllowsRetryAfterFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:                  "task-retry",
		Status:              TaskStatusCompleted,
		ContentPath:         "/downloads/retry.mp4",
		StashScanStatus:     StashScanStatusFailed,
		StashTransferStatus: StashTransferStatusFailed,
		StashScanError:      "previous failure",
		CreatedAt:           time.Unix(100, 0).UTC(),
		UpdatedAt:           time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scanner := &fakeStashScanner{jobID: "job-retry", config: stashsync.IntegrationConfig{
		Mode:                  stashsync.IntegrationModeSharedStorage,
		QBittorrentPathPrefix: "/downloads",
		StashPathPrefix:       "/library",
	}}
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

func TestTriggerStashScansPersistsSQLiteUpdatesWithExtendedStashColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteTaskStore failed: %v", err)
	}

	completedAt := time.Unix(200, 0).UTC()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-sqlite-update",
		Status:      TaskStatusCompleted,
		ContentPath: "/downloads/ABCD-123.mp4",
		CompletedAt: &completedAt,
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(
		fakeTracker{},
		&fakeTorrentAdder{},
		store,
		WithClock(func() time.Time { return time.Unix(300, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	scanner := &fakeStashScanner{
		jobID: "job-sqlite",
		config: stashsync.IntegrationConfig{
			Mode:                  stashsync.IntegrationModeSharedStorage,
			QBittorrentPathPrefix: "/downloads",
			StashPathPrefix:       "/library",
		},
	}

	tasks, err := service.TriggerStashScans(context.Background(), scanner)
	if err != nil {
		t.Fatalf("TriggerStashScans failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one updated task, got %d", len(tasks))
	}

	reloaded, err := store.Find(context.Background(), "task-sqlite-update")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if reloaded.StashJobID != "job-sqlite" || reloaded.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected persisted task: %+v", reloaded)
	}
}

type fakeStashScanner struct {
	jobID    string
	err      error
	config   stashsync.IntegrationConfig
	requests []stashsync.ScanRequest
}

func (f *fakeStashScanner) MetadataScan(_ context.Context, req stashsync.ScanRequest) (string, error) {
	f.requests = append(f.requests, req)
	return f.jobID, f.err
}

func (f *fakeStashScanner) CurrentConfig() stashsync.IntegrationConfig {
	return f.config
}

type fakeFileOperator struct {
	err   error
	calls []fileTransferCall
}

type fileTransferCall struct {
	sourcePath string
	targetPath string
	action     stashsync.TransferAction
}

func (f *fakeFileOperator) Transfer(_ context.Context, sourcePath string, action stashsync.TransferAction, targetPath string) error {
	f.calls = append(f.calls, fileTransferCall{
		sourcePath: sourcePath,
		targetPath: targetPath,
		action:     action,
	})
	return f.err
}
