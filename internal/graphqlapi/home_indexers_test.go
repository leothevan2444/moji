package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type homeIndexerSummarySource struct{ snapshot indexeradmin.Summary }

func (s *homeIndexerSummarySource) Summary(_ context.Context, hours int) (indexeradmin.Summary, error) {
	snapshot := s.snapshot
	snapshot.Hours = hours
	return snapshot, nil
}
func TestHomeIndexerStatisticsPreserveUnknownZeroAndMeasuredWindow(t *testing.T) {
	source := &homeIndexerSummarySource{snapshot: indexeradmin.Summary{Revision: "revision"}}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.HomeIndexers = source
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Summary model.HomeIndexerSummary `json:"homeIndexerSummary"`
		}
	}
	query := `{homeIndexerSummary{revision total hours statisticsErrorCode statistics{windowStartedAt windowEndedAt searches successRate averageElapsedMs records}}}`
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	if len(response.Errors) > 0 || response.Data.Summary.Statistics != nil {
		t.Fatal("absent statistics were fabricated", response.Errors)
	}
	at := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	source.snapshot.Total = 0
	source.snapshot.Statistics = &indexer.ExecutionStatistics{WindowStartedAt: at.Add(-24 * time.Hour), WindowEndedAt: at, Records: 7}
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	stats := response.Data.Summary.Statistics
	if len(response.Errors) > 0 || response.Data.Summary.Total != 0 || stats == nil || stats.Searches != 0 || stats.SuccessRate != nil || stats.AverageElapsedMs != nil || stats.Records != 7 {
		t.Fatal("empty window fabricated a rate", response.Errors)
	}
	source.snapshot.Statistics.Searches = 4
	source.snapshot.Statistics.TimedSearches = 4
	source.snapshot.Statistics.Succeeded = 3
	source.snapshot.Statistics.TotalElapsedMs = 8000
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	stats = response.Data.Summary.Statistics
	if len(response.Errors) > 0 || stats.SuccessRate == nil || *stats.SuccessRate != 75 || stats.AverageElapsedMs == nil || *stats.AverageElapsedMs != 2000 || stats.WindowEndedAt != "2026-11-08T00:00:00Z" {
		t.Fatal("statistics denominator or unit wrong", response.Errors)
	}
	source.snapshot.Statistics.TimedSearches = 3
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	stats = response.Data.Summary.Statistics
	if stats.SuccessRate == nil || *stats.SuccessRate != 75 || stats.AverageElapsedMs == nil || *stats.AverageElapsedMs != float64(8000)/3 {
		t.Fatal("mean must use only records with known timing")
	}

	for _, counts := range [][2]int64{{0, 0}, {3, 0}, {0, 8000}} {
		source.snapshot.Statistics.TimedSearches = int(counts[0])
		source.snapshot.Statistics.TotalElapsedMs = counts[1]
		executeDefinitionGraphQLInto(t, resolver, query, &response)
		if response.Data.Summary.Statistics.AverageElapsedMs != nil {
			t.Fatal("zero or absent timing must not produce an average", counts)
		}
	}

}
