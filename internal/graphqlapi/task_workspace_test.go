package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type workspaceGraphQLRuntime struct {
	fakeTaskRuntime
	store *taskruntime.EventingTaskStore
}

func (r *workspaceGraphQLRuntime) TaskPage(ctx context.Context, in taskruntime.TaskPageInput) (taskruntime.TaskPage, error) {
	return r.store.TaskPage(ctx, in)
}
func (r *workspaceGraphQLRuntime) AttentionQueue(ctx context.Context) (taskruntime.TaskPage, error) {
	return r.store.AttentionQueue(ctx)
}
func (r *workspaceGraphQLRuntime) HomeTaskSnapshot(ctx context.Context) (taskruntime.HomeTaskSnapshot, error) {
	return r.store.HomeTaskSnapshot(ctx)
}
func (r *workspaceGraphQLRuntime) TaskDetailSnapshot(ctx context.Context, id string) (taskruntime.TaskDetailSnapshot, error) {
	return r.store.TaskDetailSnapshot(ctx, id)
}
func TestTaskWorkspaceGraphQLSnapshots(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	bus := taskruntime.NewTaskEventBus(8)
	defer bus.Close()
	store, err := taskruntime.NewEventingTaskStore(taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations), bus)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, id := range []string{"one", "two"} {
		if err := store.Create(ctx, &taskruntime.Task{ID: id, Source: taskruntime.TaskSourceManual, Code: id, Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusBlocked, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	resolver := NewResolver(nil, nil, &workspaceGraphQLRuntime{store: store}, nil, "test")
	var result struct {
		Data struct {
			Page struct {
				Items []struct {
					ID, Version string
					Revision    int64
				}
				Sequence, TotalCount, TotalPages int
				StreamID                         string
			} `json:"taskPage"`
			Queue struct {
				Items    []struct{ ID string }
				Sequence int
			} `json:"taskAttentionQueue"`
			Detail struct {
				Task     *struct{ ID string }
				Sequence int
			} `json:"taskDetailSnapshot"`
			Home struct {
				Total, Attention int
				AttentionTasks   []struct{ ID string }
			} `json:"homeTaskSnapshot"`
		}
		Errors []any
	}
	executeGraphQLInto(t, resolver, `{taskPage(input:{pageSize:1,page:2}){items{id version revision} totalCount totalPages sequence streamId} taskAttentionQueue{items{id} sequence} taskDetailSnapshot(id:"missing"){task{id} sequence} homeTaskSnapshot{total attention attentionTasks{id}}}`, &result)
	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %+v", result.Errors)
	}
	p := result.Data.Page
	if p.TotalCount != 2 || p.TotalPages != 2 || p.Sequence != 2 || p.StreamID == "" || len(p.Items) != 1 || p.Items[0].ID != "two" || p.Items[0].Revision != 1 || p.Items[0].Version == "" {
		t.Fatalf("page contract: %+v", p)
	}
	if len(result.Data.Queue.Items) != 2 || result.Data.Queue.Sequence != 2 || result.Data.Detail.Task != nil || result.Data.Detail.Sequence != 2 || result.Data.Home.Attention != 2 {
		t.Fatalf("independent snapshots: %+v", result.Data)
	}
}
