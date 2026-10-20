package taskruntime

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestResourcesAreTrackedAcrossMonthAndRestartUntilValidRelease(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "tracking.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	search := &queuedTracker{}
	qbt := &fakeTorrentAdder{}
	newService := func() *Service {
		s, err := NewService(search, qbt, store, WithClock(func() time.Time { return clock }), WithSourcingRetryInterval(func() time.Duration { return 6 * time.Hour }))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := newService()
	paused := true
	task, err := s.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123", Trackers: []string{"preferred"}, Categories: []int{6000}, Limit: 3, SavePath: "/downloads", Category: "moji", Tags: "tracked", Paused: &paused})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 125; attempt++ {
		if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
			t.Fatal(err)
		}
		task, err = store.Find(ctx, task.ID)
		if err != nil || !waitingForResource(task) || task.StageErrorCode != "" || task.SourcingCompletedAt != nil || !task.SourcingLastCheckedAt.Equal(clock) || !task.SourcingNextCheckAt.Equal(clock.Add(6*time.Hour)) {
			t.Fatalf("resource absence became an error or lost its plan: %+v %v", task, err)
		}
		calls := search.calls.Load()
		clock = clock.Add(6*time.Hour - time.Second)
		if err := s.processQueuedSourcing(ctx, task.ID); err != nil || search.calls.Load() != calls {
			t.Fatal("resource searched before its scheduled time")
		}
		clock = clock.Add(time.Second)
		if attempt == 15 {
			store.db.Close()
			store, err = NewSQLiteTaskStore(path)
			if err != nil {
				t.Fatal(err)
			}
			s = newService()
			restored, _ := store.Find(ctx, task.ID)
			if restored.SourcingNextCheckAt == nil || !restored.SourcingNextCheckAt.Equal(clock) {
				t.Fatalf("restart lost schedule: %+v", restored)
			}
		}
	}
	defer store.db.Close()
	search.results = []jackett.SearchResult{{Title: "abcd00123", MagnetURI: "magnet:?xt=urn:btih:abc123&dn=ABCD_123"}}
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	if task.Stage != TaskStageDownloading || task.SourcingNextCheckAt != nil || task.SourcingCompletedAt == nil || len(qbt.options.URLs) != 1 {
		t.Fatalf("valid resource did not start downloading: %+v", task)
	}
	if search.options.Limit != 3 || len(search.options.Trackers) != 1 || search.options.Trackers[0] != "preferred" || len(search.options.Categories) != 1 || search.options.Categories[0] != 6000 {
		t.Fatalf("tracking lost search options: %+v", search.options)
	}
	if qbt.options.Paused == nil || !*qbt.options.Paused || qbt.options.SavePath == nil || *qbt.options.SavePath != "/downloads" {
		t.Fatal("tracking lost submission preferences")
	}
	calls := search.calls.Load()
	clock = clock.Add(30 * 24 * time.Hour)
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil || search.calls.Load() != calls {
		t.Fatal("downloading task was searched again")
	}
}

func TestTrackingRejectsFuzzyResultsAndManualRetryRunsImmediately(t *testing.T) {
	ctx := context.Background()
	search := &queuedTracker{results: []jackett.SearchResult{{Title: "ABCD124", MagnetURI: "magnet:?xt=urn:btih:wrong"}}}
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	s, _ := NewService(search, qbt, store)
	task, _ := s.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123"})
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	if !waitingForResource(task) || len(qbt.options.URLs) != 0 || task.TorrentURL != "" {
		t.Fatalf("fuzzy resource was used: %+v", task)
	}
	search.results = queuedResult()
	payload, err := s.RetryTasks(ctx, []string{task.ID}, nil)
	if err != nil || payload.Summary.SucceededCount != 1 {
		t.Fatalf("manual check could not bypass scheduled wait: %+v %v", payload, err)
	}
	task, _ = store.Find(ctx, task.ID)
	if task.SourcingNextCheckAt != nil {
		t.Fatal("manual check still deferred")
	}
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	if task.Stage != TaskStageDownloading {
		t.Fatalf("manual resource check did not download: %+v", task)
	}
}

func TestTransientSearchFailureIsVisibleAndScheduledButPermanentFailureIsNot(t *testing.T) {
	ctx := context.Background()
	clock := time.Now().UTC()
	store := NewMemoryTaskStore()
	s, _ := NewService(fakeTracker{err: errors.New("indexer unavailable")}, &fakeTorrentAdder{}, store, WithClock(func() time.Time { return clock }))
	task, _ := s.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123"})
	if err := s.processQueuedSourcing(ctx, task.ID); err == nil {
		t.Fatal("indexer error was hidden")
	}
	task, _ = store.Find(ctx, task.ID)
	if task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != TaskStageErrorSearch || task.SourcingNextCheckAt == nil {
		t.Fatalf("transient failure lost diagnostics or retry: %+v", task)
	}
	clock = *task.SourcingNextCheckAt
	s.tracker = fakeTracker{results: queuedResult()}
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	if task.Stage != TaskStageDownloading {
		t.Fatalf("transient failure did not recover: %+v", task)
	}
	for _, code := range []string{TaskStageErrorDuplicateLibrary, TaskStageErrorDuplicateTorrent, TaskStageErrorCodeRequired, TaskStageErrorSourcingRequest} {
		task := &Task{ID: code, Code: "ABCD-124", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, StageErrorCode: code, CreatedAt: clock, UpdatedAt: clock}
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		task, err := s.prepareResourceTracking(ctx, task)
		if err != nil || task.SourcingNextCheckAt != nil || task.StageStatus != TaskStageStatusBlocked {
			t.Fatalf("permanent failure automatically retried: %+v %v", task, err)
		}
	}
}

func TestLegacyBlockedSearchAndIntervalChangesAreAdopted(t *testing.T) {
	ctx := context.Background()
	clock := time.Now().UTC()
	interval := 6 * time.Hour
	store := NewMemoryTaskStore()
	task := &Task{ID: "legacy", Code: "ABCD-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, StageErrorCode: TaskStageErrorCandidateMismatch, Candidate: Candidate{Title: "ABCD124"}, TorrentURL: "http://indexer.invalid/old", CreatedAt: clock.Add(-48 * time.Hour), UpdatedAt: clock.Add(-24 * time.Hour)}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	search := &queuedTracker{results: queuedResult()}
	s, _ := NewService(search, &fakeTorrentAdder{}, store, WithClock(func() time.Time { return clock }), WithSourcingRetryInterval(func() time.Duration { return interval }))
	task, err := s.prepareResourceTracking(ctx, task)
	if err != nil || !waitingForResource(task) || task.SourcingRequestJSON == "" || task.TorrentURL != "" || task.SourcingNextCheckAt.After(clock) {
		t.Fatalf("legacy task was not adopted: %+v %v", task, err)
	}
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil || search.calls.Load() != 1 {
		t.Fatalf("legacy task did not search again: %v", err)
	}
	// A newly waiting task picks up settings edits without a restart.
	s.tracker = fakeTracker{}
	task, _ = s.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-125"})
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	interval = time.Hour
	updated, err := s.prepareResourceTracking(ctx, task)
	if err != nil || !updated.SourcingNextCheckAt.Equal(clock.Add(time.Hour)) || !updated.SourcingLastCheckedAt.Equal(clock) {
		t.Fatalf("interval edit did not adjust schedule: %+v %v", updated, err)
	}
}

type appearingResourceTracker struct{ calls atomic.Int32 }

func (s *appearingResourceTracker) Search(_ string, _ ...tracker.SearchOption) ([]jackett.SearchResult, error) {
	if s.calls.Add(1) == 1 {
		return nil, nil
	}
	return queuedResult(), nil
}

func TestWorkerWakesWhenResourceCheckBecomesDue(t *testing.T) {
	var unix atomic.Int64
	unix.Store(time.Now().Unix())
	search := &appearingResourceTracker{}
	store := NewMemoryTaskStore()
	s, _ := NewService(search, &fakeTorrentAdder{}, store, WithClock(func() time.Time { return time.Unix(unix.Load(), 0).UTC() }))
	task, _ := s.QueueDownloadContext(context.Background(), DownloadRequest{Code: "ABCD-123"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.RunSourcingWorker(ctx); close(done) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("worker did not stop")
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		task, _ = store.Find(context.Background(), task.ID)
		if waitingForResource(task) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker did not schedule next search: %+v", task)
		}
		time.Sleep(time.Millisecond)
	}
	unix.Store(task.SourcingNextCheckAt.Unix())
	s.sourcingWake <- struct{}{}
	task = waitQueuedTask(t, store, task.ID, TaskStageDownloading)
	if task.SourcingNextCheckAt != nil || search.calls.Load() != 2 {
		t.Fatalf("due search did not download exactly once: %+v calls=%d", task, search.calls.Load())
	}
}

func TestSchema13UpgradePreservesTasksAndAddsResourceSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	task := &Task{ID: "retained", Code: "ABCD-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, StageErrorCode: TaskStageErrorNoCandidate, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"ALTER TABLE tasks DROP COLUMN sourcing_last_checked_at", "ALTER TABLE tasks DROP COLUMN sourcing_next_check_at", "UPDATE task_store_meta SET value='13' WHERE key='schema_version'"} {
		if _, err := store.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	task, err = store.Find(context.Background(), task.ID)
	if err != nil || task.Code != "ABCD-123" || task.StageErrorCode != TaskStageErrorNoCandidate {
		t.Fatalf("upgrade discarded task: %+v %v", task, err)
	}
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	task, err = s.prepareResourceTracking(context.Background(), task)
	if err != nil || !waitingForResource(task) {
		t.Fatalf("upgraded task cannot be tracked: %+v %v", task, err)
	}
}

func TestResourceAbsenceNeverPublishesBlockedState(t *testing.T) {
	ctx := context.Background()
	bus := NewTaskEventBus(16)
	defer bus.Close()
	events := bus.Subscribe(ctx)
	store, err := NewEventingTaskStore(NewMemoryTaskStore(), bus)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	task, err := s.QueueDownloadContext(ctx, DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case event := <-events:
			if event.Task != nil && event.Task.StageStatus == TaskStageStatusBlocked {
				t.Fatalf("expected resource absence emitted an error: %+v", event.Task)
			}
		default:
			return
		}
	}
}

func TestInvalidQueuedRequestStopsScheduledRetry(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	store := NewMemoryTaskStore()
	task := &Task{ID: "invalid", Code: "ABCD-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusPending, SourcingRequestJSON: "invalid JSON", SourcingLastCheckedAt: &now, SourcingNextCheckAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err := s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(ctx, task.ID)
	if task.StageErrorCode != TaskStageErrorSourcingRequest || task.StageStatus != TaskStageStatusBlocked || task.SourcingNextCheckAt != nil {
		t.Fatalf("invalid request remained in retry loop: %+v", task)
	}
}
