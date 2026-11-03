package indexer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These explicit benchmarks measure acquisition coordination, validation and
// immutable copies. The callback supplies synthetic bytes; it performs no HTTP.
func BenchmarkArtifactCache(b *testing.B) {
	for _, mode := range []string{"Hit1MiB", "Cold1MiB", "Pressure1MiB", "Shared8"} {
		b.Run(mode, func(b *testing.B) {
			cache, err := NewArtifactCache(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			defer cache.Close()
			instance := InstanceSnapshot{ID: "synthetic", Kind: Builtin, ConfigRevision: "account", BundleRevision: "rules", SiteTimeout: time.Minute}
			ref := ResourceRef{Version: 1, InstanceID: instance.ID, Kind: Builtin, ConfigRevision: instance.ConfigRevision, Locator: Locator{DownloadURL: "https://site.invalid/one.torrent"}}
			payload := bytes.Repeat([]byte("x"), 1<<20)
			data := append([]byte(fmt.Sprintf("d7:comment%d:", len(payload))), payload...)
			data = append(data, artifactBytes("")[1:]...)
			var calls atomic.Int64
			run := func(context.Context) (Artifact, error) {
				calls.Add(1)
				return Artifact{TorrentBytes: data}, nil
			}
			if mode == "Hit1MiB" {
				if _, err := cache.Fetch(context.Background(), instance, ref, run); err != nil {
					b.Fatal(err)
				}
			}
			before := calls.Load()
			samples := make([]time.Duration, b.N)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				if mode == "Cold1MiB" {
					cache.Clear()
				} else if mode == "Pressure1MiB" {
					ref.Locator.DownloadURL = fmt.Sprintf("https://site.invalid/%d.torrent", i)
					// Distinct original bytes exercise the payload budget even though
					// every torrent has the same info dictionary and hash.
					copy(data[len("d7:comment1048576:"):], fmt.Sprintf("%016x", i))
				} else if mode == "Shared8" {
					ref.Locator.DownloadURL = fmt.Sprintf("https://site.invalid/shared-%d.torrent", i)
				}
				start := time.Now()
				if mode == "Shared8" {
					benchmarkSharedArtifact(b, cache, instance, ref, data, &calls)
				} else {
					result, err := cache.Fetch(context.Background(), instance, ref, run)
					if err != nil || !bytes.Equal(result.TorrentBytes, data) || result.infoHash == "" {
						b.Fatal("artifact was lost or changed", err)
					}
					// Mutation of a returned copy must not corrupt the cached blob.
					result.TorrentBytes[0] = 'x'
				}
				samples[i] = time.Since(start)
			}
			b.StopTimer()
			cache.mu.Lock()
			used, entries, flights, waiters := cache.used, len(cache.entries), len(cache.flights), cache.waiters
			cache.mu.Unlock()
			if used > artifactMaxBytes || entries > artifactMaxEntries || flights != 0 || waiters != 0 {
				b.Fatal("artifact budget or worker ownership violated", used, entries, flights, waiters)
			}
			invocations := calls.Load() - before
			if mode == "Hit1MiB" && invocations != 0 || mode != "Hit1MiB" && invocations != int64(b.N) {
				b.Fatal("artifact acquisition was repeated", invocations)
			}
			reportIndexerLatency(b, samples)
			b.ReportMetric(float64(used), "cache-B")
			b.ReportMetric(float64(entries), "cache-entries")
			b.ReportMetric(float64(invocations)/float64(b.N), "source-calls/op")
		})
	}
}

func benchmarkSharedArtifact(b *testing.B, cache *ArtifactCache, instance InstanceSnapshot, ref ResourceRef, data []byte, calls *atomic.Int64) {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gate := make(chan struct{})
	var group sync.WaitGroup
	results := make(chan error, 8)
	for i := range 8 {
		current := ref
		current.ID = fmt.Sprintf("opaque-%d", i)
		group.Go(func() {
			result, err := cache.Fetch(ctx, instance, current, func(worker context.Context) (Artifact, error) {
				calls.Add(1)
				select {
				case <-gate:
					return Artifact{TorrentBytes: data}, nil
				case <-worker.Done():
					return Artifact{}, worker.Err()
				}
			})
			if err == nil && (!bytes.Equal(result.TorrentBytes, data) || result.Resource.ID != current.ID || result.infoHash == "") {
				err = fmt.Errorf("shared artifact lost bytes, caller identity or hash")
			}
			results <- err
		})
	}
	// Hold the one admitted acquisition until all eight callers have joined.
	tick := time.NewTicker(50 * time.Microsecond)
	defer tick.Stop()
	for {
		cache.mu.Lock()
		joined := cache.waiters == 8
		cache.mu.Unlock()
		if joined {
			break
		}
		select {
		case <-ctx.Done():
			cancel()
			group.Wait()
			b.Fatal("shared acquisition did not admit all waiters")
		case <-tick.C:
		}
	}
	close(gate)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArtifactCancellation64(b *testing.B) {
	cache, err := NewArtifactCache(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	instance := InstanceSnapshot{ID: "synthetic", Kind: Builtin, SiteTimeout: time.Minute}
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()
	for iteration := range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		started := make(chan struct{}, artifactMaxOperations)
		results := make(chan error, artifactMaxOperations)
		var callers sync.WaitGroup
		for i := range artifactMaxOperations {
			ref := ResourceRef{Version: 1, InstanceID: instance.ID, Kind: Builtin, Locator: Locator{DownloadURL: fmt.Sprintf("https://site.invalid/cancel-%d.torrent", i)}}
			callers.Go(func() {
				_, err := cache.Fetch(ctx, instance, ref, func(worker context.Context) (Artifact, error) {
					started <- struct{}{}
					<-worker.Done()
					return Artifact{}, worker.Err()
				})
				results <- err
			})
		}
		for range artifactMaxOperations {
			select {
			case <-started:
			case <-ctx.Done():
				cancel()
				callers.Wait()
				b.Fatal("cancellation workload did not reach capacity")
			}
		}
		// Report cancellation through both caller and acquisition-worker join.
		// Admission/setup is deliberately excluded from the measured interval.
		b.StartTimer()
		start := time.Now()
		cancel()
		callers.Wait()
		cache.wg.Wait()
		samples[iteration] = time.Since(start)
		b.StopTimer()
		close(results)
		for err := range results {
			if !errors.Is(err, context.Canceled) {
				b.Fatal("cancellation lost its cause", err)
			}
		}
		cache.mu.Lock()
		empty := cache.waiters == 0 && len(cache.flights) == 0 && len(cache.entries) == 0
		cache.mu.Unlock()
		if !empty {
			b.Fatal("cancellation retained workers, slots or failure data")
		}
	}
	reportIndexerLatency(b, samples)
	b.ReportMetric(artifactMaxOperations, "workers/op")
}
