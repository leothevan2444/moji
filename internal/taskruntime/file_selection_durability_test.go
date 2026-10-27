package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/mediafiles"
)

type durableSelectionTorrent struct {
	fileTorrent
	store       TaskStore
	resumeErr   error
	beforePause func(*Task)
}

func (q *durableSelectionTorrent) PauseTorrents(ctx context.Context, hashes []string) error {
	task, err := q.store.Find(ctx, "selection")
	if err != nil {
		return err
	}
	if q.beforePause != nil {
		q.beforePause(task)
	}
	return q.fileTorrent.PauseTorrents(ctx, hashes)
}
func (q *durableSelectionTorrent) ResumeTorrents(ctx context.Context, hashes []string) error {
	if q.resumeErr != nil {
		return q.resumeErr
	}
	return q.fileTorrent.ResumeTorrents(ctx, hashes)
}
func TestFileChoiceIsDurableBeforeExternalPriorityChanges(t *testing.T) {
	store, m := relationalStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	task := &Task{ID: "selection", Code: "TEST-123", Stage: TaskStageDownloading, StageStatus: TaskStageStatusBlocked, StageErrorCode: TaskStageErrorFileReview, TorrentHash: "hash", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	q := &durableSelectionTorrent{store: store, resumeErr: errors.New("resume response lost")}
	q.beforePause = func(saved *Task) {
		plan := saved.FilePlan()
		if len(plan.Files) != 1 || plan.Files[0].Index != 7 || !plan.Confirmed || plan.Applied || saved.StageStatus != TaskStageStatusRunning {
			t.Fatal("qB changed before the choice checkpoint was committed")
		}
	}
	service, err := NewService(fakeTracker{}, q, store)
	if err != nil {
		t.Fatal(err)
	}
	plan := mediafiles.Plan{Confirmed: true, Files: []mediafiles.File{{Index: 7, Path: "TEST-123.mp4", Size: 100, Kind: mediafiles.Main, Selected: true}}}
	if err := service.applyFileSelection(ctx, task, &plan); !errors.Is(err, q.resumeErr) {
		t.Fatal("expected ambiguous apply result", err)
	}
	saved, err := store.Find(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.FilePlan().Confirmed || saved.FilePlan().Applied {
		t.Fatal("failed application lost the chosen files or claimed completion")
	}
	// Fail the next intent write. No external action may run at all.
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER no_choice BEFORE UPDATE ON task_file_plans BEGIN SELECT RAISE(ABORT,'disk full'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	q.calls = nil
	plan.KeepSubtitles = true
	if err := service.applyFileSelection(ctx, task, &plan); err == nil {
		t.Fatal("failed choice write ignored")
	}
	if len(q.calls) != 0 {
		t.Fatal("qB changed despite a failed choice commit")
	}
}
