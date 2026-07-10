package taskruntime

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
		ID:                  "task-completed",
		Stage:               TaskStagePendingIngest,
		StageStatus:         TaskStageStatusPending,
		SavePath:            "/downloads",
		ContentPath:         "/downloads/ABCD-123.mp4",
		DownloadCompletedAt: &completedAt,
		CreatedAt:           time.Unix(100, 0).UTC(),
		UpdatedAt:           time.Unix(200, 0).UTC(),
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
	if task.DeliveryMode != string(stashsync.DeliveryModePathMap) {
		t.Fatalf("unexpected stash mode: %+v", task)
	}
	if task.MojiSourcePath != "" {
		t.Fatalf("path-map mode should not record a Moji transfer source path: %+v", task)
	}
	if task.StashScanPath != "/library/ABCD-123.mp4" {
		t.Fatalf("unexpected scan path: %+v", task)
	}
	if task.StashScanJobID != "job-1" || task.Stage != TaskStageScanning || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
}

func TestTriggerStashScansFallsBackToSavePathForPathMap(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-savepath-fallback",
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
		SavePath:    "/downloads/SSIS-279",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(200, 0).UTC(),
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
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
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
	if task.MojiSourcePath != "/srv/jav-downloads/ABCD-123.mp4" {
		t.Fatalf("unexpected moji source path: %+v", task)
	}
	if task.MojiTransferPath != "/mnt/library/ABCD-123.mp4" || task.StashScanPath != "/library/ABCD-123.mp4" {
		t.Fatalf("unexpected task after transfer: %+v", task)
	}
	if task.StashScanJobID != "job-transfer" || task.Stage != TaskStageScanning || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected scan task: %+v", task)
	}
}

func TestTriggerTaskStashScanStartsTransferSymlinkBeforeScan(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-symlink",
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
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
	if task.MojiTransferPath != "/mnt/library/ABCD-999.mp4" || task.StashScanPath != "/library/ABCD-999.mp4" || task.Stage != TaskStageScanning || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected symlink task after transfer: %+v", task)
	}
}

func TestTriggerTaskStashScanRecordsTransferFailure(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-transfer-fail",
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
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
	if task.Stage != TaskStageTransferring || task.StageStatus != TaskStageStatusBlocked || task.TransferError != transferErr.Error() {
		t.Fatalf("unexpected failed transfer task: %+v", task)
	}
	if task.StashScanError != transferErr.Error() || task.StashScanJobID != "" {
		t.Fatalf("unexpected scan state after failed transfer: %+v", task)
	}
}

func TestTriggerTaskStashScanRejectsQBRootMismatch(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-mismatch",
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
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
	if task.Stage != TaskStagePendingIngest || task.StageStatus != TaskStageStatusBlocked || task.StashScanError == "" {
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
		Stage:       TaskStagePendingIngest,
		StageStatus: TaskStageStatusPending,
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
		ID:             "task-retry",
		Stage:          TaskStageScanning,
		StageStatus:    TaskStageStatusBlocked,
		SavePath:       "/downloads",
		ContentPath:    "/downloads/retry.mp4",
		TransferError:  "previous failure",
		StashScanError: "previous failure",
		CreatedAt:      time.Unix(100, 0).UTC(),
		UpdatedAt:      time.Unix(200, 0).UTC(),
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
	if task.Stage != TaskStageScanning || task.StageStatus != TaskStageStatusRunning || task.StashScanError != "" {
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
		ID:                  "task-sqlite-update",
		Stage:               TaskStagePendingIngest,
		StageStatus:         TaskStageStatusPending,
		SavePath:            "/downloads",
		ContentPath:         "/downloads/ABCD-123.mp4",
		DownloadCompletedAt: &completedAt,
		CreatedAt:           time.Unix(100, 0).UTC(),
		UpdatedAt:           time.Unix(200, 0).UTC(),
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
	if reloaded.StashScanJobID != "job-sqlite" || reloaded.Stage != TaskStageScanning || reloaded.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected persisted task: %+v", reloaded)
	}
}

func TestTriggerStashScansPersistsCompletedScanJobState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks-completed.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteTaskStore failed: %v", err)
	}

	if err := store.Create(context.Background(), &Task{
		ID:                 "task-scan-finished",
		Stage:              TaskStageScanning,
		StageStatus:        TaskStageStatusRunning,
		StashScanJobID:     "job-finished",
		CreatedAt:          time.Unix(100, 0).UTC(),
		UpdatedAt:          time.Unix(200, 0).UTC(),
		StashScanStartedAt: ptrTime(time.Unix(150, 0).UTC()),
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

	tasks, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{
		job: &stashsync.Job{ID: "job-finished", Status: "FINISHED"},
	})
	if err != nil {
		t.Fatalf("TriggerStashScans failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one updated task, got %d", len(tasks))
	}
	if tasks[0].Stage != TaskStageCompleted || tasks[0].StageStatus != TaskStageStatusDone {
		t.Fatalf("expected completed task stage, got %+v", tasks[0])
	}
	reloaded, err := store.Find(context.Background(), "task-scan-finished")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if reloaded.Stage != TaskStageCompleted || reloaded.StageStatus != TaskStageStatusDone {
		t.Fatalf("expected persisted completed stage, got %+v", reloaded)
	}
}

type fakeStashScanner struct {
	jobID    string
	err      error
	job      *stashsync.Job
	jobErr   error
	config   stashsync.IntegrationConfig
	requests []stashsync.ScanRequest
}

func (f *fakeStashScanner) MetadataScan(_ context.Context, req stashsync.ScanRequest) (string, error) {
	f.requests = append(f.requests, req)
	return f.jobID, f.err
}

func (f *fakeStashScanner) FindJob(_ context.Context, _ string) (*stashsync.Job, error) {
	return f.job, f.jobErr
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

func ptrTime(value time.Time) *time.Time {
	return &value
}
