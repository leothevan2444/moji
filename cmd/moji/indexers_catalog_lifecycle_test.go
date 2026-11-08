package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

func catalogWorker(t *testing.T, store *config.Store, adapter indexerCatalogFunc) (*indexerRuntime, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	configuration := indexerConfiguration{store: store}
	r := &indexerRuntime{configuration: configuration, registry: &runtimeIndexerRegistry{configuration: configuration, external: adapter, events: indexer.NewEvents()}}
	done := make(chan struct{})
	go func() { defer close(done); r.maintainJackettCatalog(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("catalog worker failed to stop")
		}
	})
	return r, cancel, done
}
func waitForCatalog(t *testing.T, r *indexerRuntime, name string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := r.registry.CachedSnapshot(context.Background())
		if err == nil && snapshot.JackettCatalog.Known && len(snapshot.Instances) == 1 && snapshot.Instances[0].Name == name {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("catalog was not loaded automatically", name)
}
func TestJackettCatalogLoadsOnWorkerStartupAndConnectionChange(t *testing.T) {
	store := indexerTestStore(t)
	var calls atomic.Int32
	r, _, _ := catalogWorker(t, store, indexerCatalogFunc(func(ctx context.Context, connection indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		calls.Add(1)
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 31*time.Second {
			t.Error("catalog request is not bounded")
		}
		return []indexer.InstanceSnapshot{{ID: "jackett/one", RemoteID: "one", Name: connection.JackettAPIKey}}, nil
	}))
	waitForCatalog(t, r, "synthetic")
	if calls.Load() != 1 {
		t.Fatal("initial notification bypassed catalog cache", calls.Load())
	}
	if _, err := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"api_key": "changed-account"}}}); err != nil {
		t.Fatal(err)
	}
	waitForCatalog(t, r, "changed-account")
	if calls.Load() != 2 {
		t.Fatal("account change was not loaded once", calls.Load())
	}
}
func TestJackettCatalogDoesNotLoadDisabledOrIncompleteConnections(t *testing.T) {
	for _, patch := range []map[string]any{{"enabled": false}, {"url": ""}, {"api_key": ""}} {
		t.Run("missing configuration", func(t *testing.T) {
			store := indexerTestStore(t)
			if _, err := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": patch}}); err != nil {
				t.Fatal(err)
			}
			calls := 0
			configuration := indexerConfiguration{store: store}
			r := &indexerRuntime{configuration: configuration, registry: &runtimeIndexerRegistry{configuration: configuration, external: indexerCatalogFunc(func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
				calls++
				return nil, nil
			})}}
			r.refreshJackettCatalog(context.Background())
			if calls != 0 {
				t.Fatal("unconfigured catalog was contacted")
			}
		})
	}
}
func TestJackettCatalogWorkerLoadsWhenEnabledAndPreservesConfirmedSitesOnFailure(t *testing.T) {
	store := indexerTestStore(t)
	if _, err := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	var failed atomic.Bool
	var calls atomic.Int32
	r, _, _ := catalogWorker(t, store, indexerCatalogFunc(func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		calls.Add(1)
		if failed.Load() {
			return nil, indexer.ErrTransport
		}
		return []indexer.InstanceSnapshot{{ID: "jackett/one", RemoteID: "one", Name: "One"}}, nil
	}))
	if _, err := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": true}}}); err != nil {
		t.Fatal(err)
	}
	waitForCatalog(t, r, "One")
	failed.Store(true)
	r.registry.mu.Lock()
	r.registry.catalogAt = time.Now().Add(-6 * time.Minute)
	r.registry.mu.Unlock()
	r.refreshJackettCatalog(context.Background())
	snapshot, err := r.registry.CachedSnapshot(context.Background())
	if err != nil || !snapshot.JackettCatalog.Known || !snapshot.JackettCatalog.Stale || len(snapshot.Instances) != 1 || len(snapshot.Diagnostics) != 1 {
		t.Fatal("failure discarded known sites", err, snapshot)
	}
	r.refreshJackettCatalog(context.Background())
	if calls.Load() != 2 {
		t.Fatal("failure backoff bypassed", calls.Load())
	}
	failed.Store(false)
	r.registry.mu.Lock()
	r.registry.attemptAt = time.Now().Add(-time.Minute)
	r.registry.mu.Unlock()
	if _, err = store.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() != 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() != 3 {
		t.Fatal("worker failed to retry after backoff")
	}
}
func TestJackettCatalogRequestStopsWithApplication(t *testing.T) {
	store := indexerTestStore(t)
	entered := make(chan struct{})
	_, cancel, done := catalogWorker(t, store, indexerCatalogFunc(func(ctx context.Context, _ indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}))
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("startup did not load catalog")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("catalog request outlived application")
	}
}

func TestRuntimeAutomaticallyLoadsConfiguredJackettWithoutPageOrSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requests := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/indexers/all/results/torznab/api" {
			t.Errorf("startup contacted non-catalog endpoint %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, `<indexers><indexer id="one" configured="true"><title>Automatically loaded</title></indexer></indexers>`)
		select {
		case requests <- struct{}{}:
		default:
		}
	}))
	defer server.Close()
	store := indexerTestStore(t)
	if _, err := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"url": server.URL}}}); err != nil {
		t.Fatal(err)
	}
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, &runtimeRuleSource{})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); runtime.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("runtime failed to stop")
		}
	}()
	select {
	case <-requests:
	case <-time.After(3 * time.Second):
		t.Fatal("runtime startup did not request Jackett catalog")
	}
	waitForCatalog(t, runtime, "Automatically loaded")
	summary, err := runtime.admin.Summary(ctx, 72)
	if err != nil || summary.Total != 1 {
		t.Fatal("loaded catalog not reflected in Home", summary, err)
	}
}
