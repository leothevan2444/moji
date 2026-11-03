package indexeradmin

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type CacheSnapshot struct {
	Revision                               string
	Enabled                                bool
	TTLSeconds, EmptyTTLSeconds, MaxSizeMB int
	Stats                                  indexer.SearchCacheStats
}

func (s *Service) UpdateSearchCache(ctx context.Context, expected string, patch config.IndexerCachePatch) (CacheSnapshot, error) {
	if s.store == nil || s.cache == nil {
		return CacheSnapshot{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.UpdateIndexerSearchCache(ctx, expected, patch)
	if err != nil {
		return CacheSnapshot{}, err
	}
	policy := cfg.Indexers.SearchCache
	stats, err := s.cache.Status(CachePolicy(policy))
	if err != nil {
		return CacheSnapshot{}, err
	}
	return CacheSnapshot{Revision: revision, Enabled: policy.Enabled, TTLSeconds: policy.TTLSeconds, EmptyTTLSeconds: policy.EmptyTTLSeconds, MaxSizeMB: policy.MaxSizeMB, Stats: stats}, nil
}

func CachePolicy(cfg config.IndexerSearchCacheConfig) indexer.SearchCachePolicy {
	return indexer.SearchCachePolicy{Enabled: cfg.Enabled, TTL: time.Duration(cfg.TTLSeconds) * time.Second, EmptyTTL: time.Duration(cfg.EmptyTTLSeconds) * time.Second, MaxBytes: int64(cfg.MaxSizeMB) << 20}
}
func (s *Service) SetSearchCache(cache *indexer.SearchCache, events indexer.EventPublisher) {
	s.cache = cache
	s.events = events
}
func (s *Service) CacheStatus(ctx context.Context) (CacheSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return CacheSnapshot{}, err
	}
	if s.store == nil || s.cache == nil {
		return CacheSnapshot{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	policy := cfg.Indexers.SearchCache
	stats, err := s.cache.Status(CachePolicy(policy))
	if err != nil {
		return CacheSnapshot{}, err
	}
	return CacheSnapshot{Revision: revision, Enabled: policy.Enabled, TTLSeconds: policy.TTLSeconds, EmptyTTLSeconds: policy.EmptyTTLSeconds, MaxSizeMB: policy.MaxSizeMB, Stats: stats}, nil
}
func (s *Service) ClearSearchCache(ctx context.Context, instanceID string) (CacheSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return CacheSnapshot{}, err
	}
	if s.cache == nil || s.store == nil {
		return CacheSnapshot{}, indexer.ErrNotConfigured
	}
	if len(instanceID) > 256 {
		return CacheSnapshot{}, indexer.ErrInvalidScope
	}
	if instanceID == "" {
		s.cache.Clear()
	} else {
		s.cache.InvalidateInstance(instanceID)
	}
	if s.events != nil {
		s.events.Publish(indexer.EventCache)
	}
	return s.CacheStatus(ctx)
}
