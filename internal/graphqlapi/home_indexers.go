package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type HomeIndexerService interface {
	Summary(context.Context, int) (indexeradmin.Summary, error)
}

func (r *Resolver) homeIndexerSummary(ctx context.Context, hours int) (*model.HomeIndexerSummary, error) {
	if r.HomeIndexers == nil {
		return nil, indexer.ErrNotConfigured
	}
	snapshot, err := r.HomeIndexers.Summary(ctx, hours)
	if err != nil {
		return nil, err
	}
	result := &model.HomeIndexerSummary{Revision: snapshot.Revision, Total: snapshot.Total, Hours: snapshot.Hours, StatisticsErrorCode: snapshot.StatisticsErrorCode}
	if stats := snapshot.Statistics; stats != nil {
		result.Statistics = &model.IndexerExecutionStatistics{WindowStartedAt: indexerTime(stats.WindowStartedAt), WindowEndedAt: indexerTime(stats.WindowEndedAt), Searches: stats.Searches, Records: stats.Records}
		if stats.Searches > 0 {
			rate := 100 * float64(stats.Succeeded) / float64(stats.Searches)
			result.Statistics.SuccessRate = &rate
			if stats.TimedSearches > 0 && stats.TotalElapsedMs > 0 {
				elapsed := float64(stats.TotalElapsedMs) / float64(stats.TimedSearches)
				result.Statistics.AverageElapsedMs = &elapsed
			}
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
