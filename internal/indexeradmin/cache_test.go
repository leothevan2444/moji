package indexeradmin

import (
	"context"
	"encoding/json"
	"github.com/leothevan2444/moji/internal/config"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/indexer"
)

func TestCacheAdministrationIsAggregateOnlyAndSupportsScopedClear(t *testing.T) {
	service, store, _, _ := managementFixture(t)
	cache, err := indexer.NewSearchCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	bus := indexer.NewEvents()
	defer bus.Close()
	service.SetSearchCache(cache, bus)
	instance := indexer.InstanceSnapshot{ID: "one", Kind: indexer.Builtin, ConfigRevision: "revision", Settings: map[string]indexer.SettingValue{"password": {String: func() *string { s := "synthetic-cache-secret"; return &s }()}}}
	if _, err = cache.Search(context.Background(), instance, indexer.Query{Keywords: "synthetic-query-secret"}, CachePolicy(store.Config().Indexers.SearchCache), func(context.Context) (indexer.Batch, error) { return indexer.Batch{}, nil }); err != nil {
		t.Fatal(err)
	}
	status, err := service.CacheStatus(context.Background())
	if err != nil || status.Stats.Entries != 1 || status.Stats.UsedBytes == 0 || status.Revision != store.Revision() {
		t.Fatal("cache status did not reflect runtime/config snapshot", err)
	}
	public, err := json.Marshal(status)
	if err != nil || strings.Contains(string(public), "synthetic") {
		t.Fatal("aggregate projection exposed a lookup input")
	}
	status, err = service.ClearSearchCache(context.Background(), "another-source")
	if err != nil || status.Stats.Entries != 1 {
		t.Fatal("scoped clear removed another source", err)
	}
	status, err = service.ClearSearchCache(context.Background(), "one")
	if err != nil || status.Stats.Entries != 0 || status.Stats.UsedBytes != 0 {
		t.Fatal("scoped clear left cached results", err)
	}
	disabled := false
	status, err = service.UpdateSearchCache(context.Background(), store.Revision(), config.IndexerCachePatch{Enabled: &disabled})
	if err != nil || status.Enabled || status.Revision != store.Revision() {
		t.Fatal("saved policy acknowledgment did not match durable configuration", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = service.ClearSearchCache(ctx, ""); err == nil {
		t.Fatal("canceled command accepted")
	}
}
