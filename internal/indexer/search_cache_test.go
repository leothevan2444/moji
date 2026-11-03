package indexer

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func testSearchCache(t *testing.T) (*SearchCache, SearchCachePolicy, InstanceSnapshot) {
	t.Helper()
	cache, err := NewSearchCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cache.Close)
	return cache, SearchCachePolicy{Enabled: true, TTL: 5 * time.Minute, EmptyTTL: 30 * time.Second, MaxBytes: 1 << 20}, InstanceSnapshot{ID: "one", Kind: Builtin, Enabled: true, ConfigRevision: "account", BundleRevision: "rules", SiteTimeout: time.Minute}
}
func TestSearchCacheKeepsExactSourceQueryIsolationAndIndependentCandidates(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	query := Query{Mode: "search", Keywords: "ABCD-123", Limit: 100, Parameters: map[string]string{"year": "2026"}}
	calls := 0
	run := func(context.Context) (Batch, error) {
		calls++
		batch := candidateBatch(instance)
		batch.Releases[0].Resource.Locator.Fields = map[string]string{"title": "original"}
		return batch, nil
	}
	first, err := cache.Search(context.Background(), instance, query, policy, run)
	if err != nil {
		t.Fatal(err)
	}
	first.Releases[0].Resource.Locator.Fields["title"] = "mutated"
	second, err := cache.Search(context.Background(), instance, query, policy, run)
	if err != nil || !second.CacheHit || calls != 1 || first.Releases[0].Resource.ID == second.Releases[0].Resource.ID || second.Releases[0].Resource.Locator.Fields["title"] != "original" || !first.FetchedAt.Equal(second.FetchedAt) {
		t.Fatal("cache aliases candidates, changes freshness or reuses registration IDs", err)
	}
	other := instance
	other.ConfigRevision = "other-account"
	if _, err = cache.Search(context.Background(), other, query, policy, run); err != nil {
		t.Fatal(err)
	}
	other = instance
	other.BundleRevision = "other-rules"
	if _, err = cache.Search(context.Background(), other, query, policy, run); err != nil {
		t.Fatal(err)
	}
	other = instance
	other.Network.Revision = "other-network"
	if _, err = cache.Search(context.Background(), other, query, policy, run); err != nil {
		t.Fatal(err)
	}
	query.Keywords = "abcd-123"
	if _, err = cache.Search(context.Background(), instance, query, policy, run); err != nil || calls != 5 {
		t.Fatal("source/query contexts were incorrectly merged", err, calls)
	}
}

func TestSearchCacheCancellationDetachesOneWaiterAndJoinsApplicationWork(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	query := Query{Keywords: "one"}
	started := make(chan struct{})
	finish := make(chan struct{})
	var calls atomic.Int32
	run := func(ctx context.Context) (Batch, error) {
		calls.Add(1)
		close(started)
		select {
		case <-finish:
			return Batch{}, nil
		case <-ctx.Done():
			return Batch{}, ctx.Err()
		}
	}
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, err := cache.Search(firstCtx, instance, query, policy, run); first <- err }()
	<-started
	second := make(chan error, 1)
	go func() { _, err := cache.Search(context.Background(), instance, query, policy, run); second <- err }()
	waitSearchWaiters(t, cache, 2)
	cancelFirst()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatal("first waiter did not cancel", err)
	}
	close(finish)
	if err := <-second; err != nil || calls.Load() != 1 {
		t.Fatal("first cancellation aborted another waiter", err, calls.Load())
	}
	cache.Clear()
	blocked := make(chan struct{})
	third := make(chan error, 1)
	go func() {
		_, err := cache.Search(context.Background(), instance, query, policy, func(ctx context.Context) (Batch, error) { close(blocked); <-ctx.Done(); return Batch{}, ctx.Err() })
		third <- err
	}()
	<-blocked
	cache.Close()
	if err := <-third; !errors.Is(err, context.Canceled) {
		t.Fatal("application shutdown did not stop shared request", err)
	}
}

func waitSearchWaiters(t *testing.T, cache *SearchCache, count int) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		cache.mu.Lock()
		found := false
		for _, slot := range cache.slots {
			for _, flight := range slot.flights {
				if flight.waiters >= count {
					found = true
				}
			}
		}
		cache.mu.Unlock()
		if found {
			return
		}
		select {
		case <-deadline:
			t.Fatal("waiters did not attach")
		case <-tick.C:
		}
	}
}

func TestForcedRefreshFencesLateOldResultAndDoesNotReviveCacheOnFailure(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	query := Query{Keywords: "same"}
	started := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := cache.Search(context.Background(), instance, query, policy, func(context.Context) (Batch, error) { close(started); <-finish; return Batch{}, nil })
		done <- err
	}()
	<-started
	refresh := query
	refresh.CachePolicy = CacheRefresh
	if _, err := cache.Search(context.Background(), instance, refresh, policy, func(context.Context) (Batch, error) { return Batch{}, ErrAuthentication }); !errors.Is(err, ErrAuthentication) {
		t.Fatal("refresh failure hidden", err)
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	calls := 0
	if batch, err := cache.Search(context.Background(), instance, query, policy, func(context.Context) (Batch, error) { calls++; return Batch{}, nil }); err != nil || calls != 1 || batch.CacheHit {
		t.Fatal("late pre-refresh result restored stale cache", err)
	}
}

func TestServiceCacheReexecutesCurrentAcceptanceAndKeepsSequentialStop(t *testing.T) {
	cache, policy, first := testSearchCache(t)
	second := first
	second.ID = "two"
	var calls atomic.Int32
	var releases atomic.Int32
	adapter := fakeAdapter{search: func(_ context.Context, instance InstanceSnapshot, _ Query) (Batch, error) {
		calls.Add(1)
		batch := candidateBatch(instance)
		batch.release = func() { releases.Add(1) }
		return batch, nil
	}}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) {
		return Registry{Instances: []InstanceSnapshot{first, second}, Order: []string{"one", "two"}}, nil
	}), testResources{}, adapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.SetSearchCache(cache, func() SearchCachePolicy { return policy })
	var original string
	if _, err = service.SearchSequential(context.Background(), Query{Keywords: "ABCD-123"}, func(_ context.Context, batch Batch) (bool, error) {
		if batch.InstanceID == "one" {
			original = batch.Releases[0].Resource.ID
		}
		return false, nil
	}); err != nil {
		t.Fatal(err)
	}
	accepted := 0
	report, err := service.SearchSequential(context.Background(), Query{Keywords: "ABCD-123"}, func(_ context.Context, batch Batch) (bool, error) {
		accepted++
		if !batch.CacheHit || batch.Releases[0].Resource.ID == original {
			t.Error("cache bypassed reference registration boundary")
		}
		return true, nil
	})
	if err != nil || accepted != 1 || report.AcceptedInstanceID != "one" || len(report.Sites) != 1 || calls.Load() != 2 || releases.Load() != 2 {
		t.Fatal("cached search reused a selection decision or queried after acceptance", err, calls.Load(), releases.Load())
	}
}

func TestSearchCacheAllWaitersLeavingCancelsWorkAndInstanceBudgetEvicts(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	started := make(chan struct{})
	stopped := make(chan struct{})
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_, err := cache.Search(ctx, instance, Query{Keywords: "cancel"}, policy, func(worker context.Context) (Batch, error) {
			close(started)
			<-worker.Done()
			close(stopped)
			return Batch{}, worker.Err()
		})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("last waiter left shared request running")
	}
	calls := 0
	run := func(context.Context) (Batch, error) { calls++; return Batch{}, nil }
	for i := 0; i < 101; i++ {
		if _, err := cache.Search(context.Background(), instance, Query{Keywords: fmt.Sprint(i)}, policy, run); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cache.Search(context.Background(), instance, Query{Keywords: "0"}, policy, run); err != nil || calls != 102 {
		t.Fatal("per-instance query budget did not evict oldest result", err)
	}
	cache.mu.Lock()
	used := cache.used
	cache.mu.Unlock()
	if used > policy.MaxBytes {
		t.Fatal("cache exceeded byte budget")
	}
}

type authenticationFetchAdapter struct{ fakeAdapter }

func (authenticationFetchAdapter) Fetch(context.Context, InstanceSnapshot, ResourceRef) (Artifact, error) {
	return Artifact{}, ErrAuthentication
}

func TestAcquisitionAuthenticationFailureInvalidatesSearchResults(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	calls := 0
	adapter := authenticationFetchAdapter{fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		calls++
		return candidateBatch(instance), nil
	}}}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.SetSearchCache(cache, func() SearchCachePolicy { return policy })
	first, err := service.Search(context.Background(), Query{Keywords: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Search(context.Background(), Query{Keywords: "ABCD-123"}); err != nil || calls != 1 {
		t.Fatal("initial search did not cache", err)
	}
	if _, err = service.Fetch(context.Background(), instance, first.Releases[0].Resource); !errors.Is(err, ErrAuthentication) {
		t.Fatal("expected acquisition authentication failure", err)
	}
	if _, err = service.Search(context.Background(), Query{Keywords: "ABCD-123"}); err != nil || calls != 2 {
		t.Fatal("authentication failure left stale search results active", err)
	}
}

func TestSearchCacheOnlyStoresCompleteBuiltinSuccessAndExpiresWithoutSliding(t *testing.T) {
	cache, policy, instance := testSearchCache(t)
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	cache.now = func() time.Time { return time.Unix(0, clock.Load()) }
	query := Query{Keywords: "empty"}
	calls := 0
	run := func(context.Context) (Batch, error) { calls++; return Batch{}, nil }
	first, err := cache.Search(context.Background(), instance, query, policy, run)
	if err != nil {
		t.Fatal(err)
	}
	clock.Add(int64(20 * time.Second))
	hit, err := cache.Search(context.Background(), instance, query, policy, run)
	if err != nil || !hit.CacheHit || !hit.FetchedAt.Equal(first.FetchedAt) {
		t.Fatal("cache hit extended fetched time", err)
	}
	clock.Add(int64(11 * time.Second))
	if _, err = cache.Search(context.Background(), instance, query, policy, run); err != nil || calls != 2 {
		t.Fatal("empty result outlived its original TTL", err)
	}
	for _, bad := range []Batch{{Partial: true}, {Truncated: true}, {IssueCount: 1}} {
		cache.Clear()
		calls = 0
		for i := 0; i < 2; i++ {
			if _, err = cache.Search(context.Background(), instance, query, policy, func(context.Context) (Batch, error) { calls++; return bad, nil }); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 2 {
			t.Fatal("incomplete/truncated response cached as complete")
		}
	}
	for _, kind := range []Kind{Jackett, Builtin} {
		instance.Kind = kind
		policy.Enabled = kind == Jackett
		cache.Clear()
		calls = 0
		for i := 0; i < 2; i++ {
			if _, err = cache.Search(context.Background(), instance, query, policy, run); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 2 {
			t.Fatal("external search or disabled cache received Moji result caching")
		}
	}
}
