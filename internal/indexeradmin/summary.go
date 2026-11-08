package indexeradmin

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
)

// Summary contains local statistics only. Reading Home never refreshes a remote
// directory or starts a site request; site errors belong to management/history.
type Summary struct {
	Revision            string
	Total               int
	Hours               int
	Statistics          *indexer.ExecutionStatistics
	StatisticsErrorCode string
}

func (s *Service) Summary(ctx context.Context, hours int) (Summary, error) {
	if hours != 24 && hours != 72 && hours != 336 {
		return Summary{}, indexer.ErrInvalidQuery
	}
	page, err := s.Instances(ctx)
	if err != nil {
		return Summary{}, err
	}
	result := summarize(page)
	result.Hours = hours
	if s.statistics != nil {
		stats, err := s.statistics.Statistics(ctx, time.Now().UTC(), hours)
		if err != nil {
			if ctx.Err() != nil {
				return Summary{}, ctx.Err()
			}
			result.StatisticsErrorCode = indexer.ErrStateUnavailable.Error()
		} else {
			result.Statistics = &stats
		}
	}
	return result, nil
}
func summarize(page InstancePage) Summary {
	count := 0
	for _, site := range page.Items {
		if site.Enabled {
			count++
		}
	}
	return Summary{Revision: page.Revision, Total: count}
}
