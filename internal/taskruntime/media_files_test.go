package taskruntime

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type fileTorrent struct {
	fakeTorrentAdder
	files      []qbittorrent.TorrentContentFile
	calls      []string
	priorities map[int]qbittorrent.TorrentFilePriority
}

func (q *fileTorrent) GetTorrentContents(context.Context, string, []string) ([]qbittorrent.TorrentContentFile, error) {
	return q.files, nil
}
func (q *fileTorrent) SetTorrentFilePriority(_ context.Context, _ string, ids []int, priority qbittorrent.TorrentFilePriority) error {
	if q.priorities == nil {
		q.priorities = map[int]qbittorrent.TorrentFilePriority{}
	}
	for _, id := range ids {
		q.priorities[id] = priority
	}
	q.calls = append(q.calls, "priority")
	return nil
}
func (q *fileTorrent) PauseTorrents(context.Context, []string) error {
	q.calls = append(q.calls, "pause")
	return nil
}
func (q *fileTorrent) ResumeTorrents(context.Context, []string) error {
	q.calls = append(q.calls, "resume")
	return nil
}

type controlledPreparer struct {
	mu       sync.Mutex
	requests []mediafiles.Request
	started  chan struct{}
	release  chan struct{}
	err      error
}

func (p *controlledPreparer) Prepare(ctx context.Context, r mediafiles.Request) (mediafiles.Result, error) {
	p.mu.Lock()
	p.requests = append(p.requests, r)
	p.mu.Unlock()
	if p.started != nil {
		select {
		case p.started <- struct{}{}:
		default:
		}
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return mediafiles.Result{}, ctx.Err()
		}
	}
	return mediafiles.Result{Directory: filepath.Join(r.TargetRoot, r.Code), VideoPath: filepath.Join(r.TargetRoot, r.Code, r.Code+".mkv")}, p.err
}
func mediaConfig() stashsync.IntegrationConfig {
	return stashsync.IntegrationConfig{Downloads: stashsync.DownloadsPathConfig{QBRoot: "/q/downloads", MojiRoot: "/moji/downloads"}, Library: stashsync.LibraryPathConfig{MojiRoot: "/moji/library", StashRoot: "/stash/library"}, Transfer: stashsync.TransferConfig{Action: stashsync.TransferActionCopy}}
}
func fileService(t *testing.T, files []qbittorrent.TorrentContentFile, keep bool, preparer mediafiles.Preparer) (*Service, *MemoryTaskStore, *fileTorrent) {
	t.Helper()
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	task := &Task{ID: "file-task", Code: "OFJE-329", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, TorrentHash: "hash", SavePath: "/q/downloads", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	q := &fileTorrent{files: files, fakeTorrentAdder: fakeTorrentAdder{torrents: []qbittorrent.Torrent{{Hash: "hash", Progress: 1, State: qbittorrent.TorrentStateUploading, SavePath: "/q/downloads", ContentPath: "/q/downloads/torrent"}}}}
	cfg := mediaConfig()
	cfg.KeepSubtitles = keep
	service, err := NewService(fakeTracker{}, q, store, WithMediaPreparation(func() stashsync.IntegrationConfig { return cfg }, preparer))
	if err != nil {
		t.Fatal(err)
	}
	return service, store, q
}
func waitPrepared(t *testing.T, store TaskStore, stage TaskStage) *Task {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := store.Find(context.Background(), "file-task")
		if err != nil {
			t.Fatal(err)
		}
		if task.Stage == stage && task.StageStatus != TaskStageStatusRunning {
			return task
		}
		time.Sleep(5 * time.Millisecond)
	}
	task, _ := store.Find(context.Background(), "file-task")
	t.Fatalf("preparation did not finish: %+v", task)
	return nil
}
func TestFileSelectionUsesQBTIndicesAndSelectedProgress(t *testing.T) {
	files := []qbittorrent.TorrentContentFile{{Index: 7, Name: "OFJE-329A.mkv", Size: 100, Progress: .5}, {Index: 12, Name: "OFJE-329B.mkv", Size: 100, Progress: 1}, {Index: 22, Name: "OFJE-329A.en.srt", Size: 10}, {Index: 31, Name: "广告.mp4", Size: 100}}
	service, store, q := fileService(t, files, false, &controlledPreparer{})
	tasks, err := service.SyncProgress(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	task := tasks[0]
	if task.Stage != TaskStageDownloading || task.Progress != .75 {
		t.Fatalf("raw torrent completion hid incomplete main: %+v", task)
	}
	if q.priorities[7] != 1 || q.priorities[12] != 1 || q.priorities[22] != 0 || q.priorities[31] != 0 {
		t.Fatalf("wrong file priorities: %v", q.priorities)
	}
	if !reflect.DeepEqual(q.calls, []string{"pause", "priority", "priority", "resume"}) {
		t.Fatalf("unsafe selection sequence: %v", q.calls)
	}
	q.files[0].Progress = 1
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Find(context.Background(), task.ID)
	if task.Stage != TaskStagePendingIngest {
		t.Fatalf("selected files did not complete: %+v", task)
	}
}
func TestUnknownVideosPauseAndRequireManualOrderedSelection(t *testing.T) {
	service, store, q := fileService(t, []qbittorrent.TorrentContentFile{{Index: 5, Name: "mystery-a.mkv", Size: 100}, {Index: 9, Name: "mystery-b.mkv", Size: 100}}, false, &controlledPreparer{})
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	task, _ := store.Find(context.Background(), "file-task")
	if task.StageErrorCode != TaskStageErrorFileReview {
		t.Fatalf("expected actionable review: %+v", task)
	}
	if !reflect.DeepEqual(q.calls, []string{"pause"}) {
		t.Fatalf("uncertain files should not download: %v", q.calls)
	}
	task, err := service.ResolveTaskFiles(context.Background(), task.ID, []int{9, 5})
	if err != nil {
		t.Fatal(err)
	}
	videos := task.FilePlan().Videos()
	if videos[0].Index != 9 || videos[1].Index != 5 || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("manual decision lost: %+v", task)
	}
}
func TestPreparationRunsOutsideRequestAndPublishesCleanScanPath(t *testing.T) {
	preparer := &controlledPreparer{started: make(chan struct{}, 1), release: make(chan struct{})}
	service, store, _ := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329A.mkv", Size: 100, Progress: 1}, {Index: 1, Name: "OFJE-329B.mkv", Size: 100, Progress: 1}}, false, preparer)
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	tasks, err := service.TriggerStashScans(ctx, &fakeStashScanner{})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if tasks[0].Stage != TaskStageTransferring || tasks[0].StageStatus != TaskStageStatusRunning {
		t.Fatalf("expected background preparation: %+v", tasks[0])
	}
	select {
	case <-preparer.started:
	case <-time.After(time.Second):
		t.Fatal("preparation did not start")
	}
	close(preparer.release)
	task := waitPrepared(t, store, TaskStageScanning)
	if task.StashScanPath != "/stash/library/OFJE-329" || task.MojiTransferPath != "/moji/library/OFJE-329" {
		t.Fatalf("scan must use normalized delivery: %+v", task)
	}
	preparer.mu.Lock()
	defer preparer.mu.Unlock()
	if len(preparer.requests) != 1 || len(preparer.requests[0].Plan.Videos()) != 2 {
		t.Fatal("wrong preparation request")
	}
}
func TestPreparationFailureBlocksBeforeStashAndRetries(t *testing.T) {
	preparer := &controlledPreparer{err: errors.New("incompatible parts")}
	service, store, _ := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329.mkv", Size: 100, Progress: 1}}, false, preparer)
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	scanner := &fakeStashScanner{}
	if _, err := service.TriggerStashScans(context.Background(), scanner); err != nil {
		t.Fatal(err)
	}
	task := waitPrepared(t, store, TaskStageTransferring)
	if task.StageErrorCode != TaskStageErrorMediaPreparation || len(scanner.requests) != 0 {
		t.Fatalf("failed preparation reached Stash: %+v", task)
	}
	preparer.err = nil
	if _, err := service.RetryTask(context.Background(), task.ID, scanner); err != nil {
		t.Fatal(err)
	}
	waitPrepared(t, store, TaskStageScanning)
}
func TestFilePlanSurvivesSQLiteReload(t *testing.T) {
	store, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	now := time.Now().UTC()
	task := &Task{ID: "saved", Stage: TaskStageDownloading, CreatedAt: now, UpdatedAt: now}
	p := mediafiles.Classify("OFJE-329", []mediafiles.File{{Index: 8, Path: "OFJE-329A.mp4"}, {Index: 14, Path: "OFJE-329B.mp4"}}, true)
	if err := p.Select([]int{14, 8}); err != nil {
		t.Fatal(err)
	}
	task.setFilePlan(p)
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Find(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(task.FilePlan(), reloaded.FilePlan()) {
		t.Fatalf("file plan was lost: %+v", reloaded)
	}
}
func TestTorrentSubmissionStopsAtMetadataBeforeFiltering(t *testing.T) {
	service, _, q := fileService(t, nil, false, &controlledPreparer{})
	task := &Task{ID: "submission", Code: "OFJE-329", Stage: TaskStageDownloading}
	now := time.Now().UTC()
	task.CreatedAt, task.UpdatedAt = now, now
	if err := service.store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if err := service.submitTaskTorrent(context.Background(), task, "magnet:?xt=urn:btih:submission&dn=OFJE-329", "", "", "", nil); err != nil {
		t.Fatal(err)
	}
	if q.options.StopCondition == nil || *q.options.StopCondition != "MetadataReceived" {
		t.Fatal("magnet should stop after metadata acquisition")
	}
}

func TestPendingTransferCheckpointResumesAndDoesNotRepeatActivePreparation(t *testing.T) {
	preparer := &controlledPreparer{started: make(chan struct{}, 1), release: make(chan struct{})}
	service, store, _ := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329.mkv", Size: 100, Progress: 1}}, false, preparer)
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	task, _ := store.Find(context.Background(), "file-task")
	setTaskStage(task, TaskStageTransferring, TaskStageStatusPending)
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-preparer.started:
	case <-time.After(time.Second):
		t.Fatal("checkpoint did not resume")
	}
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	preparer.mu.Lock()
	count := len(preparer.requests)
	preparer.mu.Unlock()
	if count != 1 {
		t.Fatal("a second worker was started for the same task")
	}
	close(preparer.release)
	waitPrepared(t, store, TaskStageScanning)
}

func TestChangingSubtitlePolicyWaitsForNewSelectedFiles(t *testing.T) {
	cfg := mediaConfig()
	preparer := &controlledPreparer{}
	service, store, q := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329.mkv", Size: 100, Progress: 1}, {Index: 1, Name: "OFJE-329.srt", Size: 10, Progress: 0}}, false, preparer)
	service.mediaConfig = func() stashsync.IntegrationConfig { return cfg }
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	cfg.KeepSubtitles = true
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	task, _ := store.Find(context.Background(), "file-task")
	if task.Stage != TaskStageDownloading || q.priorities[1] != qbittorrent.TorrentFilePriorityNormal {
		t.Fatalf("subtitle policy change bypassed downloading: %+v %v", task, q.priorities)
	}
	preparer.mu.Lock()
	count := len(preparer.requests)
	preparer.mu.Unlock()
	if count != 0 {
		t.Fatal("preparation started with a missing subtitle")
	}
}

func TestPreparationQueueDoesNotSpawnAdditionalWorkers(t *testing.T) {
	preparer := &controlledPreparer{started: make(chan struct{}, 2), release: make(chan struct{})}
	service, store, _ := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329.mkv", Size: 100, Progress: 1}}, false, preparer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.SetMediaPreparationContext(ctx)
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-preparer.started:
	case <-time.After(time.Second):
		t.Fatal("first preparation did not start")
	}
	second, _ := store.Find(context.Background(), "file-task")
	second.ID, second.Code = "second", "SSIS-618"
	setTaskStage(second, TaskStagePendingIngest, TaskStageStatusPending)
	plan := mediafiles.Classify(second.Code, []mediafiles.File{{Index: 0, Path: "SSIS-618.mkv", Size: 100}}, false)
	plan.Applied = true
	second.setFilePlan(plan)
	if err := store.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	unlock := service.lockTask(second.ID)
	queued, err := service.startMediaPreparation(context.Background(), second)
	unlock()
	if err != nil {
		t.Fatal(err)
	}
	if queued.StageStatus != TaskStageStatusPending {
		t.Fatal("second task should stay in persisted queue")
	}
	service.mediaMu.Lock()
	count := len(service.mediaJobs)
	service.mediaMu.Unlock()
	if count != 1 {
		t.Fatalf("unbounded preparation workers: %d", count)
	}
	close(preparer.release)
	waitPrepared(t, store, TaskStageScanning)
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{jobID: "scan"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-preparer.started:
	case <-time.After(time.Second):
		t.Fatal("queued preparation did not resume")
	}
}

func TestShutdownLeavesRecoverablePreparationCheckpoint(t *testing.T) {
	preparer := &controlledPreparer{started: make(chan struct{}, 1), release: make(chan struct{})}
	service, store, q := fileService(t, []qbittorrent.TorrentContentFile{{Index: 0, Name: "OFJE-329.mkv", Size: 100, Progress: 1}}, false, preparer)
	ctx, cancel := context.WithCancel(context.Background())
	service.SetMediaPreparationContext(ctx)
	if _, err := service.SyncProgress(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-preparer.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	cancel()
	checkpoint, _ := store.Find(context.Background(), "file-task")
	if checkpoint.Stage != TaskStageTransferring {
		t.Fatal("missing restart checkpoint")
	}
	cfg := mediaConfig()
	resumed, err := NewService(fakeTracker{}, q, store, WithMediaPreparation(func() stashsync.IntegrationConfig { return cfg }, &controlledPreparer{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.TriggerStashScans(context.Background(), &fakeStashScanner{}); err != nil {
		t.Fatal(err)
	}
	waitPrepared(t, store, TaskStageScanning)
}
