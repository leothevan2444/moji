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
