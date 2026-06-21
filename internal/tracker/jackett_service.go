package tracker

import (
	"github.com/leothevan2444/moji/pkg/jackett"
)

type JackettService struct {
	client *jackett.Client
}

func NewJackettService(url string, apiKey string, password string) *JackettService {
	return &JackettService{
		client: jackett.NewClient(url, apiKey, password),
	}
}

// Client exposes the underlying Jackett client so the stats collector can
// poll GetIndexers without going through the search path.
func (s *JackettService) Client() *jackett.Client {
	return s.client
}

func (s *JackettService) Search(query string, options ...SearchOption) ([]jackett.SearchResult, error) {
	opts := &SearchOptions{}
	for _, opt := range options {
		opt(opts)
	}

	req := jackett.SearchRequest{
		Query:      query,
		Categories: opts.Categories,
		Trackers:   opts.Trackers,
	}

	// Use SearchWithIndexerStatus so the global indexer-status hook (if any)
	// can observe per-indexer latency and errors for the stats collector.
	results, _, err := s.client.SearchWithIndexerStatus(req)
	if err != nil {
		return nil, err
	}

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}
