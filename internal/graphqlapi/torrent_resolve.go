package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
)

func (r *Resolver) resolveTaskSourcingCandidate(ctx context.Context, input model.ResolveTaskSourcingCandidateInput) (*model.Task, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	runtime, ok := r.TaskRuntime.(interface {
		ResolveIndexedResourceContext(context.Context, string, taskruntime.ResolveIndexedResourceRequest) (*taskruntime.Task, error)
	})
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	snapshot, err := r.SearchRuns.Get(ctx, input.SearchID, torrentsearch.Page{Limit: 1})
	if err != nil {
		return nil, err
	}
	if snapshot.TaskID != "" && snapshot.TaskID != input.TaskID {
		return nil, torrentsearch.ErrTaskScope
	}
	resource, err := r.SearchRuns.ResourceID(ctx, input.SearchID, input.CandidateID)
	if err != nil {
		return nil, err
	}
	task, err := runtime.ResolveIndexedResourceContext(ctx, input.TaskID, taskruntime.ResolveIndexedResourceRequest{ResourceID: resource, ExpectedRevision: input.ExpectedRevision, Paused: input.Paused})
	if err != nil {
		return nil, err
	}
	return taskToModel(task), nil
}
