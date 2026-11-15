package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFilmSnapshotRoundTripIdentityAndImmutability(t *testing.T) {
	store, _ := relationalStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	target := testFilm("300MIUM-1437")
	task := &Task{ID: "film-task", Film: target, Code: target.Code, SourcingMode: "AUTO", Stage: TaskStageSourcing, StageStatus: TaskStageStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.FindByFilm(ctx, *target)
	if err != nil || loaded.Film == nil || loaded.Film.Code != target.Code {
		t.Fatal("film snapshot lost", err)
	}
	copy := cloneTask(loaded)
	copy.Film.SceneID = "other"
	if err := store.Update(ctx, copy); err == nil {
		t.Fatal("task film was replaced")
	}
	duplicate := cloneTask(task)
	duplicate.ID = "duplicate"
	duplicate.Code = "T38-042"
	duplicate.Film.Code = duplicate.Code
	if err := store.Create(ctx, duplicate); !errors.Is(err, ErrDuplicateCodeTask) {
		t.Fatal("same scene acquired a second task", err)
	}
	copy = cloneTask(task)
	copy.ID = "same-code"
	copy.Film.SceneID = "another-scene"
	if err := store.Create(ctx, copy); !errors.Is(err, ErrDuplicateCodeTask) {
		t.Fatal("same code acquired a second task", err)
	}
}
