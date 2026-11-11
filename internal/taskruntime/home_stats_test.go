package taskruntime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestHomeStatsBoundariesAndLifecycle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	relational, _ := relationalStore(t)
	sqlite, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlite.db.Close() })
	for name, store := range map[string]TaskStore{"relational": relational, "sqlite": sqlite, "memory": NewMemoryTaskStore()} {
		t.Run(name, func(t *testing.T) {
			bus := NewTaskEventBus(8)
			defer bus.Close()
			eventing, err := NewEventingTaskStore(store, bus)
			if err != nil {
				t.Fatal(err)
			}
			service := &Service{store: eventing}
			if stats, err := service.HomeStats(ctx, now); err != nil || stats != (HomeStats{}) {
				t.Fatal(stats, err)
			}
			add := func(id string, stage TaskStage, status TaskStageStatus, completed, next *time.Time) {
				t.Helper()
				if err := store.Create(ctx, &Task{ID: id, Code: id, Source: TaskSourceManual, Stage: stage, StageStatus: status, IngestCompletedAt: completed, SourcingNextCheckAt: next, CreatedAt: now, UpdatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			for _, tc := range []struct {
				id    string
				delta time.Duration
			}{
				{"lower", -24 * time.Hour}, {"upper", 0}, {"old", -24*time.Hour - time.Second}, {"future", time.Second},
			} {
				at := now.Add(tc.delta).In(time.FixedZone("offset", 8*60*60))
				add(tc.id, TaskStageCompleted, TaskStageStatusDone, &at, nil)
			}
			add("missing-time", TaskStageCompleted, TaskStageStatusDone, nil, nil)
			past := now.Add(-time.Hour)
			add("overdue-wait", TaskStageSourcing, TaskStageStatusPending, nil, &past)
			add("blocked-source", TaskStageSourcing, TaskStageStatusBlocked, nil, &past)
			add("queued", TaskStageSourcing, TaskStageStatusPending, nil, nil)
			want := HomeStats{Total: 8, Attention: 1, Waiting: 1, Active: 1, CompletedLast24Hours: 2}
			check := func() {
				t.Helper()
				got, err := service.HomeStats(ctx, now)
				if err != nil || got != want {
					t.Fatalf("got %+v, want %+v: %v", got, want, err)
				}
				overview, err := service.HomeOverview(ctx, now)
				if err != nil || overview.HomeStats != got {
					t.Fatalf("home differs: %+v %v", overview, err)
				}
			}
			check()
			if _, err := store.Delete(ctx, "lower"); err != nil {
				t.Fatal(err)
			}
			want.Total--
			want.CompletedLast24Hours--
			check()
			task, err := store.Find(ctx, "upper")
			if err != nil {
				t.Fatal(err)
			}
			task.Stage, task.StageStatus = TaskStageSourcing, TaskStageStatusPending
			if err := store.Update(ctx, task); err != nil {
				t.Fatal(err)
			}
			want.Active++
			want.CompletedLast24Hours--
			check()
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := service.HomeStats(canceled, now); err == nil {
				t.Fatal("ignored cancellation")
			}
		})
	}
}

func TestHomeStatsDoesNotDecodeTaskPayloads(t *testing.T) {
	ctx := context.Background()
	store, manager := relationalStore(t)
	now := time.Now()
	if err := store.Create(ctx, &Task{ID: "broken-payload", Code: "TEST-1", Source: TaskSourceManual, Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE task_checkpoints SET payload_version = 999 WHERE task_id = 'broken-payload'`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.HomeOverview(ctx, now); err == nil {
		t.Fatal("fixture must break sampled task decoding")
	}
	stats, err := (&Service{store: store}).HomeStats(ctx, now)
	if err != nil || stats.Active != 1 {
		t.Fatalf("counts decoded task payload: %+v %v", stats, err)
	}
}

// Exposes only the general store contract, with an observable forbidden fallback.
type unsupportedHomeStatsStore struct{ TaskStore }

func (unsupportedHomeStatsStore) List(context.Context) ([]*Task, error) {
	panic("statistics must not fall back to loading all tasks")
}

func TestHomeStatsRejectsUnsupportedStore(t *testing.T) {
	store := unsupportedHomeStatsStore{TaskStore: NewMemoryTaskStore()}
	if _, err := (&Service{store: store}).HomeStats(context.Background(), time.Now()); err == nil {
		t.Fatal("unsupported statistics were reported as zero")
	}
	bus := NewTaskEventBus(8)
	defer bus.Close()
	eventing, err := NewEventingTaskStore(store, bus)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eventing.HomeStats(context.Background(), time.Now()); err == nil {
		t.Fatal("eventing store hid unsupported statistics")
	}
}
