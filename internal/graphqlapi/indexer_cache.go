package graphqlapi

import (
	"context"
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type IndexerCacheService interface {
	CacheStatus(context.Context) (indexeradmin.CacheSnapshot, error)
	ClearSearchCache(context.Context, string) (indexeradmin.CacheSnapshot, error)
	UpdateSearchCache(context.Context, string, config.IndexerCachePatch) (indexeradmin.CacheSnapshot, error)
}

func (r *Resolver) updateIndexerCache(ctx context.Context, expected string, input model.IndexerSearchCachePatchInput) (*model.IndexerCacheStatus, error) {
	if r.IndexerCache == nil {
		return nil, indexer.ErrNotConfigured
	}
	status, err := r.IndexerCache.UpdateSearchCache(ctx, expected, config.IndexerCachePatch{Enabled: input.Enabled, TTLSeconds: input.TTLSeconds, EmptyTTLSeconds: input.EmptyTTLSeconds, MaxSizeMB: input.MaxSizeMb})
	if err != nil {
		return nil, err
	}
	return indexerCacheModel(status), nil
}

func indexerCacheModel(snapshot indexeradmin.CacheSnapshot) *model.IndexerCacheStatus {
	return &model.IndexerCacheStatus{Revision: snapshot.Revision, Enabled: snapshot.Enabled, TTLSeconds: snapshot.TTLSeconds, EmptyTTLSeconds: snapshot.EmptyTTLSeconds, MaxSizeMb: snapshot.MaxSizeMB, Entries: snapshot.Stats.Entries, UsedBytes: snapshot.Stats.UsedBytes, ActiveRequests: snapshot.Stats.ActiveRequests, ActiveWaiters: snapshot.Stats.ActiveWaiters}
}
func (r *Resolver) indexerCacheStatus(ctx context.Context) (*model.IndexerCacheStatus, error) {
	if r.IndexerCache == nil {
		return nil, indexer.ErrNotConfigured
	}
	status, err := r.IndexerCache.CacheStatus(ctx)
	if err != nil {
		return nil, err
	}
	return indexerCacheModel(status), nil
}
func (r *Resolver) clearIndexerCache(ctx context.Context, id *string) (*model.IndexerCacheStatus, error) {
	if r.IndexerCache == nil {
		return nil, indexer.ErrNotConfigured
	}
	instance := ""
	if id != nil {
		if strings.TrimSpace(*id) == "" {
			return nil, indexer.ErrInvalidScope
		}
		instance = *id
	}
	status, err := r.IndexerCache.ClearSearchCache(ctx, instance)
	if err != nil {
		return nil, err
	}
	return indexerCacheModel(status), nil
}
