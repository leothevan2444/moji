package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type IndexerDefinitionPolicyService interface {
	DefinitionPolicy(context.Context) (indexeradmin.DefinitionPolicy, error)
	UpdateDefinitionPolicy(context.Context, string, config.IndexerDefinitionPolicyPatch) (indexeradmin.DefinitionPolicy, error)
}

func definitionPolicyModel(value indexeradmin.DefinitionPolicy, err error) (*model.IndexerDefinitionPolicy, error) {
	if err != nil {
		return nil, err
	}
	return &model.IndexerDefinitionPolicy{Revision: value.Revision, Source: value.Source, SchemaVersion: value.SchemaVersion, CheckIntervalHours: value.CheckIntervalHours, AutoUpdate: value.AutoUpdate}, nil
}
func (r *Resolver) indexerDefinitionPolicy(ctx context.Context) (*model.IndexerDefinitionPolicy, error) {
	if r.IndexerDefinitionPolicy == nil {
		return nil, indexer.ErrNotConfigured
	}
	return definitionPolicyModel(r.IndexerDefinitionPolicy.DefinitionPolicy(ctx))
}
func (r *Resolver) updateIndexerDefinitionPolicy(ctx context.Context, expected string, input model.IndexerDefinitionPolicyPatchInput) (*model.IndexerDefinitionPolicy, error) {
	if r.IndexerDefinitionPolicy == nil {
		return nil, indexer.ErrNotConfigured
	}
	return definitionPolicyModel(r.IndexerDefinitionPolicy.UpdateDefinitionPolicy(ctx, expected, config.IndexerDefinitionPolicyPatch{CheckIntervalHours: input.CheckIntervalHours, AutoUpdate: input.AutoUpdate}))
}
