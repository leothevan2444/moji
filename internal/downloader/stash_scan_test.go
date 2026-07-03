package downloader

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
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
		SavePath:    "/downloads",
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
			DeliveryMode: stashsync.DeliveryModePathMap,
			Downloads: stashsync.DownloadsPathConfig{
				QBRoot: "/downloads",
			},
			Library: stashsync.LibraryPathConfig{
				StashRoot: "/library",
			},
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
	if task.StashMode != string(stashsync.DeliveryModePathMap) {
		t.Fatalf("unexpected stash mode: %+v", task)
	}
	if task.StashSourcePath != "" {
		t.Fatalf("path-map mode should not record a Moji transfer source path: %+v", task)
	}
	if task.StashScanPath != "/library/ABCD-123.mp4" {
		t.Fatalf("unexpected scan path: %+v", task)
	}
	if task.StashJobID != "job-1" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
}

func TestTriggerStashScansFallsBackToSavePathForPathMap(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:        "task-savepath-fallback",
		Status:    TaskStatusCompleted,
		SavePath:  "/downloads/SSIS-279",
		CreatedAt: time.Unix(100, 0).UTC(),
		UpdatedAt: time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.TriggerTaskStashScan(context.Background(), "task-savepath-fallback", &fakeStashScanner{
		jobID: "job-fallback",
		config: stashsync.IntegrationConfig{
			DeliveryMode: stashsync.DeliveryModePathMap,
			Downloads:    stashsync.DownloadsPathConfig{QBRoot: "/downloads"},
			Library:      stashsync.LibraryPathConfig{StashRoot: "/library"},
		},
	})
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if task.StashScanPath != "/library/SSIS-279" {
		t.Fatalf("unexpected scan path from save-path fallback: %+v", task)
	}
}

func TestTriggerStashScansStartsFileTransferCopyBeforeScan(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-transfer",
		Status:      TaskStatusCompleted,
		SavePath:    "/downloads",
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
			DeliveryMode: stashsync.DeliveryModeTransfer,
			Downloads: stashsync.DownloadsPathConfig{
				QBRoot:   "/downloads",
				MojiRoot: "/srv/jav-downloads",
			},
			Library: stashsync.LibraryPathConfig{
				MojiRoot:  "/mnt/library",
				StashRoot: "/library",
			},
			Transfer: stashsync.TransferConfig{Action: stashsync.TransferActionCopy},
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
	if call.sourcePath != "/srv/jav-downloads/ABCD-123.mp4" || call.targetPath != "/mnt/library/ABCD-123.mp4" || call.action != stashsync.TransferActionCopy {
		t.Fatalf("unexpected file transfer call: %+v", call)
	}
	if task.StashSourcePath != "/srv/jav-downloads/ABCD-123.mp4" {
		t.Fatalf("unexpected moji source path: %+v", task)
	}
	if task.StashTransferStatus != StashTransferStatusCompleted || task.StashScanPath != "/library/ABCD-123.mp4" {
		t.Fatalf("unexpected task after transfer: %+v", task)
	}
	if task.StashJobID != "job-transfer" || task.StashScanStatus != StashScanStatusStarted {
		t.Fatalf("unexpected scan task: %+v", task)
	}
}

func TestTriggerTaskStashScanStartsTransferSymlinkBeforeScan(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-symlink",
		Status:      TaskStatusCompleted,
		SavePath:    "/downloads",
		ContentPath: "/downloads/ABCD-999.mp4",
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
		jobID: "job-symlink",
		config: stashsync.IntegrationConfig{
			DeliveryMode: stashsync.DeliveryModeTransfer,
			Downloads: stashsync.DownloadsPathConfig{
				QBRoot:   "/downloads",
				MojiRoot: "/srv/jav-downloads",
			},
			Library: stashsync.LibraryPathConfig{
				MojiRoot:  "/mnt/library",
				StashRoot: "/library",
			},
			Transfer: stashsync.TransferConfig{Action: stashsync.TransferActionSymlink},
		},
	}
	task, err := service.TriggerTaskStashScan(context.Background(), "task-symlink", scanner)
	if err != nil {
		t.Fatalf("TriggerTaskStashScan failed: %v", err)
	}
	if len(fileOps.calls) != 1 {
		t.Fatalf("expected a single file transfer, got %+v", fileOps.calls)
	}
	call := fileOps.calls[0]
	if call.action != stashsync.TransferActionSymlink || call.sourcePath != "/srv/jav-downloads/ABCD-999.mp4" || call.targetPath != "/mnt/library/ABCD-999.mp4" {
		t.Fatalf("unexpected symlink transfer call: %+v", call)
	}
	if task.StashTransferStatus != StashTransferStatusCompleted || task.StashScanPath != "/library/ABCD-999.mp4" {
		t.Fatalf("unexpected symlink task after transfer: %+v", task)
	}
}

func TestTriggerTaskStashScanRecordsTransferFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-transfer-fail",
		Status:      TaskStatusCompleted,
		SavePath:    "/downloads",
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
			DeliveryMode: stashsync.DeliveryModeTransfer,
			Downloads: stashsync.DownloadsPathConfig{
				QBRoot:   "/downloads",
				MojiRoot: "/srv/jav-downloads",
			},
			Library: stashsync.LibraryPathConfig{
				MojiRoot:  "/mnt/library",
				StashRoot: "/library",
			},
			Transfer: stashsync.TransferConfig{Action: stashsync.TransferActionMove},
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

func TestTriggerTaskStashScanRejectsQBRootMismatch(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-mismatch",
		Status:      TaskStatusCompleted,
		SavePath:    "/downloads",
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
			DeliveryMode: stashsync.DeliveryModePathMap,
			Downloads:    stashsync.DownloadsPathConfig{QBRoot: "/downloads"},
			Library:      stashsync.LibraryPathConfig{StashRoot: "/library"},
		},
	})
	if err == nil {
		t.Fatal("expected mapping error")
	}
	if task.StashScanStatus != StashScanStatusFailed || task.StashScanError == "" {
		t.Fatalf("expected failed scan task, got %+v", task)
	}
	if got := task.StashScanError; !strings.Contains(got, "resolve qB relative path failed") {
		t.Fatalf("expected qB relative path error, got %q", got)
	}
}

func TestTriggerTaskStashScanRejectsMissingTransferRoots(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-missing-transfer-root",
		Status:      TaskStatusCompleted,
		ContentPath: "/downloads/scene/file.mp4",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	_, err = service.TriggerTaskStashScan(context.Background(), "task-missing-transfer-root", &fakeStashScanner{
		config: stashsync.IntegrationConfig{
			DeliveryMode: stashsync.DeliveryModeTransfer,
			Downloads:    stashsync.DownloadsPathConfig{QBRoot: "/downloads"},
			Library:      stashsync.LibraryPathConfig{StashRoot: "/library"},
			Transfer:     stashsync.TransferConfig{Action: stashsync.TransferActionCopy},
		},
	})
	if err == nil {
		t.Fatal("expected missing transfer roots error")
	}
}

func TestTriggerTaskStashScanAllowsRetryAfterFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:                  "task-retry",
		Status:              TaskStatusCompleted,
		SavePath:            "/downloads",
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
		DeliveryMode: stashsync.DeliveryModePathMap,
		Downloads:    stashsync.DownloadsPathConfig{QBRoot: "/downloads"},
		Library:      stashsync.LibraryPathConfig{StashRoot: "/library"},
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
		SavePath:    "/downloads",
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
			DeliveryMode: stashsync.DeliveryModePathMap,
			Downloads:    stashsync.DownloadsPathConfig{QBRoot: "/downloads"},
			Library:      stashsync.LibraryPathConfig{StashRoot: "/library"},
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
