package tracker

import (
	"github.com/leothevan2444/moji/pkg/jackett"
)

type JackettService struct {
	client *jackett.Client
}

func NewJackettService(url string, apiKey string) *JackettService {
	return &JackettService{
		client: jackett.NewClient(url, apiKey),
	}
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

	results, err := s.client.Search(req)
	if err != nil {
		return nil, err
	}

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}
