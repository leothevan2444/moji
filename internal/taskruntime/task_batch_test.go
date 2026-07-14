package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestTaskBatchNormalizesIDsAndEnforcesLimit(t *testing.T) {
	ids, err := normalizeTaskBatchIDs([]string{" a ", "a", "b"})
	if err != nil || len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("unexpected normalized ids: %v, %v", ids, err)
	}
	if _, err := normalizeTaskBatchIDs(nil); !errors.Is(err, ErrTaskBatchEmpty) {
		t.Fatalf("expected empty batch error, got %v", err)
	}
	tooMany := make([]string, MaxTaskBatchSize+1)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("task-%d", index)
	}
	if _, err := normalizeTaskBatchIDs(tooMany); !errors.Is(err, ErrTaskBatchTooLarge) {
		t.Fatalf("expected batch limit error, got %v", err)
	}
}

func TestRetryTasksReturnsPerTaskResults(t *testing.T) {
	store := NewMemoryTaskStore()
	now := time.Unix(100, 0).UTC()
	blocked := &Task{ID: "blocked", Code: "ABCD-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, CreatedAt: now, UpdatedAt: now}
	completed := &Task{ID: "completed", Code: "EFGH-456", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), blocked); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), completed); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fakeTracker{results: []jackett.SearchResult{{Title: "ABCD-123", MagnetURI: "magnet:?xt=urn:btih:batch"}}}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatal(err)
	}

	payload, err := service.RetryTasks(context.Background(), []string{"blocked", "completed", "blocked"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Summary.RequestedCount != 2 || payload.Summary.SucceededCount != 1 || payload.Summary.SkippedCount != 1 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}
	if payload.Results[0].ReasonCode != TaskBatchReasonRetried || payload.Results[1].ReasonCode != TaskBatchReasonNotRetryable {
		t.Fatalf("unexpected results: %+v", payload.Results)
	}
}

func TestDeleteTasksKeepsMissingTaskAsSkipped(t *testing.T) {
	store := NewMemoryTaskStore()
	now := time.Unix(100, 0).UTC()
	if err := store.Create(context.Background(), &Task{ID: "delete-me", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := service.DeleteTasks(context.Background(), []string{"delete-me", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if payload.Summary.SucceededCount != 1 || payload.Summary.SkippedCount != 1 || payload.Results[1].ReasonCode != TaskBatchReasonTaskNotFound {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
