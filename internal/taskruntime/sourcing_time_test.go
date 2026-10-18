package taskruntime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestSourcingCompletionRequiresSuccessfulTorrentSubmission(t *testing.T) {
	for _, mode := range []string{"search", "manual", "resolve"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			clock := time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC)
			qbt := &fakeTorrentAdder{err: errors.New("downloader unavailable")}
			store := NewMemoryTaskStore()
			url := "magnet:?xt=urn:btih:abc123&dn=ABCD-123"
			service, err := NewService(fakeTracker{results: []jackett.SearchResult{{Title: "ABCD-123", MagnetURI: url, Seeders: 10}}}, qbt, store, WithClock(func() time.Time { return clock }), WithIDGenerator(func() string { return "sourcing" }))
			if err != nil {
				t.Fatal(err)
			}
			var task *Task
			switch mode {
			case "search":
				task, err = service.DownloadMediaContext(ctx, DownloadRequest{Code: "ABCD-123"})
			case "manual":
				task, err = service.AddTorrentContext(ctx, AddTorrentRequest{URL: url})
			case "resolve":
				if err := store.Create(ctx, &Task{ID: "sourcing", Code: "ABCD-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, CreatedAt: clock, UpdatedAt: clock}); err != nil {
					t.Fatal(err)
				}
				task, err = service.ResolveBlockedSourcingTask(ctx, "sourcing", ResolveBlockedSourcingRequest{URL: url})
			}
			if err == nil || task == nil || task.SourcingCompletedAt != nil {
				t.Fatalf("failed submission recorded a completion: %+v %v", task, err)
			}
			stored, err := store.Find(ctx, task.ID)
			if err != nil || stored.SourcingCompletedAt != nil {
				t.Fatalf("failed completion persisted: %+v %v", stored, err)
			}
			clock = clock.Add(time.Minute)
			qbt.err = nil
			task, err = service.RetryTask(ctx, task.ID, nil)
			if err != nil || task.SourcingCompletedAt == nil || !task.SourcingCompletedAt.Equal(clock) {
				t.Fatalf("successful retry missing completion: %+v %v", task, err)
			}
			completedAt := clock
			stored, _ = store.Find(ctx, task.ID)
			if stored.SourcingCompletedAt == nil || !stored.SourcingCompletedAt.Equal(completedAt) {
				t.Fatal("completion was not persisted")
			}
			copy := cloneTask(stored)
			*copy.SourcingCompletedAt = clock.Add(time.Hour)
			if !stored.SourcingCompletedAt.Equal(completedAt) {
				t.Fatal("completion pointer shared across clones")
			}
			// Resubmitting a previously accepted download must retain its original sourcing time.
			clock = clock.Add(time.Hour)
			blockTask(task, TaskStageErrorTorrentSubmit, "retry download", clock)
			if err := store.Update(ctx, task); err != nil {
				t.Fatal(err)
			}
			task, err = service.RetryTask(ctx, task.ID, nil)
			if err != nil || !task.SourcingCompletedAt.Equal(completedAt) {
				t.Fatalf("retry changed sourcing time: %+v %v", task, err)
			}
			setTaskStage(task, TaskStageSourcing, TaskStageStatusRunning)
			if task.SourcingCompletedAt != nil {
				t.Fatal("restarting sourcing retained an obsolete time")
			}
		})
	}
}

func TestSQLiteV10SourcingTimeUpgradePreservesHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC)
	if err := store.Create(ctx, &Task{ID: "historical", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, IngestCompletedAt: &now, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO task_events (task_id,event_type,message,created_at) VALUES ('historical','CREATED','original history','2026-10-18T03:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`ALTER TABLE tasks DROP COLUMN sourcing_completed_at`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE task_store_meta SET value='10' WHERE key='schema_version'`); err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	task, err := store.Find(ctx, "historical")
	if err != nil || task.SourcingCompletedAt != nil || task.IngestCompletedAt == nil || !task.IngestCompletedAt.Equal(now) {
		t.Fatalf("upgrade lost or fabricated timestamps: %+v %v", task, err)
	}
	var count int
	if err := store.db.Get(&count, `SELECT COUNT(*) FROM task_events WHERE message='original history'`); err != nil || count != 1 {
		t.Fatalf("history lost: %d %v", count, err)
	}
	task.SourcingCompletedAt = &now
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	task, err = store.Find(ctx, task.ID)
	if err != nil || task.SourcingCompletedAt == nil || !task.SourcingCompletedAt.Equal(now) {
		t.Fatalf("completion time not persisted: %+v %v", task, err)
	}
}
