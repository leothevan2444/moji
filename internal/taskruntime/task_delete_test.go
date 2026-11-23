package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
)

type fakeSceneDeleter struct {
	calls    int
	err      error
	onDelete func()
	identity string
}

func (f *fakeSceneDeleter) DeletionIdentity() string {
	if f.identity != "" {
		return f.identity
	}
	return "stash-account"
}
func (f *fakeSceneDeleter) DeleteTaskScenes(_ context.Context, identity, path string, ids []string) error {
	f.calls++
	if identity != "stash-account" || path != "/library/DEMO-123" || !reflect.DeepEqual(ids, []string{"12"}) {
		return errors.New("wrong deletion target")
	}
	if f.onDelete != nil {
		f.onDelete()
	}
	return f.err
}
func deletionTask(id string) *Task {
	return &Task{ID: id, Code: "DEMO-123", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, TorrentHash: "torrent", StashScanPath: "/library/DEMO-123", StashSceneIDsJSON: `["12"]`, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func TestStashDeletionOptionIsIndependentOfDownloaderPolicy(t *testing.T) {
	for _, policy := range []config.TaskDeletePolicy{config.TaskDeletePolicyKeepOnly, config.TaskDeletePolicyRemoveTorrent, config.TaskDeletePolicyRemoveTorrentAndFiles} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", policy, enabled), func(t *testing.T) {
				store := NewMemoryTaskStore()
				if err := store.Create(context.Background(), deletionTask("task")); err != nil {
					t.Fatal(err)
				}
				qbt := &fakeTorrentAdder{}
				stash := &fakeSceneDeleter{}
				s, _ := NewService(fakeTracker{}, qbt, store, WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy { return policy }))
				if _, err := s.DeleteTaskWithOptions(context.Background(), "task", DeleteTaskOptions{DeleteStashScene: enabled}, stash); err != nil {
					t.Fatal(err)
				}
				if (stash.calls == 1) != enabled || (len(qbt.deleteHashes) > 0) != (policy != config.TaskDeletePolicyKeepOnly) || qbt.deleteFiles != (policy == config.TaskDeletePolicyRemoveTorrentAndFiles) {
					t.Fatalf("options leaked across policies: stash=%d qbt=%v files=%t", stash.calls, qbt.deleteHashes, qbt.deleteFiles)
				}
			})
		}
	}
}
func TestStashDeletionFailureKeepsTaskAndDownloader(t *testing.T) {
	store := NewMemoryTaskStore()
	_ = store.Create(context.Background(), deletionTask("task"))
	qbt := &fakeTorrentAdder{}
	stash := &fakeSceneDeleter{err: errors.New("Stash unreachable")}
	s, _ := NewService(fakeTracker{}, qbt, store, WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy { return config.TaskDeletePolicyRemoveTorrentAndFiles }))
	if _, err := s.DeleteTaskWithOptions(context.Background(), "task", DeleteTaskOptions{DeleteStashScene: true}, stash); err == nil {
		t.Fatal("expected failure")
	}
	if task, err := store.Find(context.Background(), "task"); err != nil || task == nil || len(qbt.deleteHashes) > 0 {
		t.Fatal("failure removed task or torrent")
	}
	// A subsequent ordinary deletion never inherits the previous checkbox.
	if _, err := s.DeleteTask(context.Background(), "task"); err != nil || stash.calls != 1 {
		t.Fatal("checkbox persisted", err)
	}
}
func TestStashDeletionWithoutLinkedScenesNeedsNoConnection(t *testing.T) {
	store := NewMemoryTaskStore()
	task := deletionTask("task")
	task.StashSceneIDsJSON = ""
	_ = store.Create(context.Background(), task)
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if _, err := s.DeleteTaskWithOptions(context.Background(), "task", DeleteTaskOptions{DeleteStashScene: true}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStashDeletionIntentIsDurableAndCannotRetargetAfterRestart(t *testing.T) {
	store, manager := relationalStore(t)
	ctx := context.Background()
	task := deletionTask("task-journal")
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	stash := &fakeSceneDeleter{err: errors.New("response lost")}
	stash.onDelete = func() {
		raw, err := store.TaskCommandPayload(ctx, task, "TASK_DELETE_STASH")
		var payload struct {
			Intent stashDeletionIntent `json:"intent"`
		}
		if err != nil || json.Unmarshal(raw, &payload) != nil || payload.Intent.Identity != "stash-account" || !reflect.DeepEqual(payload.Intent.SceneIDs, []string{"12"}) {
			t.Fatal("deletion preceded its durable intent", err)
		}
	}
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if _, err := s.DeleteTaskWithOptions(ctx, task.ID, DeleteTaskOptions{DeleteStashScene: true}, stash); err == nil {
		t.Fatal("lost response hidden")
	}
	reopened := NewRelationalTaskStore(manager.Core, manager.Runtime, manager.Observations)
	s, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, reopened)
	stash.identity = "another-account"
	stash.err = nil
	stash.onDelete = nil
	if _, err := s.DeleteTaskWithOptions(ctx, task.ID, DeleteTaskOptions{DeleteStashScene: true}, stash); err == nil || stash.calls != 1 {
		t.Fatal("retry retargeted another Stash account")
	}
	stash.identity = "stash-account"
	if _, err := s.DeleteTaskWithOptions(ctx, task.ID, DeleteTaskOptions{DeleteStashScene: true}, stash); err != nil || stash.calls != 2 {
		t.Fatal("recorded deletion could not resume", err)
	}
}

func TestBatchStashDeletionPreservesFailures(t *testing.T) {
	store := NewMemoryTaskStore()
	_ = store.Create(context.Background(), deletionTask("linked"))
	unlinked := deletionTask("unlinked")
	unlinked.StashSceneIDsJSON = ""
	unlinked.TorrentHash = ""
	unlinked.Code = "OTHER-123"
	_ = store.Create(context.Background(), unlinked)
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	// An upstream 'not found' error is a failed lookup, not proof the Moji task disappeared.
	payload, err := s.DeleteTasksWithOptions(context.Background(), []string{"linked", "unlinked"}, DeleteTaskOptions{DeleteStashScene: true}, &fakeSceneDeleter{err: errors.New("remote service not found")})
	if err != nil || payload.Summary.SucceededCount != 1 || payload.Summary.FailedCount != 1 || payload.Summary.SkippedCount != 0 {
		t.Fatalf("incorrect partial result: %+v %v", payload, err)
	}
	if task, err := store.Find(context.Background(), "linked"); task == nil || err != nil {
		t.Fatal("failed task lost")
	}
}
