package indexer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

// Explicit benchmarks use fake callbacks/HTTP; never load user configuration.
func BenchmarkIndexerCache(b *testing.B) {
	for _, mode := range []string{"Hit1000", "Refresh1000", "Pressure1000"} {
		b.Run(mode, func(b *testing.B) {
			cache, err := NewSearchCache(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			defer cache.Close()
			policy := SearchCachePolicy{Enabled: true, TTL: 5 * time.Minute, EmptyTTL: 30 * time.Second, MaxBytes: 32 << 20}
			instance := InstanceSnapshot{ID: "synthetic", Kind: Builtin, Enabled: true, ConfigRevision: "account", BundleRevision: "rules", SiteTimeout: time.Minute}
			batch := Batch{InstanceID: instance.ID}
			for i := range 1000 {
				row := candidateBatch(instance).Releases[0]
				row.Title = fmt.Sprintf("ABCD-123 item %d", i)
				row.Resource.Locator = Locator{DownloadURL: fmt.Sprintf("https://site.invalid/%d.torrent", i), Fields: map[string]string{"title": row.Title}}
				batch.Releases = append(batch.Releases, row)
			}
			var calls atomic.Int64
			run := func(context.Context) (Batch, error) { calls.Add(1); return batch, nil }
			query := Query{Keywords: "ABCD-123", Limit: 1000}
			if mode == "Hit1000" {
				if _, err := cache.Search(context.Background(), instance, query, policy, run); err != nil {
					b.Fatal(err)
				}
			}
			before := calls.Load()
			samples := make([]time.Duration, b.N)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				if mode == "Refresh1000" {
					query.CachePolicy = CacheRefresh
				} else if mode == "Pressure1000" {
					query.Keywords = fmt.Sprintf("ABCD-123-%d", i)
				}
				start := time.Now()
				result, err := cache.Search(context.Background(), instance, query, policy, run)
				samples[i] = time.Since(start)
				if err != nil || len(result.Releases) != 1000 || result.CacheHit != (mode == "Hit1000") {
					b.Fatal("cache benchmark lost rows or changed hit semantics", err)
				}
			}
			b.StopTimer()
			stats, err := cache.Status(policy)
			if err != nil || stats.UsedBytes > policy.MaxBytes || stats.ActiveRequests != 0 || stats.ActiveWaiters != 0 {
				b.Fatal("cache budget or worker ownership violated", stats, err)
			}
			invocations := calls.Load() - before
			if mode == "Hit1000" && invocations != 0 || mode != "Hit1000" && invocations != int64(b.N) {
				b.Fatal("cache changed source request count", invocations)
			}
			reportIndexerLatency(b, samples)
			b.ReportMetric(float64(stats.UsedBytes), "cache-B")
			b.ReportMetric(float64(stats.Entries), "cache-entries")
			b.ReportMetric(float64(invocations)/float64(b.N), "source-calls/op")
		})
	}
}

func benchmarkNetwork(b *testing.B, respond func(*http.Request) (*http.Response, error)) *Network {
	b.Helper()
	network := NewNetwork(context.Background())
	b.Cleanup(network.Close)
	network.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("1.1.1.1")}}, nil
	})
	network.roundTrip = respond
	return network
}

func BenchmarkIndexerPacedOrigin(b *testing.B) {
	var calls atomic.Int64
	network := benchmarkNetwork(b, func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return httpResponse(200, nil, "ok"), nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, nil)
	if err != nil {
		b.Fatal(err)
	}
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		start := time.Now()
		_, err := session.Do(context.Background(), cardigann.Request{URL: "https://site.invalid/search"})
		samples[i] = time.Since(start)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if calls.Load() != int64(b.N) {
		b.Fatal("normal pacing lost or repeated a request", calls.Load())
	}
	reportIndexerLatency(b, samples)
	b.ReportMetric(float64(calls.Load())/float64(b.N), "http-calls/op")
}

func BenchmarkIndexerShared429(b *testing.B) {
	var calls atomic.Int64
	network := benchmarkNetwork(b, func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return httpResponse(429, http.Header{"Retry-After": []string{"120"}}, "rate limited"), nil
	})
	var sessions []*HTTPSession
	for i := range 8 {
		session, err := network.NewSession(InstanceSnapshot{ID: fmt.Sprintf("account-%d", i), BaseURL: "https://site.invalid/"}, nil)
		if err != nil {
			b.Fatal(err)
		}
		sessions = append(sessions, session)
	}
	_, err := sessions[0].Do(context.Background(), cardigann.Request{URL: "https://site.invalid/search"})
	var cooldown *CoolingDownError
	if !errors.As(err, &cooldown) || calls.Load() != 1 {
		b.Fatal("shared cooldown was not established", err)
	}
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		start := time.Now()
		var workers sync.WaitGroup
		results := make(chan error, len(sessions))
		for _, session := range sessions {
			workers.Add(1)
			go func() {
				defer workers.Done()
				_, err := session.Do(context.Background(), cardigann.Request{URL: "https://site.invalid/search"})
				results <- err
			}()
		}
		workers.Wait()
		samples[i] = time.Since(start)
		close(results)
		for err := range results {
			var blocked *CoolingDownError
			if !errors.As(err, &blocked) || !blocked.Until.Equal(cooldown.Until) {
				b.Fatal("account escaped the shared 429 deadline", err)
			}
		}
	}
	b.StopTimer()
	if calls.Load() != 1 || len(network.slots) != 0 || len(network.queue) != 0 {
		b.Fatal("cooldown retried HTTP or leaked admission", calls.Load())
	}
	reportIndexerLatency(b, samples)
	b.ReportMetric(0, "http-calls/op")
	b.ReportMetric(8, "attempts/op")
}

func reportIndexerLatency(b *testing.B, samples []time.Duration) {
	b.Helper()
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	for _, percentile := range []int{50, 95} {
		index := (len(samples)*percentile+99)/100 - 1
		b.ReportMetric(float64(samples[index].Nanoseconds()), fmt.Sprintf("p%d-ns/op", percentile))
	}
}
