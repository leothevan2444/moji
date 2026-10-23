package graphqlapi

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestHomeOverviewGraphQLContract(t *testing.T) {
	store, err := taskruntime.NewSQLiteTaskStore(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := &homeGraphQLRuntime{store: store}
	now := time.Now().UTC()
	tasks := []*taskruntime.Task{
		{ID: "blocked", Source: taskruntime.TaskSourceManual, Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusBlocked, CreatedAt: now, UpdatedAt: now},
		{ID: "ingesting", Source: taskruntime.TaskSourceManual, Stage: taskruntime.TaskStagePendingIngest, StageStatus: taskruntime.TaskStageStatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "completed", Source: taskruntime.TaskSourceManual, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone, IngestCompletedAt: &now, CreatedAt: now, UpdatedAt: now},
	}
	for _, task := range tasks {
		if err := store.Create(context.Background(), task); err != nil {
			t.Fatal(err)
		}
	}
	resolver := NewResolver(nil, nil, runtime, nil, "test")
	var result struct {
		Data struct {
			Overview struct {
				Attention, Active, Waiting, CompletedLast24Hours int
				AttentionTasks, ActiveTasks, RecentTasks         []struct{ ID string }
			} `json:"homeOverview"`
			Subscription struct {
				Total                 int
				AttentionPerformerIDs []string
			} `json:"homeSubscriptionSummary"`
		}
		Errors []any
	}
	executeGraphQLInto(t, resolver, `{ homeOverview { attention active waiting completedLast24Hours attentionTasks { id files { index } } activeTasks { id } recentTasks { id } } homeSubscriptionSummary { total attentionPerformerIds } }`, &result)
	if len(result.Errors) != 0 {
		t.Fatalf("GraphQL errors: %+v", result.Errors)
	}
	got := result.Data.Overview
	if got.Attention != 1 || got.Active != 1 || got.Waiting != 0 || got.CompletedLast24Hours != 1 || got.ActiveTasks[0].ID != "ingesting" || got.RecentTasks[0].ID != "completed" {
		t.Fatalf("incorrect home contract: %+v", got)
	}
	if result.Data.Subscription.Total != 0 || result.Data.Subscription.AttentionPerformerIDs == nil {
		t.Fatal("missing subscription service must return an empty summary")
	}
}

type homeGraphQLRuntime struct {
	fakeTaskRuntime
	store *taskruntime.SQLiteTaskStore
}

func (r *homeGraphQLRuntime) HomeOverview(ctx context.Context, now time.Time) (taskruntime.HomeOverview, error) {
	return r.store.HomeOverview(ctx, now)
}
