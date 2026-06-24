package tracker

import (
	"sync"

	"github.com/leothevan2444/moji/pkg/jackett"
)

// JackettConfig captures the runtime Jackett connection fields. Mirroring
// the config.Config.Jackett block as a typed struct lets callers hand a
// provider to NewJackettService instead of capturing values at startup.
type JackettConfig struct {
	URL      string
	APIKey   string
	Password string
}

// JackettConfigProvider supplies the latest JackettConfig at the moment of
// each operation. Reading the config lazily means Web UI edits to
// jackett.url / api_key / password take effect on the next search without
// restarting Moji.
type JackettConfigProvider func() JackettConfig

type JackettService struct {
	configProvider JackettConfigProvider

	// clientMu guards client + lastKey. Reads and writes follow the
	// double-checked-locking pattern so a hot path (Search) avoids the lock
	// when nothing has changed.
	clientMu sync.RWMutex
	client   *jackett.Client
	lastKey  jackettClientKey
}

// jackettClientKey is the identity of a Jackett client. A new client is built
// only when the key changes, so password rotation doesn't drop the in-flight
// HTTP session unnecessarily.
type jackettClientKey struct {
	URL      string
	APIKey   string
	Password string
}

func NewJackettService(configProvider JackettConfigProvider) *JackettService {
	return &JackettService{
		configProvider: configProvider,
	}
}

// currentClient returns a Jackett client matching the latest config. It
// reuses the cached client when the identity has not changed; otherwise it
// builds a new one. Safe for concurrent use.
func (s *JackettService) currentClient() *jackett.Client {
	if s.configProvider == nil {
		return nil
	}
	cfg := s.configProvider()
	key := jackettClientKey{URL: cfg.URL, APIKey: cfg.APIKey, Password: cfg.Password}

	s.clientMu.RLock()
	if s.client != nil && s.lastKey == key {
		client := s.client
		s.clientMu.RUnlock()
		return client
	}
	s.clientMu.RUnlock()

	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	// Re-check after acquiring the write lock — another goroutine may have
	// already produced a matching client while we were waiting.
	if s.client != nil && s.lastKey == key {
		return s.client
	}
	s.client = jackett.NewClient(cfg.URL, cfg.APIKey, cfg.Password)
	s.lastKey = key
	return s.client
}

// Client exposes the underlying Jackett client so the stats collector can
// poll GetIndexers without going through the search path. The returned
// client reflects the most recent config; callers should treat it as
// transient and not retain pointers across config changes.
func (s *JackettService) Client() *jackett.Client {
	return s.currentClient()
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

	client := s.currentClient()
	if client == nil {
		return nil, nil
	}

	// Use SearchWithIndexerStatus so the global indexer-status hook (if any)
	// can observe per-indexer latency and errors for the stats collector.
	results, _, err := client.SearchWithIndexerStatus(req)
	if err != nil {
		return nil, err
	}

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}