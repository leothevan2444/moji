package taskruntime

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

type fakeIngestScanner struct {
	fakeStashScanner
	calls       []string
	expectedIDs []string
	onGenerate  func()
	submitted   [][]string
	generateErr error
	identifyErr error
	scenes      []stashsync.IngestScene
	jobs        map[string]*stashsync.Job
}

func (f *fakeIngestScanner) MetadataGenerate(_ context.Context, ids []string) (string, error) {
	f.calls = append(f.calls, "generate")
	expected := f.expectedIDs
	if expected == nil {
		expected = []string{"scene-1"}
	}
	if !reflect.DeepEqual(ids, expected) {
		return "", errors.New("incorrect scene scope")
	}
	f.submitted = append(f.submitted, append([]string(nil), ids...))
	err := f.generateErr
	if f.onGenerate != nil {
		f.onGenerate()
	}
	return "generate-1", err
}
func (f *fakeIngestScanner) MetadataIdentify(_ context.Context, ids []string) (string, error) {
	f.calls = append(f.calls, "identify")
	expected := f.expectedIDs
	if expected == nil {
		expected = []string{"scene-1"}
	}
	if !reflect.DeepEqual(ids, expected) {
		return "", errors.New("incorrect scene scope")
	}
	return "identify-1", f.identifyErr
}
func (f *fakeIngestScanner) FindIngestScenes(_ context.Context, _ string, _ []string) ([]stashsync.IngestScene, error) {
	f.calls = append(f.calls, "verify")
	return f.scenes, nil
}
func (f *fakeIngestScanner) FindJob(_ context.Context, id string) (*stashsync.Job, error) {
	if f.jobErr != nil {
		return nil, f.jobErr
	}
	if f.jobs != nil {
		return f.jobs[id], nil
	}
	return f.job, nil
}
func readyIngestScanner() *fakeIngestScanner {
	return &fakeIngestScanner{
		fakeStashScanner: fakeStashScanner{jobID: "scan-1", config: stashsync.IntegrationConfig{
			Downloads: stashsync.DownloadsPathConfig{QBRoot: "/downloads", MojiRoot: "/moji/downloads"},
			Library:   stashsync.LibraryPathConfig{StashRoot: "/library", MojiRoot: "/moji/library"},
			Transfer:  stashsync.TransferConfig{Action: stashsync.TransferActionMove},
		}},
		scenes: []stashsync.IngestScene{{ID: "scene-1", Code: "ABCD-123", Title: "Movie", Date: "2026-01-01", StudioID: "studio-1", PerformerCount: 1, Paths: []string{"/library/ABCD-123.mp4"}}},
		jobs: map[string]*stashsync.Job{
			"scan-1":     {ID: "scan-1", Status: "FINISHED"},
			"generate-1": {ID: "generate-1", Status: "FINISHED"},
			"identify-1": {ID: "identify-1", Status: "FINISHED"},
		},
	}
}
func createIngestTask(t *testing.T, store TaskStore) {
	t.Helper()
	now := time.Now().UTC()
	if err := store.Create(context.Background(), &Task{ID: "ingest", Code: "ABCD-123", Stage: TaskStagePendingIngest, StageStatus: TaskStageStatusPending, ContentPath: "/downloads/ABCD-123.mp4", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
}
func advanceIngest(t *testing.T, service *Service, scanner StashScanner) *Task {
	t.Helper()
	tasks, err := service.TriggerStashScans(context.Background(), scanner)
	if err != nil {
		t.Fatal(err)
	}
	return tasks[0]
}

func TestStashIngestRunsAllStepsAndSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	createIngestTask(t, store)
	files := &fakeFileOperator{}
	scanner := readyIngestScanner()
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	task := advanceIngest(t, service, scanner)
	if task.StashIngestPhase != StashIngestScan {
		t.Fatalf("unexpected phase: %+v", task)
	}
	task = advanceIngest(t, service, scanner)
	if task.Stage == TaskStageCompleted || task.IngestCompletedAt != nil || task.StashGenerateJobID != "generate-1" {
		t.Fatalf("scan must not complete ingest: %+v", task)
	}
	// Re-open storage to exercise real checkpoint serialization.
	store.db.Close()
	reopened, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.db.Close()
	completedAt := time.Date(2026, 10, 18, 6, 30, 0, 0, time.UTC)
	service, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, reopened, WithFileOperator(files), WithClock(func() time.Time { return completedAt }))
	task = advanceIngest(t, service, scanner)
	if task.IngestCompletedAt != nil || task.StashIngestPhase != StashIngestIdentify || task.StashIdentifyJobID != "identify-1" {
		t.Fatalf("unexpected phase after restart: %+v", task)
	}
	task = advanceIngest(t, service, scanner)
	if task.Stage != TaskStageCompleted || task.StageStatus != TaskStageStatusDone {
		t.Fatalf("unexpected completion: %+v", task)
	}
	if task.IngestCompletedAt == nil || !task.IngestCompletedAt.Equal(completedAt) {
		t.Fatalf("completion timestamp not recorded after verification: %+v", task)
	}
	reopened.db.Close()
	reopened, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.db.Close()
	stored, err := reopened.Find(context.Background(), "ingest")
	if err != nil || stored.IngestCompletedAt == nil || !stored.IngestCompletedAt.Equal(completedAt) {
		t.Fatalf("completion timestamp not persisted: %+v %v", stored, err)
	}
	copy := cloneTask(stored)
	*copy.IngestCompletedAt = completedAt.Add(time.Hour)
	if !stored.IngestCompletedAt.Equal(completedAt) {
		t.Fatal("timestamp pointer is shared")
	}
	service, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, reopened, WithClock(func() time.Time { return completedAt.Add(time.Hour) }))
	advanceIngest(t, service, scanner)
	stored, _ = reopened.Find(context.Background(), "ingest")
	if !stored.IngestCompletedAt.Equal(completedAt) {
		t.Fatal("completed polling changed timestamp")
	}
	if len(files.calls) != 1 || len(scanner.requests) != 1 {
		t.Fatal("delivery and scan must each run once")
	}
	if !reflect.DeepEqual(scanner.calls, []string{"verify", "generate", "verify", "identify", "verify"}) {
		t.Fatalf("unexpected order: %v", scanner.calls)
	}
}

func TestStashIngestRetriesOnlyFailedStep(t *testing.T) {
	for _, step := range []string{"scan", "generate", "identify", "verify", "poll"} {
		t.Run(step, func(t *testing.T) {
			store := NewMemoryTaskStore()
			createIngestTask(t, store)
			files := &fakeFileOperator{}
			scanner := readyIngestScanner()
			service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
			if step == "scan" {
				scanner.err = errors.New("scan unavailable")
			}
			if step == "generate" {
				scanner.generateErr = errors.New("generate unavailable")
			}
			if step == "identify" {
				scanner.identifyErr = errors.New("sources missing")
			}
			if step == "verify" {
				scanner.scenes[0].PerformerCount = 0
			}
			if step == "poll" {
				scanner.jobErr = errors.New("temporary poll failure")
			}
			var err error
			for range 4 {
				_, err = service.TriggerStashScans(context.Background(), scanner)
				if err != nil {
					break
				}
			}
			if err == nil {
				t.Fatal("expected blocked ingest")
			}
			task, _ := store.Find(context.Background(), "ingest")
			if task.IngestCompletedAt != nil || task.StageStatus != TaskStageStatusBlocked {
				t.Fatalf("expected blocked: %+v", task)
			}
			scanner.err, scanner.generateErr, scanner.identifyErr, scanner.jobErr = nil, nil, nil, nil
			scanner.scenes[0].PerformerCount = 1
			if _, err := service.RetryTask(context.Background(), "ingest", scanner); err != nil {
				t.Fatal(err)
			}
			for range 4 {
				task = advanceIngest(t, service, scanner)
			}
			if task.Stage != TaskStageCompleted || task.IngestCompletedAt == nil {
				t.Fatalf("expected complete after retry: %+v", task)
			}
			if len(files.calls) != 1 {
				t.Fatal("retry repeated delivery")
			}
			if step != "scan" && len(scanner.requests) != 1 {
				t.Fatal("retry repeated successful scan")
			}
		})
	}
}

func TestFinishedIdentifyJobDoesNotHideMissingOrWrongMetadata(t *testing.T) {
	for _, mutation := range []func(*stashsync.IngestScene){
		func(s *stashsync.IngestScene) { s.Title = "" },
		func(s *stashsync.IngestScene) { s.Code = "WRONG-999" },
		func(s *stashsync.IngestScene) { s.Date = "" },
		func(s *stashsync.IngestScene) { s.StudioID = "" },
		func(s *stashsync.IngestScene) { s.PerformerCount = 0 },
		func(s *stashsync.IngestScene) { s.Paths = nil },
	} {
		scanner := readyIngestScanner()
		mutation(&scanner.scenes[0])
		if err := verifyIngestScenes(&Task{Code: "ABCD-123"}, []string{"scene-1"}, scanner.scenes); err == nil {
			t.Fatal("incomplete result should fail verification")
		}
	}
	if err := verifyIngestScenes(&Task{}, []string{"missing"}, nil); err == nil {
		t.Fatal("missing scenes must fail")
	}
}

func TestFailedStashJobCanBeResubmittedWithoutTransfer(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	files := &fakeFileOperator{}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	advanceIngest(t, service, scanner)
	advanceIngest(t, service, scanner)
	scanner.jobs["generate-1"].Status = "CANCELLED"
	if _, err := service.TriggerStashScans(context.Background(), scanner); err == nil {
		t.Fatal("expected job failure")
	}
	task, _ := store.Find(context.Background(), "ingest")
	if task.StashGenerateJobID != "" || task.StageErrorCode != TaskStageErrorGenerate {
		t.Fatalf("incorrect failed checkpoint: %+v", task)
	}
	scanner.jobs["generate-1"].Status = "FINISHED"
	if _, err := service.RetryTask(context.Background(), "ingest", scanner); err != nil {
		t.Fatal(err)
	}
	advanceIngest(t, service, scanner)
	advanceIngest(t, service, scanner)
	if len(files.calls) != 1 {
		t.Fatal("retry transferred again")
	}
}

func TestStashIngestResumesAfterDeliveryCheckpoint(t *testing.T) {
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	if err := store.Create(context.Background(), &Task{ID: "checkpoint", Stage: TaskStageScanning, StageStatus: TaskStageStatusPending, StashIngestPhase: StashIngestScan, StashScanPath: "/library/ABCD-123.mp4", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	scanner := readyIngestScanner()
	files := &fakeFileOperator{}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	task := advanceIngest(t, service, scanner)
	if task.StashScanJobID != "scan-1" || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("checkpoint did not resume: %+v", task)
	}
	if len(files.calls) != 0 {
		t.Fatal("recovery repeated delivery")
	}
}

func TestSceneReplacementRestartsGenerationWithoutDelivery(t *testing.T) {
	for _, phase := range []string{StashIngestGenerate, StashIngestIdentify, StashIngestVerify} {
		t.Run(phase, func(t *testing.T) {
			store := NewMemoryTaskStore()
			scanner := readyIngestScanner()
			scanner.scenes[0].ID = "scene-2"
			scanner.expectedIDs = []string{"scene-2"}
			now := time.Now().UTC()
			task := &Task{ID: "replace", Code: "ABCD-123", Stage: TaskStageScanning, StageStatus: TaskStageStatusBlocked, StashIngestPhase: phase, StashSceneIDsJSON: `["scene-1"]`, StashScanPath: "/library/ABCD-123.mp4", StashGenerateJobID: "old-gen", StashIdentifyJobID: "old-identify", CreatedAt: now, UpdatedAt: now}
			// A failed job has already been cleared when a task is blocked.
			setStashJobID(task, "")
			if err := store.Create(context.Background(), task); err != nil {
				t.Fatal(err)
			}
			files := &fakeFileOperator{}
			service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
			got, err := service.RetryTask(context.Background(), task.ID, scanner)
			if err != nil {
				t.Fatal(err)
			}
			if got.StashIngestPhase != StashIngestGenerate || got.StashGenerateJobID != "generate-1" || got.StashIdentifyJobID != "" || got.StashSceneRecoveryCount != 1 {
				t.Fatalf("invalid recovery: %+v", got)
			}
			got = advanceIngest(t, service, scanner)
			got = advanceIngest(t, service, scanner)
			if got.Stage != TaskStageCompleted || len(files.calls) != 0 || len(scanner.requests) != 0 {
				t.Fatalf("recovery did not finish independently: %+v", got)
			}
		})
	}
}

func TestPartialSceneReplacementAndPathBoundary(t *testing.T) {
	scanner := readyIngestScanner()
	good := scanner.scenes[0]
	good.ID = "new"
	outside := good
	outside.ID, outside.Paths = "outside", []string{"/library/ABCD-123.mp4-other/movie.mp4"}
	scanner.scenes = []stashsync.IngestScene{outside, good}
	scanner.expectedIDs = []string{"new"}
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	task := &Task{ID: "partial", Stage: TaskStageScanning, StageStatus: TaskStageStatusPending, StashIngestPhase: StashIngestIdentify, StashSceneIDsJSON: `["old","new"]`, StashScanPath: "/library/ABCD-123.mp4", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	got := advanceIngest(t, service, scanner)
	if !reflect.DeepEqual(got.StashSceneIDs(), []string{"new"}) || got.StashIngestPhase != StashIngestGenerate {
		t.Fatalf("wrong scope after partial replacement: %+v", got)
	}
}

func TestSceneDisappearsAfterPreflightRecoversOnceAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "recover.db")
	store, err := NewSQLiteTaskStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.db.Close() }()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	files := &fakeFileOperator{}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	advanceIngest(t, service, scanner)
	scanner.onGenerate = func() {
		scanner.scenes[0].ID = "scene-2"
		scanner.expectedIDs = []string{"scene-2"}
		scanner.onGenerate = nil
	}
	advanceIngest(t, service, scanner)
	message := "error encountered while generating: scene with id scene-1 not found"
	scanner.jobs["generate-1"] = &stashsync.Job{Status: "FAILED", Error: &message}
	got := advanceIngest(t, service, scanner)
	if got.StashSceneRecoveryCount != 1 || !reflect.DeepEqual(got.StashSceneIDs(), []string{"scene-2"}) {
		t.Fatalf("race recovery failed: %+v", got)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(files))
	// Another missing-scene failure must not automatically resubmit forever.
	_, err = service.TriggerStashScans(context.Background(), scanner)
	if err == nil {
		t.Fatal("expected recovery limit")
	}
	got, _ = store.Find(context.Background(), "ingest")
	if got.StageErrorCode != TaskStageErrorSceneChanged || got.StageStatus != TaskStageStatusBlocked || len(scanner.submitted) != 2 {
		t.Fatalf("recovery budget not persisted: %+v submissions=%v", got, scanner.submitted)
	}
	scanner.jobs["generate-1"].Status = "FINISHED"
	if _, err = service.RetryTask(context.Background(), "ingest", scanner); err != nil {
		t.Fatal(err)
	}
	got = advanceIngest(t, service, scanner)
	got = advanceIngest(t, service, scanner)
	if got.Stage != TaskStageCompleted || len(files.calls) != 1 || len(scanner.requests) != 1 {
		t.Fatalf("manual retry failed: %+v", got)
	}
}

func TestNoScenesBlocksWithoutSubmittingJobs(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	scanner.scenes = nil
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	advanceIngest(t, service, scanner)
	if _, err := service.TriggerStashScans(context.Background(), scanner); err == nil {
		t.Fatal("expected no-scenes failure")
	}
	task, _ := store.Find(context.Background(), "ingest")
	if task.StageErrorCode != TaskStageErrorSceneMissing || len(scanner.submitted) != 0 {
		t.Fatalf("unexpected failure: %+v", task)
	}
}

func TestSceneMissingAtSubmissionRecoversImmediately(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	advanceIngest(t, service, scanner)
	scanner.generateErr = errors.New("input: metadataGenerate scene with id scene-1 not found")
	scanner.onGenerate = func() {
		scanner.scenes[0].ID = "scene-2"
		scanner.expectedIDs = []string{"scene-2"}
		scanner.generateErr = nil
		scanner.onGenerate = nil
	}
	got := advanceIngest(t, service, scanner)
	if got.StashSceneRecoveryCount != 1 || len(scanner.submitted) != 2 || got.StageStatus != TaskStageStatusRunning {
		t.Fatalf("submission race not recovered: %+v", got)
	}
}

func TestSceneOrderDoesNotRestartSuccessfulGeneration(t *testing.T) {
	scanner := readyIngestScanner()
	second := scanner.scenes[0]
	second.ID = "scene-2"
	scanner.scenes = []stashsync.IngestScene{second, scanner.scenes[0]}
	scanner.expectedIDs = []string{"scene-2", "scene-1"}
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	task := &Task{ID: "order", Stage: TaskStageScanning, StageStatus: TaskStageStatusPending, StashIngestPhase: StashIngestIdentify, StashSceneIDsJSON: `["scene-2","scene-1"]`, StashScanPath: "/library/ABCD-123.mp4", StashGenerateJobID: "already-generated", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	got := advanceIngest(t, service, scanner)
	if got.StashIngestPhase != StashIngestIdentify || got.StashSceneRecoveryCount != 0 || len(scanner.submitted) != 0 {
		t.Fatalf("reordered scenes caused regeneration: %+v", got)
	}
}

func TestIdentifySourcesHaveSpecificFailureCode(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	scanner.identifyErr = stashsync.ErrIdentifySourcesMissing
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	advanceIngest(t, service, scanner)
	advanceIngest(t, service, scanner)
	if _, err := service.TriggerStashScans(context.Background(), scanner); err == nil {
		t.Fatal("expected missing sources")
	}
	task, _ := store.Find(context.Background(), "ingest")
	if task.StageErrorCode != TaskStageErrorSourcesMissing || task.StashSceneRecoveryCount != 0 {
		t.Fatalf("wrong classification: %+v", task)
	}
}
