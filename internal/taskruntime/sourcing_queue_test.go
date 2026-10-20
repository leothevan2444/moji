package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type queuedTracker struct {
	optionsMu sync.Mutex
	entered   chan string
	release   <-chan struct{}
	calls     atomic.Int32
	options   tracker.SearchOptions
	results   []jackett.SearchResult
}

func (f *queuedTracker) Search(code string, options ...tracker.SearchOption) ([]jackett.SearchResult, error) {
	f.calls.Add(1)
	if f.entered != nil {
		f.entered <- code
	}
	if f.release != nil {
		<-f.release
	}
	f.optionsMu.Lock()
	for _, option := range options {
		option(&f.options)
	}
	f.optionsMu.Unlock()
	return f.results, nil
}
func queuedResult() []jackett.SearchResult {
	return []jackett.SearchResult{{Title: "ABCD-123", MagnetURI: "magnet:?xt=urn:btih:abc123&dn=ABCD-123", Seeders: 10}}
}
func waitQueuedTask(t *testing.T, store TaskStore, id string, stage TaskStage) *Task {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		task, err := store.Find(context.Background(), id)
		if err == nil && task.Stage == stage {
			return task
		}
		select {
		case <-deadline.C:
			t.Fatalf("task did not reach %s: %+v %v", stage, task, err)
		case <-ticker.C:
		}
	}
}
func TestQueueReturnsAndPublishesBeforeExternalWork(t *testing.T) {
	release := make(chan struct{})
	search := &queuedTracker{entered: make(chan string, 1), release: release, results: queuedResult()}
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	bus := NewTaskEventBus(16)
	defer bus.Close()
	events := bus.Subscribe(context.Background())
	eventStore, err := NewEventingTaskStore(store, bus)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(search, qbt, eventStore, WithIDGenerator(func() string { return "queued" }))
	if err != nil {
		t.Fatal(err)
	}
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	task, err := service.QueueDownloadContext(requestCtx, DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if search.calls.Load() != 0 || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusPending {
		t.Fatalf("creation executed sourcing: %+v", task)
	}
	select {
	case event := <-events:
		if event.Type != TaskEventCreated || event.Task.StageStatus != TaskStageStatusPending {
			t.Fatalf("unexpected creation: %+v", event)
		}
	default:
		t.Fatal("created task was not published immediately")
	}
	cancelRequest() // The HTTP request may finish while the worker is still active.
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { service.RunSourcingWorker(workerCtx); close(done) }()
	defer func() {
		cancelWorker()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not stop")
		}
	}()
	select {
	case <-search.entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("worker did not start")
	}
	visible, _ := store.Find(context.Background(), task.ID)
	if visible.Stage != TaskStageSourcing || visible.StageStatus != TaskStageStatusRunning {
		close(release)
		t.Fatalf("task not visible while search blocked: %+v", visible)
	}
	close(release)
	completed := waitQueuedTask(t, store, task.ID, TaskStageDownloading)
	if completed.StageStatus != TaskStageStatusRunning {
		t.Fatalf("worker failed after request cancellation: %+v", completed)
	}
	if task.StageStatus != TaskStageStatusPending {
		t.Fatal("worker mutated the creation response")
	}
}
func TestQueuedSourcingSurvivesRestartWithOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	search := &queuedTracker{results: queuedResult()}
	qbt := &fakeTorrentAdder{}
	service, _ := NewService(search, qbt, store)
	paused := true
	req := DownloadRequest{Code: "ABCD-123", Source: TaskSourceSearch, Trackers: []string{"indexer"}, Categories: []int{1}, Limit: 3, SavePath: "/downloads", Category: "moji", Tags: "favorite", Paused: &paused}
	task, err := service.QueueDownloadContext(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	restored, err := store.Find(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	var saved DownloadRequest
	if err := json.Unmarshal([]byte(restored.SourcingRequestJSON), &saved); err != nil || saved.Limit != 3 || saved.Paused == nil || !*saved.Paused {
		t.Fatalf("request options lost: %+v %v", saved, err)
	}
	service, _ = NewService(search, qbt, store)
	if err := service.processQueuedSourcing(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.Find(context.Background(), task.ID)
	if stored.Stage != TaskStageDownloading || stored.SourcingCompletedAt == nil || search.calls.Load() != 1 {
		t.Fatalf("queue did not recover: %+v", stored)
	}
	if qbt.options.SavePath == nil || *qbt.options.SavePath != "/downloads" || qbt.options.Paused == nil || !*qbt.options.Paused {
		t.Fatalf("submission options lost: %+v", qbt.options)
	}
}
func TestQueuedSourcingFailureAndRetry(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	search := &queuedTracker{}
	service, _ := NewService(search, &fakeTorrentAdder{}, store)
	task, err := service.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.processQueuedSourcing(ctx, task.ID); err == nil {
		t.Fatal("expected search failure")
	}
	failed, _ := store.Find(ctx, task.ID)
	if failed.StageStatus != TaskStageStatusBlocked || failed.StageErrorCode != TaskStageErrorNoCandidate {
		t.Fatalf("failure not visible: %+v", failed)
	}
	search.results = queuedResult()
	retried, err := service.RetryTask(ctx, task.ID, nil)
	if err != nil || retried.StageStatus != TaskStageStatusPending || search.calls.Load() != 1 {
		t.Fatalf("retry did not requeue: %+v %v", retried, err)
	}
	if err := service.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123"}); !errors.Is(err, ErrDuplicateCodeTask) {
		t.Fatalf("duplicate not rejected: %v", err)
	}
}
func TestQueuedLibraryValidationRunsAfterCreation(t *testing.T) {
	store := NewMemoryTaskStore()
	search := &queuedTracker{results: queuedResult()}
	service, _ := NewService(search, &fakeTorrentAdder{}, store, WithLibraryCodeChecker(fakeLibraryCodeChecker{codes: map[string]bool{"ABCD-123": true}}))
	task, err := service.QueueDownloadContext(context.Background(), DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.processQueuedSourcing(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.Find(context.Background(), task.ID)
	if stored.StageStatus != TaskStageStatusBlocked || stored.StageErrorCode != TaskStageErrorDuplicateLibrary || search.calls.Load() != 0 {
		t.Fatalf("library validation failed: %+v", stored)
	}
}

func TestSourcingWorkerBoundsConcurrencyAndLeavesInterruptedJobsRecoverable(t *testing.T) {
	release := make(chan struct{})
	search := &queuedTracker{entered: make(chan string, 3), release: release}
	store := NewMemoryTaskStore()
	var sequence atomic.Int32
	service, _ := NewService(search, &fakeTorrentAdder{}, store, WithIDGenerator(func() string { return fmt.Sprintf("queued-%d", sequence.Add(1)) }))
	for _, code := range []string{"ABCD-123", "ABCD-124", "ABCD-125"} {
		if _, err := service.QueueDownloadContext(context.Background(), DownloadRequest{Code: code}); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { service.RunSourcingWorker(ctx); close(done) }()
	for range 2 {
		select {
		case <-search.entered:
		case <-time.After(time.Second):
			cancel()
			close(release)
			t.Fatal("workers did not run")
		}
	}
	select {
	case <-search.entered:
		cancel()
		close(release)
		t.Fatal("more than two searches ran concurrently")
	default:
	}
	cancel()
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	tasks, _ := store.List(context.Background())
	for _, task := range tasks {
		if task.Stage != TaskStageSourcing || task.StageStatus == TaskStageStatusBlocked || task.SourcingRequestJSON == "" {
			t.Fatalf("shutdown lost recoverable intent: %+v", task)
		}
	}
}

type interruptedQueueAdder struct {
	fakeTorrentAdder
	cancel context.CancelFunc
}

func (f *interruptedQueueAdder) AddNewTorrent(ctx context.Context, _ qbittorrent.AddTorrentOptions) error {
	f.cancel()
	return ctx.Err()
}
func TestInterruptedSubmissionResumesTheSameValidatedTorrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "submit.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	search := &queuedTracker{results: queuedResult()}
	service, _ := NewService(search, &interruptedQueueAdder{cancel: cancel}, store)
	task, err := service.QueueDownloadContext(context.Background(), DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.processQueuedSourcing(ctx, task.ID); err == nil {
		t.Fatal("expected interrupted submission")
	}
	stored, err := store.Find(context.Background(), task.ID)
	if err != nil || stored.Stage != TaskStageSourcing || stored.StageStatus != TaskStageStatusRunning || stored.TorrentURL == "" {
		t.Fatalf("selection was not checkpointed: %+v %v", stored, err)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	recoveredSearch := &queuedTracker{}
	service, _ = NewService(recoveredSearch, &fakeTorrentAdder{}, store)
	if err := service.processQueuedSourcing(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if recoveredSearch.calls.Load() != 0 {
		t.Fatal("recovery selected a different torrent")
	}
	recovered, _ := store.Find(context.Background(), task.ID)
	if recovered.Stage != TaskStageDownloading || recovered.TorrentURL != stored.TorrentURL {
		t.Fatalf("submission did not resume: %+v", recovered)
	}
}

func TestSQLiteV12QueueUpgradePreservesExistingTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.Create(context.Background(), &Task{ID: "existing", Code: "ABCD-123", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`ALTER TABLE tasks DROP COLUMN sourcing_request_json`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE task_store_meta SET value='12' WHERE key='schema_version'`); err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	task, err := store.Find(context.Background(), "existing")
	if err != nil || task.Stage != TaskStageCompleted || task.SourcingRequestJSON != "" {
		t.Fatalf("upgrade changed existing task: %+v %v", task, err)
	}
	var events int
	if err := store.db.Get(&events, `SELECT COUNT(*) FROM task_events WHERE task_id='existing'`); err != nil || events != 1 {
		t.Fatalf("upgrade lost history: %d %v", events, err)
	}
}
