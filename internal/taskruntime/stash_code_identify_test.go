package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/stashsync"
)

type codeIngestScanner struct {
	*fakeIngestScanner
	prepares, applies    int
	plan                 *stashsync.CodeIdentifyPlan
	prepareErr, applyErr error
	onApply              func(*stashsync.CodeIdentifyPlan)
}

func (f *codeIngestScanner) PrepareFilmIdentify(_ context.Context, target film.Snapshot, _ string, _ []string) (*stashsync.CodeIdentifyPlan, error) {
	f.prepares++
	if f.plan != nil {
		f.plan.Endpoint = target.Endpoint
	}
	return f.plan, f.prepareErr
}
func (f *codeIngestScanner) ApplyCodeIdentify(_ context.Context, _ string, _ []string, plan *stashsync.CodeIdentifyPlan) error {
	f.applies++
	if f.onApply != nil {
		f.onApply(plan)
	}
	return f.applyErr
}

func TestCodeFallbackCompletesOnlyAfterReadingSavedMetadata(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := &codeIngestScanner{fakeIngestScanner: readyIngestScanner(), plan: &stashsync.CodeIdentifyPlan{Version: 2, Endpoint: "https://example.test/graphql", Code: "ABCD-123", RemoteID: "chosen"}}
	scanner.scenes[0].Title = ""
	scanner.onApply = func(*stashsync.CodeIdentifyPlan) { scanner.scenes[0].Title = "Saved by Stash" }
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	for range 4 {
		advanceIngest(t, service, scanner)
	}
	task, _ := store.Find(context.Background(), "ingest")
	if task.Stage != TaskStageCompleted || task.IngestCompletedAt == nil || scanner.prepares != 1 || scanner.applies != 1 {
		t.Fatalf("fallback did not finish: %+v", task)
	}
}

func TestCodeFallbackPersistsChoiceAndReusesItAfterRestart(t *testing.T) {
	store, manager := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.Code = "ABCD-123"
	task.Film = ingestFilm()
	task.Film.SceneID = "original"
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	scanner := &codeIngestScanner{fakeIngestScanner: readyIngestScanner(), plan: &stashsync.CodeIdentifyPlan{Version: 2, Code: "ABCD-123", RemoteID: "original"}, applyErr: errors.New("response lost")}
	scanner.onApply = func(plan *stashsync.CodeIdentifyPlan) {
		raw, err := store.TaskCommandPayload(ctx, task, stashCodeIdentifyCommand)
		if err != nil || len(raw) == 0 {
			t.Fatal("save ran before selection was durable", err)
		}
		var envelope struct {
			Plan stashsync.CodeIdentifyPlan `json:"plan"`
		}
		if json.Unmarshal(raw, &envelope) != nil || envelope.Plan.RemoteID != "original" || plan.RemoteID != "original" {
			t.Fatal("selected identity changed")
		}
	}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if _, err := service.tryStashCodeIdentify(ctx, task, scanner, true); err == nil {
		t.Fatal("lost response hidden")
	}
	// Rebuild the production store and event wrapper. The next search would
	// return a different scene, but recovery must use the recorded selection.
	reopened := NewRelationalTaskStore(manager.Core, manager.Observations)
	wrapped, err := NewEventingTaskStore(reopened, NewTaskEventBus(16))
	if err != nil {
		t.Fatal(err)
	}
	service, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, wrapped)
	scanner.plan = &stashsync.CodeIdentifyPlan{Version: 2, Code: "ABCD-123", RemoteID: "changed-search"}
	scanner.applyErr = nil
	if applied, err := service.tryStashCodeIdentify(ctx, task, scanner, true); err != nil || !applied || scanner.prepares != 1 {
		t.Fatalf("recovery searched again: %v", err)
	}
	// A new task generation must not inherit the previous generation's choice.
	other := cloneTask(task)
	other.Generation++
	if raw, err := reopened.TaskCommandPayload(ctx, other, stashCodeIdentifyCommand); err != nil || len(raw) != 0 {
		t.Fatal("old generation was reused", err)
	}
}

func TestCodeFallbackBlocksAmbiguityWithoutSaving(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := &codeIngestScanner{fakeIngestScanner: readyIngestScanner(), prepareErr: stashsync.ErrCodeAmbiguous}
	scanner.scenes[0].Title = ""
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	for range 3 {
		advanceIngest(t, service, scanner)
	}
	_, err := service.TriggerStashScans(context.Background(), scanner)
	task, _ := store.Find(context.Background(), "ingest")
	if err == nil || task.StageErrorCode != "STASH_CODE_AMBIGUOUS" || scanner.applies != 0 {
		t.Fatalf("ambiguous result reached save: %+v", task)
	}
}

func TestCodeCommandConfirmationCommitsWithVerifiedTask(t *testing.T) {
	store, manager := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := store.RunTaskCommand(ctx, task, stashCodeIdentifyCommand, taskCommandTarget(task, stashCodeIdentifyCommand), nil, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	setTaskStage(task, TaskStageCompleted, TaskStageStatusDone)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := manager.Core.Reader.Get(&state, `SELECT state FROM external_commands WHERE kind='STASH_CODE_IDENTIFY'`); err != nil || state != "CONFIRMED" {
		t.Fatalf("verified save left unconfirmed: %s %v", state, err)
	}
}

func TestCodePlanMustFinishEvenWhenRequiredMetadataAlreadyExists(t *testing.T) {
	store, _ := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.Code = "ABCD-123"
	task.Film = ingestFilm()
	task.Stage = TaskStageScanning
	task.StageStatus = TaskStageStatusRunning
	task.StageErrorCode = ""
	task.StageErrorMessage = ""
	task.StashIngestPhase = StashIngestVerify
	task.StashSceneIDsJSON = `["scene-1"]`
	task.StashScanPath = "/library/ABCD-123.mp4"
	task.IngestCompletedAt = nil
	task.RestartStateJSON = ""
	if e := store.Create(ctx, task); e != nil {
		t.Fatal(e)
	}
	plan := &stashsync.CodeIdentifyPlan{Version: 2, Endpoint: "https://example.test/graphql", Code: "ABCD-123", RemoteID: "chosen"}
	if e := store.RunTaskCommand(ctx, task, stashCodeIdentifyCommand, taskCommandTarget(task, stashCodeIdentifyCommand), map[string]any{"plan": plan}, func() error { return nil }); e != nil {
		t.Fatal(e)
	}
	scanner := &codeIngestScanner{fakeIngestScanner: readyIngestScanner(), applyErr: errors.New("cover verification failed")}
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	_, e := service.TriggerStashScans(ctx, scanner)
	blocked, _ := store.Find(ctx, task.ID)
	if e == nil || blocked.StageStatus != TaskStageStatusBlocked || blocked.IngestCompletedAt != nil || scanner.applies != 1 || scanner.prepares != 0 {
		t.Fatalf("scalar completeness hid an unfinished cover save: %+v %v", blocked, e)
	}
	scanner.applyErr = nil
	if _, e = service.RetryTask(ctx, task.ID, scanner); e != nil {
		t.Fatal(e)
	}
	if _, e = service.TriggerStashScans(ctx, scanner); e != nil {
		t.Fatal(e)
	}
	completed, _ := store.Find(ctx, task.ID)
	if completed.Stage != TaskStageCompleted || scanner.prepares != 0 {
		t.Fatal("retry did not reuse and finish the recorded full save")
	}
}
