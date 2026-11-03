package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type HomeIndexerService interface {
	Summary(context.Context) (indexeradmin.Summary, error)
}

func (r *Resolver) homeIndexerSummary(ctx context.Context) (*model.HomeIndexerSummary, error) {
	if r.HomeIndexers == nil {
		return nil, indexer.ErrNotConfigured
	}
	snapshot, err := r.HomeIndexers.Summary(ctx)
	if err != nil {
		return nil, err
	}
	result := &model.HomeIndexerSummary{DefinitionJob: definitionJobToModel(snapshot.DefinitionJob), DefinitionErrorCode: snapshot.DefinitionErrorCode, BuiltinSearchSucceeded: snapshot.BuiltinSearchSucceeded, HealthErrorCode: snapshot.HealthErrorCode, Revision: snapshot.Revision, Total: snapshot.Total, BuiltinEnabled: snapshot.BuiltinEnabled, BuiltinUnverified: snapshot.BuiltinUnverified, Attention: snapshot.Attention, Issues: make([]*model.HomeIndexerIssue, 0, len(snapshot.Issues))}
	for _, issue := range snapshot.Issues {
		result.Issues = append(result.Issues, &model.HomeIndexerIssue{Operation: issue.Operation, Observation: indexerObservationModel(issue.Observation), ID: issue.ID, Name: issue.Name, ErrorCode: issue.ErrorCode})
	}
	if gateway := snapshot.Jackett; gateway != nil {
		result.Jackett = &model.HomeJackettSummary{Sites: gateway.Sites, Stale: gateway.Stale, ErrorCode: gateway.ErrorCode}
		if gateway.UpdatedAt != nil {
			value := indexerTime(*gateway.UpdatedAt)
			result.Jackett.UpdatedAt = &value
		}
	}
	return result, nil
}

func indexerObservationModel(observation *indexer.Observation) *model.IndexerObservation {
	if observation == nil {
		return nil
	}
	result := &model.IndexerObservation{StartedAt: indexerTime(observation.StartedAt), FinishedAt: indexerTime(observation.FinishedAt), ErrorCode: observation.ErrorCode, CorrelationID: observation.CorrelationID}
	if observation.RetryAt != nil {
		value := indexerTime(*observation.RetryAt)
		result.RetryAt = &value
	}
	return result
}
