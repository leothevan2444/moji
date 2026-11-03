package graphqlapi

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type cacheAPISource struct {
	scope, revision string
	patch           config.IndexerCachePatch
}

func (s *cacheAPISource) UpdateSearchCache(ctx context.Context, revision string, patch config.IndexerCachePatch) (indexeradmin.CacheSnapshot, error) {
	s.revision = revision
	s.patch = patch
	return s.CacheStatus(ctx)
}

func (*cacheAPISource) CacheStatus(context.Context) (indexeradmin.CacheSnapshot, error) {
	return indexeradmin.CacheSnapshot{Revision: "revision", Enabled: true, TTLSeconds: 300, EmptyTTLSeconds: 30, MaxSizeMB: 32}, nil
}
func (s *cacheAPISource) ClearSearchCache(ctx context.Context, id string) (indexeradmin.CacheSnapshot, error) {
	s.scope = id
	return s.CacheStatus(ctx)
}

func TestCacheGraphQLExposesAggregateStatusAndPreservesClearScope(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, "test")
	source := &cacheAPISource{}
	resolver.IndexerCache = source
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Status  model.IndexerCacheStatus `json:"indexerCacheStatus"`
			Cleared model.IndexerCacheStatus `json:"clearIndexerSearchCache"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `{indexerCacheStatus{revision enabled ttlSeconds emptyTtlSeconds maxSizeMb entries usedBytes activeRequests activeWaiters}}`, &response)
	if len(response.Errors) > 0 || response.Data.Status.Revision != "revision" || response.Data.Status.EmptyTTLSeconds != 30 {
		t.Fatal("cache status contract", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{clearIndexerSearchCache(instanceId:"one"){entries usedBytes}}`, &response)
	if len(response.Errors) > 0 || source.scope != "one" {
		t.Fatal("cache clear scope lost", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{clearIndexerSearchCache(instanceId:""){entries}}`, &response)
	if len(response.Errors) != 1 || source.scope != "one" {
		t.Fatal("explicit empty scope silently became a global clear", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerSearchCache(expectedRevision:"reviewed",input:{enabled:false,maxSizeMb:64}){revision enabled maxSizeMb}}`, &response)
	if len(response.Errors) > 0 || source.revision != "reviewed" || source.patch.Enabled == nil || *source.patch.Enabled || source.patch.MaxSizeMB == nil || *source.patch.MaxSizeMB != 64 || source.patch.TTLSeconds != nil || source.patch.EmptyTTLSeconds != nil {
		t.Fatal("cache patch changed explicit false, capacity, revision or absent-field semantics", response.Errors)
	}
}
