package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
	"testing"
)

type taskChoiceAPI struct {
	fakeTaskRuntime
	id    string
	req   taskruntime.ResolveIndexedResourceRequest
	calls int
}

func (s *taskChoiceAPI) ResolveIndexedResourceContext(_ context.Context, id string, req taskruntime.ResolveIndexedResourceRequest) (*taskruntime.Task, error) {
	s.calls++
	s.id = id
	s.req = req
	return &taskruntime.Task{ID: id, Code: "ABCD-123", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusPending, Revision: 8}, nil
}

type boundSearchAPI struct {
	searchAPISource
	bound string
}

func (s *boundSearchAPI) Get(ctx context.Context, id string, page torrentsearch.Page) (torrentsearch.Snapshot, error) {
	snapshot, err := s.searchAPISource.Get(ctx, id, page)
	snapshot.TaskID = s.bound
	return snapshot, err
}
func TestTaskChoiceGraphQLChecksScopeAndPassesRevisionToRuntimeWithoutCreatingTask(t *testing.T) {
	runtime := &taskChoiceAPI{}
	source := &boundSearchAPI{bound: "task-one"}
	resolver := NewResolver(nil, runtime, nil, "test")
	resolver.SearchRuns = source
	var response struct {
		Errors []struct {
			Message    string
			Extensions struct{ Code string }
		}
		Data struct {
			Task *model.Task `json:"resolveTaskSourcingCandidate"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{resolveTaskSourcingCandidate(input:{taskId:"task-other",searchId:"run",candidateId:"candidate",expectedRevision:7}){id}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != "TORRENT_SEARCH_TASK_SCOPE_INVALID" || runtime.calls != 0 {
		t.Fatal("task-scoped run escaped ownership", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{resolveTaskSourcingCandidate(input:{taskId:"task-one",searchId:"run",candidateId:"candidate",expectedRevision:7,paused:false}){id stage stageStatus revision}}`, &response)
	if len(response.Errors) != 0 || response.Data.Task == nil || response.Data.Task.StageStatus != model.TaskStageStatusPending || runtime.id != "task-one" || runtime.req.ResourceID != "server-private-resource" || runtime.req.ExpectedRevision != 7 || runtime.req.Paused == nil || *runtime.req.Paused {
		t.Fatal("existing-task API bypassed server resource/runtime version", response.Errors)
	}
}
