package graphqlapi

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
)

type IndexerCheckService interface {
	Check(context.Context, indexercheck.Input) (indexercheck.Report, error)
}

func (r *Resolver) testIndexer(ctx context.Context, input model.TestIndexerInput) (*model.IndexerCheckReport, error) {
	if r.IndexerChecks == nil {
		return nil, indexer.ErrNotConfigured
	}
	req := indexercheck.Input{ConnectionOnly: input.ConnectionOnly, ExpectedRevision: input.ExpectedRevision, Search: input.Search, Download: input.Download}
	if input.ClientID != nil {
		req.ClientID = *input.ClientID
	}
	if input.RequestID != nil {
		req.RequestID = *input.RequestID
	}
	if input.InstanceID != nil {
		req.InstanceID = *input.InstanceID
	}
	if input.Keywords != nil {
		req.Keywords = *input.Keywords
	}
	if input.Draft != nil {
		patch, err := builtinPatchInput(*input.Draft)
		if err != nil {
			return nil, err
		}
		req.Draft = &patch
	}
	if input.DefinitionID != nil {
		if req.InstanceID != "" || len(*input.DefinitionID) > 128 || *input.DefinitionID == "" {
			return nil, indexer.ErrInvalidSettings
		}
		if req.Draft == nil {
			req.Draft = &config.BuiltinInstancePatch{}
		}
		req.Draft.Definition = input.DefinitionID
	}
	report, err := r.IndexerChecks.Check(ctx, req)
	if err != nil {
		return nil, err
	}
	result := &model.IndexerCheckReport{ID: report.ID, InstanceID: report.InstanceID, ConfigRevision: report.ConfigRevision, RuleRevision: report.RuleRevision, StartedAt: report.StartedAt.UTC().Format(time.RFC3339Nano), FinishedAt: report.FinishedAt.UTC().Format(time.RFC3339Nano), Results: report.Results, Partial: report.Partial, Truncated: report.Truncated, Stages: []*model.IndexerCheckStage{}}
	if report.Kind != "" {
		kind := model.IndexerSourceKind(report.Kind)
		result.Kind = &kind
	}
	for _, stage := range report.Stages {
		result.Stages = append(result.Stages, &model.IndexerCheckStage{Name: model.IndexerCheckStageName(stage.Name), State: model.IndexerCheckStageState(stage.State), ErrorCode: stage.ErrorCode, ElapsedMs: stage.Elapsed.Milliseconds()})
	}
	return result, nil
}

type IndexerCheckActivityService interface {
	SubscribeActivity(string) (<-chan []indexercheck.Activity, func(), error)
}

func (r *Resolver) indexerCheckActivity(ctx context.Context, clientID string) (<-chan []*model.IndexerCheckActivity, error) {
	source, ok := r.IndexerChecks.(IndexerCheckActivityService)
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	events, stop, err := source.SubscribeActivity(clientID)
	if err != nil {
		return nil, err
	}
	out := make(chan []*model.IndexerCheckActivity, 1)
	go func() {
		defer close(out)
		defer stop()
		for {
			select {
			case <-ctx.Done():
				return
			case snapshot, ok := <-events:
				if !ok {
					return
				}
				items := make([]*model.IndexerCheckActivity, 0, len(snapshot))
				for _, item := range snapshot {
					items = append(items, &model.IndexerCheckActivity{RequestID: item.RequestID, InstanceID: item.InstanceID, State: model.IndexerCheckActivityState(item.State)})
				}
				select {
				case <-out:
				default:
				}
				out <- items
			}
		}
	}()
	return out, nil
}
