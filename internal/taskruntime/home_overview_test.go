package taskruntime

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestHomeOverviewSeparatesAttentionAndAutomaticWork(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	sqlite, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.db.Close()
	memory := NewMemoryTaskStore()
	var fixture []*Task
	add := func(id string, stage TaskStage, status TaskStageStatus, complete, next *time.Time) {
		task := &Task{ID: id, Source: TaskSourceManual, Code: id, Stage: stage, StageStatus: status, CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now, IngestCompletedAt: complete, SourcingNextCheckAt: next}
		fixture = append(fixture, task)
	}
	for i := 0; i < 9; i++ {
		add(fmt.Sprintf("blocked-%02d", i), TaskStageDownloading, TaskStageStatusBlocked, nil, nil)
	}
	for i := 0; i < 12; i++ {
		add(fmt.Sprintf("downloading-%02d", i), TaskStageDownloading, TaskStageStatusRunning, nil, nil)
	}
	next := now.Add(time.Hour)
	add("waiting", TaskStageSourcing, TaskStageStatusPending, nil, &next)
	add("queued", TaskStageSourcing, TaskStageStatusPending, nil, nil)
	add("pending-ingest", TaskStagePendingIngest, TaskStageStatusPending, nil, nil)
	add("transferring", TaskStageTransferring, TaskStageStatusRunning, nil, nil)
	add("scanning", TaskStageScanning, TaskStageStatusPending, nil, nil)
	for i, delta := range []time.Duration{-25 * time.Hour, -24 * time.Hour, -time.Hour, 0, time.Hour} {
		complete := now.Add(delta)
		add(fmt.Sprintf("completed-%02d", i), TaskStageCompleted, TaskStageStatusDone, &complete, nil)
	}
	add("legacy-completed", TaskStageCompleted, TaskStageStatusDone, nil, nil)
	for _, task := range fixture {
		if err := sqlite.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		if err := memory.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	var previous HomeOverview
	for name, store := range map[string]TaskStore{"sqlite": sqlite, "memory": memory} {
		t.Run(name, func(t *testing.T) {
			service := &Service{store: store}
			got, err := service.HomeOverview(ctx, now)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total != 32 || got.Attention != 9 || got.Active != 16 || got.Waiting != 1 || got.CompletedLast24Hours != 3 {
				t.Fatalf("incorrect counts: %+v", got)
			}
			if len(got.AttentionTasks) != 3 || len(got.ActiveTasks) != 5 || len(got.RecentTasks) != 3 {
				t.Fatalf("lists must be bounded: %+v", got)
			}
			if got.AttentionTasks[0].ID != "blocked-00" {
				t.Fatal("attention order must be deterministic")
			}
			// Polling timestamps must not reorder recent completion milestones.
			if got.RecentTasks[0].ID != "completed-04" {
				t.Fatal("recent order must use ingest completion")
			}
			counts := HomeOverview{Total: got.Total, Attention: got.Attention, Active: got.Active, Waiting: got.Waiting, CompletedLast24Hours: got.CompletedLast24Hours}
			if previous.Total != 0 && !reflect.DeepEqual(previous, counts) {
				t.Fatalf("store summaries differ: %+v %+v", previous, counts)
			}
			previous = counts
		})
	}
	// Corrupt an unsampled row to ensure the SQL path does not decode all history.
	if _, err := sqlite.db.ExecContext(ctx, `UPDATE tasks SET file_plan_json = '{invalid' WHERE id = 'legacy-completed'`); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{store: sqlite}).HomeOverview(ctx, now); err != nil {
		t.Fatalf("must not decode unsampled history: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := sqlite.HomeOverview(canceled, now); err == nil {
		t.Fatal("must honor cancellation")
	}
}

func TestHomeOverviewEmptyStore(t *testing.T) {
	store, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	got, err := store.HomeOverview(context.Background(), time.Now())
	if err != nil || got.Total != 0 || len(got.AttentionTasks) != 0 {
		t.Fatalf("unexpected empty overview: %+v %v", got, err)
	}
}
