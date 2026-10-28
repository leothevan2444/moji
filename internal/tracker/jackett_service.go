package tracker

import (
	"context"
	"errors"
	"github.com/leothevan2444/moji/internal/logging"
	"net/url"
	"strings"
	"sync"

	"github.com/leothevan2444/moji/pkg/jackett"
)

// JackettConfig captures the runtime Jackett connection fields. Mirroring
// the config.Config.Connection.Jackett compatibility view as a typed struct lets callers hand a
// provider to NewJackettService instead of capturing values at startup.
type JackettConfig struct {
	QueryOrder []string
	URL        string
	APIKey     string
	Password   string
}

// JackettConfigProvider supplies the latest JackettConfig at the moment of
// each operation. Reading the config lazily means Web UI edits to
// indexers.jackett.url / api_key / password take effect on the next search without
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

// ListIndexers surfaces the configured Jackett indexers for the GraphQL
// resolver. Returns an empty slice when Jackett is not configured so the
// frontend can render an "未连接" hint instead of an error.
func (s *JackettService) ListIndexers() ([]jackett.Indexer, error) {
	client := s.currentClient()
	if client == nil {
		return []jackett.Indexer{}, nil
	}
	return client.GetIndexers()
}

func (s *JackettService) Search(query string, options ...SearchOption) ([]jackett.SearchResult, error) {
	opts := &SearchOptions{}
	for _, option := range options {
		option(opts)
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	var results []jackett.SearchResult
	err := s.SearchSequential(ctx, query, func(batch []jackett.SearchResult) (bool, error) {
		results = append(results, batch...)
		return false, nil
	}, options...)
	return results, err
}

func (s *JackettService) SearchSequential(ctx context.Context, query string, accept func([]jackett.SearchResult) (bool, error), options ...SearchOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	opts := &SearchOptions{}
	for _, option := range options {
		option(opts)
	}
	client := s.currentClient()
	if client == nil {
		return nil
	}
	sites, err := client.GetIndexersContext(ctx)
	if err != nil {
		return err
	}
	eligible := map[string]bool{}
	for _, site := range sites {
		if site.Configured {
			eligible[site.ID] = true
		}
	}
	priority := s.configProvider().QueryOrder
	requested := map[string]bool{}
	for _, id := range opts.Trackers {
		requested[strings.ToLower(id)] = true
	}
	ordered := []string{}
	seen := map[string]bool{}
	add := func(id string) {
		if !eligible[id] || seen[id] || len(requested) > 0 && !requested[strings.ToLower(id)] {
			return
		}
		ordered = append(ordered, id)
		seen[id] = true
	}
	for _, id := range priority {
		add(id)
	}
	for _, site := range sites {
		add(site.ID)
	}
	var failures []error
	successful := false
	for _, id := range ordered {
		if err := ctx.Err(); err != nil {
			return err
		}
		batch, err := client.SearchContext(ctx, jackett.SearchRequest{Query: query, Trackers: []string{id}, Categories: opts.Categories})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var requestError *url.Error
			if errors.As(err, &requestError) {
				err = requestError.Err
			}
			logging.Infof("tracker: indexer %s search failed (%v); continuing to next site", id, err)
			failures = append(failures, errors.New("indexer "+id+" search failed"))
			continue
		}
		successful = true
		if opts.Limit > 0 && len(batch) > opts.Limit {
			batch = batch[:opts.Limit]
		}
		accepted, err := accept(batch)
		if err != nil {
			return err
		}
		if accepted {
			return nil
		}
	}
	if !successful && len(failures) > 0 {
		return errors.Join(failures...)
	}
	return ctx.Err()
}
