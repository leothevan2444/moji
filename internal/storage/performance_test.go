package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestPerformanceUsesLiveOwnerAndDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if other, err := OpenManager(ctx, DefaultManagerOptions(root)); !errors.Is(err, ErrDirectoryInUse) {
		if other != nil {
			other.Close()
		}
		t.Fatal("owner lock not held", err)
	}
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	baseline := m.PerformanceSnapshot()
	if len(baseline.Databases) != 3 || baseline.SampledAt == "" || baseline.StartedAt == "" {
		t.Fatal("missing initial sample", baseline)
	}
	before := m.Core.Metrics().Writes
	if err = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('performance','MANUAL','','DOWNLOADING','RUNNING','now','now')`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	after := m.PerformanceSnapshot()
	core := after.Databases[0]
	if core.Role != "core" || core.WriteAttempts != int64(before+1) || core.WritesPerSecond <= 0 || core.DatabaseBytes <= 0 || core.Reader.Limit != 4 || core.Writer.Limit != 1 {
		t.Fatalf("incorrect live metrics: %+v", core)
	}
	if m.Core.Metrics().Writes != before+1 {
		t.Fatal("sampling wrote to core")
	}
	if baseline.Databases[0].WriteAttempts == core.WriteAttempts {
		t.Fatal("prior immutable snapshot changed")
	}
	core.Reader.Limit = 999
	core.Role = "mutated"
	if m.PerformanceSnapshot().Databases[0].Reader.Limit == 999 || m.PerformanceSnapshot().Databases[0].Role == "mutated" {
		t.Fatal("caller could mutate cached metrics")
	}
	failed, cancel := context.WithCancel(ctx)
	cancel()
	if err = m.SamplePerformance(failed); err == nil {
		t.Fatal("canceled sample succeeded")
	}
	stale := m.PerformanceSnapshot()
	if stale.LastError == "" || stale.SampledAt != after.SampledAt {
		t.Fatal("failed sample disguised as fresh data")
	}
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	if m.PerformanceSnapshot().LastError != "" {
		t.Fatal("sample error did not recover")
	}
}

func TestPerformanceMonitorStopsBeforeDatabaseClosure(t *testing.T) {
	m, err := OpenManager(context.Background(), DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); m.MonitorPerformance(ctx) }()
	deadline := time.After(2 * time.Second)
	for m.PerformanceSnapshot().SampledAt == "" {
		select {
		case <-deadline:
			cancel()
			t.Fatal("monitor never sampled")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitor did not stop")
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPerformanceCountsRejectedAndCanceledWriterAdmission(t *testing.T) {
	d, err := Open(context.Background(), Options{Path: t.TempDir() + "/test.db", WriteQueue: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- d.WithWrite(context.Background(), func(tx *sqlx.Tx) error { close(started); <-release; return nil })
	}()
	<-started
	// A canceled caller waiting for the occupied writer is distinct from an
	// execution error; saturation rejects before entering SQLite at all.
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err = d.WithWrite(canceled, func(*sqlx.Tx) error { return errors.New("must not run") }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	d.admission <- struct{}{}
	if err = d.WithWrite(context.Background(), func(*sqlx.Tx) error { return nil }); !errors.Is(err, ErrWriteQueueFull) {
		t.Fatal(err)
	}
	<-d.admission
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	s := d.Metrics()
	if s.QueueRejected != 1 || s.QueueCanceled != 1 || s.WriteErrors != 0 || s.Writes != 1 || s.Queued != 0 {
		t.Fatalf("wrong writer outcome metrics: %+v", s)
	}
}

func TestPerformanceSubscriptionsReplayLatestCoalesceAndRelease(t *testing.T) {
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	slowCtx, stopSlow := context.WithCancel(ctx)
	slow := m.SubscribePerformance(slowCtx)
	fastCtx, stopFast := context.WithCancel(ctx)
	defer stopFast()
	fast := m.SubscribePerformance(fastCtx)
	first := <-fast
	first.Databases[0].Role = "mutated"
	if m.PerformanceSnapshot().Databases[0].Role != "core" {
		t.Fatal("subscriber changed the cached state")
	}
	for i := 0; i < 4; i++ {
		if err = m.SamplePerformance(ctx); err != nil {
			t.Fatal(err)
		}
	}
	latest := m.PerformanceSnapshot().SampledAt
	if (<-slow).SampledAt != latest || (<-fast).SampledAt != latest {
		t.Fatal("slow subscribers did not retain only the latest snapshot")
	}
	stopSlow()
	select {
	case _, ok := <-slow:
		if ok {
			t.Fatal("canceled subscription still open")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled subscription leaked")
	}
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	if (<-fast).SampledAt == "" {
		t.Fatal("canceling another subscriber broke this stream")
	}
	m.ClosePerformanceSubscriptions()
	if _, ok := <-fast; ok {
		t.Fatal("owner close did not close the stream")
	}
	if _, ok := <-m.SubscribePerformance(ctx); ok {
		t.Fatal("closed owner accepted a new subscriber")
	}
	m.performance.mu.RLock()
	count := len(m.performance.subscribers)
	m.performance.mu.RUnlock()
	if count != 0 {
		t.Fatal("subscribers retained after owner close")
	}
}
