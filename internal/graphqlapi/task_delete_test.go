package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

type deletionOptionsRuntime struct {
	fakeTaskRuntime
	calls   int
	options taskruntime.DeleteTaskOptions
}

func (r *deletionOptionsRuntime) DeleteTaskWithOptions(ctx context.Context, id string, options taskruntime.DeleteTaskOptions, _ taskruntime.StashSceneDeleter) (*taskruntime.Task, error) {
	r.calls++
	r.options = options
	return r.DeleteTask(ctx, id)
}
func (r *deletionOptionsRuntime) DeleteTasksWithOptions(_ context.Context, _ []string, options taskruntime.DeleteTaskOptions, _ taskruntime.StashSceneDeleter) (taskruntime.TaskBatchPayload, error) {
	r.calls++
	r.options = options
	return taskruntime.TaskBatchPayload{BatchID: "test-batch"}, nil
}
func TestTaskDeletionGraphQLRequiresExplicitStashOptIn(t *testing.T) {
	for _, query := range []string{`mutation{deleteTask(id:"task"){id}}`, `mutation{deleteTask(id:"task",deleteStashScene:false){id}}`, `mutation{deleteTask(id:"task",deleteStashScene:true){id}}`, `mutation{deleteTasks(ids:["task"],deleteStashScene:true){batchId}}`} {
		t.Run(query, func(t *testing.T) {
			runtime := &deletionOptionsRuntime{fakeTaskRuntime: fakeTaskRuntime{deleteTask: &taskruntime.Task{ID: "task", Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone, CreatedAt: time.Now(), UpdatedAt: time.Now()}}}
			resp := executeGraphQL(t, NewResolver(nil, runtime, nil, "test"), query)
			if len(resp.Errors) > 0 {
				t.Fatal(resp.Errors)
			}
			want := query == `mutation{deleteTask(id:"task",deleteStashScene:true){id}}` || query == `mutation{deleteTasks(ids:["task"],deleteStashScene:true){batchId}}`
			if (runtime.calls > 0) != want || runtime.options.DeleteStashScene != want {
				t.Fatal("option did not follow explicit request")
			}
		})
	}
}
