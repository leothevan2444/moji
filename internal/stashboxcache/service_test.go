package stashboxcache

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type fakeClient struct {
	mu             sync.Mutex
	scenes         []*stashboxgraphql.SceneFragment
	queryCalls     []stashboxgraphql.SceneQueryInput
	queryErr       error
	performer      *stashboxgraphql.PerformerFragment
	performerCalls int
	performerErr   error
}

func (f *fakeClient) FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.performerCalls++
	return f.performer, f.performerErr
}

func (f *fakeClient) QueryScenesPage(_ context.Context, input stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queryCalls = append(f.queryCalls, input)
	if f.queryErr != nil {
		return stashboxpkg.ScenePage{}, f.queryErr
	}
	start := (input.Page - 1) * input.PerPage
	if start > len(f.scenes) {
		start = len(f.scenes)
	}
	end := start + input.PerPage
	if end > len(f.scenes) {
		end = len(f.scenes)
	}
	return stashboxpkg.ScenePage{Scenes: append([]*stashboxgraphql.SceneFragment(nil), f.scenes[start:end]...), Count: len(f.scenes)}, nil
}

func newTestService(t *testing.T, now *time.Time) *Service {
	t.Helper()
	service, err := New(filepath.Join(t.TempDir(), "cache.db"), func() Config { return Config{TTL: 24 * time.Hour, StaleRetention: 30 * 24 * time.Hour} })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	service.now = func() time.Time { return *now }
	t.Cleanup(func() { _ = service.Close() })
	return service
}

func TestLoadExpandsInFortyItemBatchesAndHitsCache(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(85)}

	first, err := service.Load(context.Background(), client, "HTTPS://BOX/GRAPHQL", "p1", 24, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	if first.LoadedCount != 40 || first.RemoteCount != 85 || first.Complete || len(client.queryCalls) != 1 {
		t.Fatalf("unexpected first result: %+v calls=%d", first, len(client.queryCalls))
	}
	second, err := service.Load(context.Background(), client, "https://box/graphql", "p1", 48, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	if second.LoadedCount != 80 || len(client.queryCalls) != 2 || client.queryCalls[1].Page != 2 || client.queryCalls[1].PerPage != 40 {
		t.Fatalf("unexpected second result: %+v calls=%+v", second, client.queryCalls)
	}
	if _, err := service.Load(context.Background(), client, "https://box/graphql", "p1", 72, CachePreferred); err != nil {
		t.Fatal(err)
	}
	if len(client.queryCalls) != 2 {
		t.Fatalf("fresh coverage should be reused, calls=%d", len(client.queryCalls))
	}
	last, err := service.Load(context.Background(), client, "https://box/graphql", "p1", 85, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	if last.LoadedCount != 85 || !last.Complete || len(client.queryCalls) != 3 {
		t.Fatalf("unexpected complete result: %+v calls=%d", last, len(client.queryCalls))
	}
}

func TestExpiredBrowseFallsBackToStaleButRequiredFreshFails(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(12)}
	if _, err := service.Load(context.Background(), client, "https://box", "p1", 12, CachePreferred); err != nil {
		t.Fatal(err)
	}
	now = now.Add(25 * time.Hour)
	client.queryErr = errors.New("upstream unavailable")
	stale, err := service.Load(context.Background(), client, "https://box", "p1", 12, CachePreferred)
	if err != nil || !stale.Stale || stale.LoadedCount != 12 {
		t.Fatalf("expected stale fallback, result=%+v err=%v", stale, err)
	}
	if _, err := service.Load(context.Background(), client, "https://box", "p1", 12, RequireFresh); !errors.Is(err, client.queryErr) && err == nil {
		t.Fatalf("RequireFresh error=%v", err)
	}
}

func TestRefreshHeadCreatesGenerationWhenOrderingChanges(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(50)}
	before, err := service.Load(context.Background(), client, "https://box", "p1", 50, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	client.scenes = append([]*stashboxgraphql.SceneFragment{{ID: "new"}}, client.scenes...)
	now = now.Add(time.Hour)
	after, err := service.Load(context.Background(), client, "https://box", "p1", 40, RefreshHead)
	if err != nil {
		t.Fatal(err)
	}
	if after.Generation <= before.Generation || after.LoadedCount != 40 || after.Scenes[0].ID != "new" || after.RemoteCount != 51 {
		t.Fatalf("generation was not replaced: before=%+v after=%+v", before, after)
	}
}

func TestRefreshHeadKeepsCachedTailWhenOrderingIsUnchanged(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(85)}
	if _, err := service.Load(context.Background(), client, "https://box", "p1", 80, CachePreferred); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	refreshed, err := service.Load(context.Background(), client, "https://box", "p1", 40, RefreshHead)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.LoadedCount != 80 {
		t.Fatalf("head refresh discarded cached tail: %+v", refreshed)
	}
	loaded, err := service.Load(context.Background(), client, "https://box", "p1", 80, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LoadedCount != 80 || len(client.queryCalls) != 3 {
		t.Fatalf("tail was fetched again: result=%+v calls=%d", loaded, len(client.queryCalls))
	}
}

func TestBrowseFallsBackToCompletePreviousGeneration(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(85)}
	before, err := service.Load(context.Background(), client, "https://box", "p1", 85, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	client.scenes = append([]*stashboxgraphql.SceneFragment{{ID: "new"}}, client.scenes...)
	now = now.Add(time.Hour)
	if _, err := service.Load(context.Background(), client, "https://box", "p1", 40, RefreshHead); err != nil {
		t.Fatal(err)
	}
	client.queryErr = errors.New("upstream unavailable")
	stale, err := service.Load(context.Background(), client, "https://box", "p1", 80, CachePreferred)
	if err != nil {
		t.Fatal(err)
	}
	if !stale.Stale || stale.Generation != before.Generation || stale.LoadedCount != 85 {
		t.Fatalf("did not preserve complete previous generation: before=%+v stale=%+v", before, stale)
	}
}

func TestPerformerTTLStatusClearAndCleanup(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(2), performer: &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Alice"}}
	if _, _, err := service.ResolvePerformer(context.Background(), client, "https://box", "p1", CachePreferred); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ResolvePerformer(context.Background(), client, "https://box", "p1", CachePreferred); err != nil {
		t.Fatal(err)
	}
	if client.performerCalls != 1 {
		t.Fatalf("performer calls=%d", client.performerCalls)
	}
	if _, err := service.Load(context.Background(), client, "https://box", "p1", 2, CachePreferred); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(context.Background())
	if err != nil || status.SceneCount != 2 || status.PerformerCount != 1 || status.SnapshotCount != 1 || status.UsedBytes == 0 {
		t.Fatalf("unexpected status: %+v err=%v", status, err)
	}
	now = now.Add(31 * 24 * time.Hour)
	if err := service.Cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, _ = service.Status(context.Background())
	if status.SceneCount != 0 || status.PerformerCount != 0 || status.SnapshotCount != 0 {
		t.Fatalf("cleanup status: %+v", status)
	}
	if _, err := service.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCacheHitsThrottleAccessWritesAndRetainScenesThroughSnapshots(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	client := &fakeClient{scenes: testScenes(2), performer: &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Alice"}}
	ctx := context.Background()

	if _, _, err := service.ResolvePerformer(ctx, client, "https://box", "p1", CachePreferred); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Load(ctx, client, "https://box", "p1", 2, CachePreferred); err != nil {
		t.Fatal(err)
	}
	initialChanges := totalChanges(t, service)

	now = now.Add(59 * time.Minute)
	if _, _, err := service.ResolvePerformer(ctx, client, "https://box", "p1", CachePreferred); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Load(ctx, client, "https://box", "p1", 2, CachePreferred); err != nil {
		t.Fatal(err)
	}
	if changes := totalChanges(t, service); changes != initialChanges {
		t.Fatalf("cache hits inside touch interval wrote to SQLite: before=%d after=%d", initialChanges, changes)
	}

	now = now.Add(time.Minute)
	if _, _, err := service.ResolvePerformer(ctx, client, "https://box", "p1", CachePreferred); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Load(ctx, client, "https://box", "p1", 2, CachePreferred); err != nil {
		t.Fatal(err)
	}
	if changes := totalChanges(t, service); changes != initialChanges+2 {
		t.Fatalf("expected one performer and one snapshot touch after an hour: before=%d after=%d", initialChanges, changes)
	}
	if client.performerCalls != 1 || len(client.queryCalls) != 1 {
		t.Fatalf("cache hits reached upstream: performer=%d scenes=%d", client.performerCalls, len(client.queryCalls))
	}

	now = now.Add(30*24*time.Hour - 30*time.Minute)
	if err := service.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.SceneCount != 2 || status.PerformerCount != 1 || status.SnapshotCount != 1 {
		t.Fatalf("recent snapshot data was removed: %+v", status)
	}

	now = now.Add(time.Hour)
	if err := service.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.SceneCount != 0 || status.PerformerCount != 0 || status.SnapshotCount != 0 {
		t.Fatalf("expired snapshot data was retained: %+v", status)
	}
}

func TestSceneSchemaContainsOnlyEntityData(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	service := newTestService(t, &now)
	var columns []struct {
		Name string `db:"name"`
	}
	if err := service.store.db.Select(&columns, `SELECT name FROM pragma_table_info('stashbox_cache_scenes')`); err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(columns))
	for _, column := range columns {
		names[column.Name] = true
	}
	if names["fetched_at"] || names["last_accessed_at"] {
		t.Fatalf("scene schema retains redundant timestamps: %+v", names)
	}
	for _, required := range []string{"endpoint", "scene_id", "payload_json"} {
		if !names[required] {
			t.Fatalf("scene schema missing %s: %+v", required, names)
		}
	}
}

func totalChanges(t *testing.T, service *Service) int64 {
	t.Helper()
	var changes int64
	if err := service.store.db.Get(&changes, `SELECT total_changes()`); err != nil {
		t.Fatal(err)
	}
	return changes
}

func testScenes(count int) []*stashboxgraphql.SceneFragment {
	out := make([]*stashboxgraphql.SceneFragment, count)
	for index := range out {
		title := fmt.Sprintf("Scene %d", index+1)
		out[index] = &stashboxgraphql.SceneFragment{ID: fmt.Sprintf("s-%03d", index+1), Title: &title}
	}
	return out
}
