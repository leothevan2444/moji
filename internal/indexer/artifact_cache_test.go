package indexer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func artifactFixture(t *testing.T) (*ArtifactCache, InstanceSnapshot, ResourceRef) {
	t.Helper()
	cache, err := NewArtifactCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cache.Close)
	instance := InstanceSnapshot{ID: "one", Kind: Jackett, ConfigRevision: "account", Network: NetworkProfile{Revision: "private", ProxyURL: "http://synthetic-user:synthetic-password@proxy.example"}}
	return cache, instance, ResourceRef{Version: 1, ID: "first", InstanceID: instance.ID, Kind: Jackett, ConfigRevision: instance.ConfigRevision, Locator: Locator{DownloadURL: "https://site.example/grab?ticket=synthetic-ticket", Title: "ABCD-123"}}
}
func artifactBytes(announce string) []byte {
	info := "d6:lengthi1e4:name12:ABCD-123.mp4e"
	if announce == "" {
		return []byte("d4:info" + info + "e")
	}
	return []byte(fmt.Sprintf("d8:announce%d:%s4:info%se", len(announce), announce, info))
}
func awaitArtifactState(t *testing.T, c *ArtifactCache, predicate func() bool) {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		c.mu.Lock()
		ready := predicate()
		c.mu.Unlock()
		if ready {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("artifact coordinator did not reach expected state")
		case <-tick.C:
		}
	}
}
func TestAcquisitionMergesOpaqueReferencesAndKeepsWaiterCancellationIndependent(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	var calls atomic.Int32
	gate := make(chan struct{})
	defer close(gate)
	started := make(chan struct{})
	run := func(ctx context.Context) (Artifact, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-gate:
			return Artifact{TorrentBytes: artifactBytes("")}, nil
		case <-ctx.Done():
			return Artifact{}, ctx.Err()
		}
	}
	firstCtx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, err := cache.Fetch(firstCtx, instance, ref, run); first <- err }()
	<-started
	otherRef := ref
	otherRef.ID = "second"
	second := make(chan Artifact, 1)
	failure := make(chan error, 1)
	go func() {
		artifact, err := cache.Fetch(context.Background(), instance, otherRef, run)
		second <- artifact
		failure <- err
	}()
	awaitArtifactState(t, cache, func() bool { return cache.waiters == 2 })
	cancel()
	if !errors.Is(<-first, context.Canceled) {
		t.Fatal("waiter cancellation ignored")
	}
	// Release without closing the channel twice in failure cleanup.
	gate <- struct{}{}
	result := <-second
	if err := <-failure; err != nil || result.Resource.ID != "second" || calls.Load() != 1 {
		t.Fatal("shared acquisition duplicated or lost caller identity", err)
	}
	result.TorrentBytes[0] = 'x'
	result.Resource.Locator.DownloadURL = "changed"
	hit, err := cache.Fetch(context.Background(), instance, ref, run)
	if err != nil || calls.Load() != 1 || hit.Resource.ID != "first" || hit.TorrentBytes[0] != 'd' {
		t.Fatal("cached result shared mutable bytes/reference", err)
	}
}
func TestAcquisitionTTLAndSourceIsolationDoNotSlideOrCrossAccounts(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	now := time.Now().UTC()
	cache.now = func() time.Time { return now }
	calls := 0
	run := func(context.Context) (Artifact, error) {
		calls++
		return Artifact{TorrentBytes: artifactBytes("")}, nil
	}
	first, err := cache.Fetch(context.Background(), instance, ref, run)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(9 * time.Minute)
	hit, err := cache.Fetch(context.Background(), instance, ref, run)
	if err != nil || calls != 1 || hit.AcquiredAt != first.AcquiredAt {
		t.Fatal("cache hit faked freshness", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err = cache.Fetch(context.Background(), instance, ref, run); err != nil || calls != 2 {
		t.Fatal("reads extended acquisition TTL", err)
	}
	account := instance
	account.ConfigRevision = "another-account"
	accountRef := ref
	accountRef.ConfigRevision = account.ConfigRevision
	if _, err = cache.Fetch(context.Background(), account, accountRef, run); err != nil {
		t.Fatal(err)
	}
	network := instance
	network.Network.ProxyURL = "http://different-proxy.example"
	if _, err = cache.Fetch(context.Background(), network, ref, run); err != nil {
		t.Fatal(err)
	}
	rule := ref
	rule.DefinitionDigest = strings.Repeat("a", 64)
	if _, err = cache.Fetch(context.Background(), instance, rule, run); err != nil {
		t.Fatal(err)
	}
	locator := ref
	locator.Locator.DownloadURL = "https://site.example/another-ticket"
	if _, err = cache.Fetch(context.Background(), instance, locator, run); err != nil || calls != 6 {
		t.Fatal("cache crossed frozen source/locator boundary", err, calls)
	}
}
func TestAcquisitionDeduplicatesExactMetainfoButNeverInterchangesTrackerParameters(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	first := artifactBytes("https://tracker.example/a")
	second := artifactBytes("https://tracker.example/b")
	fetch := func(data []byte, locator string) {
		t.Helper()
		current := ref
		current.Locator.DownloadURL = locator
		if _, err := cache.Fetch(context.Background(), instance, current, func(context.Context) (Artifact, error) { return Artifact{TorrentBytes: data}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	fetch(first, "https://site.example/one")
	fetch(first, "https://site.example/two")
	cache.mu.Lock()
	if len(cache.blobs) != 1 || len(cache.entries) != 2 {
		t.Error("same actual hash/bytes were not deduplicated")
	}
	cache.mu.Unlock()
	fetch(second, "https://site.example/three")
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.blobs) != 2 {
		t.Fatal("same info hash interchanged different tracker data")
	}
	hashes := map[string]bool{}
	for _, blob := range cache.blobs {
		hashes[blob.infoHash] = true
	}
	if len(hashes) != 1 {
		t.Fatal("fixture did not actually share its info dictionary hash")
	}
	for key := range cache.entries {
		if len(key) != 64 || strings.Contains(key, "synthetic") {
			t.Fatal("lookup exported locator or credential")
		}
	}
}
func TestAcquisitionDoesNotRetryOrCacheFailuresAndInvalidBodies(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		artifact Artifact
		err      error
	}{
		{"transport", Artifact{}, ErrTransport}, {"authentication", Artifact{}, ErrAuthentication}, {"html", Artifact{TorrentBytes: []byte("<html>challenge</html>")}, nil}, {"mixed", Artifact{TorrentBytes: artifactBytes(""), Magnet: "magnet:?xt=urn:btih:a"}, nil}, {"not-magnet", Artifact{Magnet: "https://private.example/ticket"}, nil},
		{"short-hash", Artifact{Magnet: "magnet:?xt=urn:btih:a"}, nil},
		{"wrong-namespace", Artifact{Magnet: "magnet:?xt=urn:sha1:" + strings.Repeat("a", 40)}, nil},
		{"invalid-base32", Artifact{Magnet: "magnet:?xt=urn:btih:" + strings.Repeat("0", 32)}, nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cache, instance, ref := artifactFixture(t)
			var calls atomic.Int32
			run := func(context.Context) (Artifact, error) { calls.Add(1); return scenario.artifact, scenario.err }
			for range 2 {
				if _, err := cache.Fetch(context.Background(), instance, ref, run); err == nil {
					t.Fatal("invalid acquisition succeeded")
				}
			}
			if calls.Load() != 2 {
				t.Fatal("failure was automatically retried or cached")
			}
		})
	}
}

func TestAcquisitionRetainsMagnetTrackersAndCanonicalIdentity(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	const hash = "0123456789abcdef0123456789abcdef01234567"
	values := []struct{ uri, hash string }{
		{"magnet:?xt=urn:btih:" + hash + "&dn=ABCD%20123&tr=https%3A%2F%2Fone.fixture.invalid%2Fannounce", hash},
		{"magnet:?xt=urn:btih:AERUKZ4JVPG66AJDIVTYTK6N54ASGRLH&dn=ABCD%20123&tr=https%3A%2F%2Ftwo.fixture.invalid%2Fannounce", hash},
		{"magnet:?xt=urn:btmh:1220" + strings.Repeat("b", 64) + "&dn=ABCD-123", strings.Repeat("b", 64)},
	}
	for i, value := range values {
		current := ref
		current.Locator.DownloadURL += fmt.Sprint(i)
		var calls int
		for range 2 {
			artifact, err := cache.Fetch(context.Background(), instance, current, func(context.Context) (Artifact, error) {
				calls++
				return Artifact{Magnet: value.uri}, nil
			})
			if err != nil || artifact.Magnet != value.uri || artifact.infoHash != value.hash || len(artifact.TorrentBytes) != 0 {
				t.Fatalf("magnet changed across validation/cache: %+v %v", artifact, err)
			}
		}
		if calls != 1 {
			t.Fatal("successful magnet was fetched again", calls)
		}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.blobs) != len(values) || len(cache.entries) != len(values) {
		t.Fatal("equal hashes interchanged magnet metadata")
	}
}
func TestLastWaiterCancellationCannotStartAnOverlappingGrab(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	finish := make(chan struct{})
	defer close(finish)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := cache.Fetch(ctx, instance, ref, func(worker context.Context) (Artifact, error) {
			close(started)
			<-worker.Done()
			close(cancelled)
			<-finish
			return Artifact{}, worker.Err()
		})
		result <- err
	}()
	<-started
	cancel()
	<-result
	<-cancelled
	_, err := cache.Fetch(context.Background(), instance, ref, func(context.Context) (Artifact, error) {
		t.Error("overlapping grab started before cancelled work exited")
		return Artifact{}, nil
	})
	if !errors.Is(err, ErrBusy) {
		t.Fatal("new acquisition overlapped a cancelling grab", err)
	}
	finish <- struct{}{}
	awaitArtifactState(t, cache, func() bool { return len(cache.flights) == 0 })
	if _, err = cache.Fetch(context.Background(), instance, ref, func(context.Context) (Artifact, error) { return Artifact{TorrentBytes: artifactBytes("")}, nil }); err != nil {
		t.Fatal("explicit later request could not recover", err)
	}
}
func TestClearAndCloseFenceLateAcquisitionAndJoinWorkers(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	started := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := cache.Fetch(context.Background(), instance, ref, func(context.Context) (Artifact, error) {
			close(started)
			<-finish
			return Artifact{TorrentBytes: artifactBytes("")}, nil
		})
		done <- err
	}()
	<-started
	cache.Clear()
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	awaitArtifactState(t, cache, func() bool { return len(cache.flights) == 0 })
	cache.mu.Lock()
	if len(cache.entries) != 0 {
		t.Error("late result repopulated cleared cache")
	}
	cache.mu.Unlock()
	started = make(chan struct{})
	exited := make(chan struct{})
	go func() {
		_, _ = cache.Fetch(context.Background(), instance, ref, func(worker context.Context) (Artifact, error) {
			close(started)
			<-worker.Done()
			close(exited)
			return Artifact{}, worker.Err()
		})
	}()
	<-started
	cache.Close()
	select {
	case <-exited:
	default:
		t.Fatal("close returned before its worker")
	}
}
func TestArtifactBudgetEvictsLRUAndSharesImmutableOriginalBytes(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	base := artifactBytes("")
	payload := bytes.Repeat([]byte("x"), 7<<20)
	for i := range 10 {
		data := append([]byte(fmt.Sprintf("d7:comment%d:", len(payload))), payload...)
		data = append(data, base[1:]...)
		current := ref
		current.Locator.DownloadURL = fmt.Sprintf("https://site.example/%d", i)
		if _, err := cache.Fetch(context.Background(), instance, current, func(context.Context) (Artifact, error) { return Artifact{TorrentBytes: data}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	// Fill with distinct original bodies to exercise the actual 64 MiB limit.
	for i := range 10 {
		data := append([]byte(fmt.Sprintf("d7:comment%d:", len(payload)+1)), payload...)
		data = append(data, byte('a'+i))
		data = append(data, base[1:]...)
		current := ref
		current.Locator.DownloadURL = fmt.Sprintf("https://site.example/unique-%d", i)
		if _, err := cache.Fetch(context.Background(), instance, current, func(context.Context) (Artifact, error) { return Artifact{TorrentBytes: data}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	// Equivalent bodies share one blob even across ten independent locators.
	if len(cache.blobs) != 9 || len(cache.entries) != 9 || cache.used > artifactMaxBytes {
		t.Fatal("bounded acquisition payload accounting incorrect")
	}
}

type acquisitionAdapter struct{ calls atomic.Int32 }

func (a *acquisitionAdapter) Search(context.Context, InstanceSnapshot, Query) (Batch, error) {
	return Batch{}, nil
}
func (a *acquisitionAdapter) Fetch(context.Context, InstanceSnapshot, ResourceRef) (Artifact, error) {
	a.calls.Add(1)
	return Artifact{TorrentBytes: artifactBytes("")}, nil
}

type acquisitionHealth struct{ count atomic.Int32 }

func (h *acquisitionHealth) Observe(context.Context, InstanceSnapshot, HealthOperation, time.Time, error, string) (bool, error) {
	h.count.Add(1)
	return true, nil
}
func TestUnifiedServiceReusesAcquisitionAcrossReferenceAndCorePathsWithoutFakingHealth(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	adapter := &acquisitionAdapter{}
	health := &acquisitionHealth{}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{}, nil }), testResources{}, nil, adapter)
	if err != nil {
		t.Fatal(err)
	}
	service.SetArtifactCache(cache)
	service.SetHealthRecorder(health, nil)
	first, err := service.Fetch(context.Background(), instance, ref)
	if err != nil {
		t.Fatal(err)
	}
	ref.ID = "second-reference"
	record := ResourceRecord{Version: 1, Instance: instance, Ref: ref}
	second, err := service.FetchRecord(context.Background(), record)
	if err != nil || adapter.calls.Load() != 1 || health.count.Load() != 1 || second.Resource.ID != ref.ID || first.AcquiredAt != second.AcquiredAt {
		t.Fatal("core/reference reuse duplicated upstream acquisition or health", err)
	}
}

func TestAcquisitionCapacityRejectsExcessWorkAndReleasesCancelledWaiters(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, artifactMaxOperations)
	done := make(chan struct{}, artifactMaxOperations)
	for i := range artifactMaxOperations {
		current := ref
		current.Locator.DownloadURL = fmt.Sprintf("https://site.example/capacity-%d", i)
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = cache.Fetch(ctx, instance, current, func(worker context.Context) (Artifact, error) {
				started <- struct{}{}
				<-worker.Done()
				return Artifact{}, worker.Err()
			})
		}()
	}
	for range artifactMaxOperations {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("work capacity did not fill")
		}
	}
	extra := ref
	extra.Locator.DownloadURL = "https://site.example/excess"
	if _, err := cache.Fetch(context.Background(), instance, extra, func(context.Context) (Artifact, error) { return Artifact{}, nil }); !errors.Is(err, ErrBusy) {
		t.Fatal("acquisition queue accepted work past capacity", err)
	}
	cancel()
	for range artifactMaxOperations {
		<-done
	}
	awaitArtifactState(t, cache, func() bool { return cache.waiters == 0 && len(cache.flights) == 0 })
}
func TestKnownSuccessfulLateAcquisitionCanBeReusedAfterItsLastWaiterLeaves(t *testing.T) {
	cache, instance, ref := artifactFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cache.Fetch(ctx, instance, ref, func(context.Context) (Artifact, error) {
			close(started)
			<-finish
			return Artifact{TorrentBytes: artifactBytes("")}, nil
		})
	}()
	<-started
	cancel()
	<-done
	close(finish)
	awaitArtifactState(t, cache, func() bool { return len(cache.flights) == 0 })
	var additional atomic.Int32
	result, err := cache.Fetch(context.Background(), instance, ref, func(context.Context) (Artifact, error) { additional.Add(1); return Artifact{}, ErrTransport })
	if err != nil || additional.Load() != 0 || !bytes.Equal(result.TorrentBytes, artifactBytes("")) {
		t.Fatal("known acquired data was discarded and grabbed again", err)
	}
}
