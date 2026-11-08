package indexer

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"testing"
	"time"
)

func executionRecord(site, correlation string, at time.Time, failed bool) Diagnostic {
	r := Diagnostic{Executed: true, ElapsedKnown: true, InstanceID: site, Name: site, Operation: HealthSearch, StartedAt: at, FinishedAt: at.Add(time.Second), Report: SiteReport{CorrelationID: correlation, Elapsed: time.Second}}
	if failed {
		r.Report.Code = ErrSolver.Error()
		r.Report.Partial = true
	}
	return r
}
func executionStore(t *testing.T) (*DiagnosticStore, *storage.Manager) {
	t.Helper()
	m, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	s, _ := NewDiagnosticStore(m)
	return s, m
}

func TestUnifiedExecutionsDriveHistoryStatisticsAndRecentState(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 12; i++ {
		r := executionRecord("one", fmt.Sprint(i), now.Add(time.Duration(i-12)*time.Minute), i == 0 || i == 10)
		for range 2 {
			if err := s.Record(ctx, r); err != nil {
				t.Fatal(err)
			}
		}
	}
	fetch := executionRecord("one", "fetch", now.Add(-time.Second), true)
	fetch.Operation = HealthFetch
	if err := s.Record(ctx, fetch); err != nil {
		t.Fatal(err)
	}
	// Cached and cancelled attempts remain inspectable but never count as upstream executions.
	cached := executionRecord("one", "cache", now, false)
	cached.Report.CacheHit = true
	cancelled := executionRecord("one", "cancel", now, false)
	cancelled.Report.Code = "INDEXER_CANCELLED"
	skipped := executionRecord("one", "skip", now, false)
	skipped.Executed = false
	for _, r := range []Diagnostic{cached, cancelled, skipped} {
		if err := s.Record(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := s.Statistics(ctx, now, 72)
	if err != nil || stats.Searches != 12 || stats.Succeeded != 10 || stats.TimedSearches != 12 || stats.TotalElapsedMs != 12000 || stats.Records != 16 {
		t.Fatalf("stats %+v %v", stats, err)
	}
	states, err := s.RecentExecutions(ctx, []string{"one"})
	if err != nil || states["one"].Count != 10 || states["one"].Failures != 2 || !states["one"].LatestFailed {
		t.Fatalf("recent %+v %v", states, err)
	}
	page, err := s.Diagnostics(ctx, DiagnosticFilter{InstanceID: "one", Outcome: "FAILED", Limit: 100})
	if err != nil || page.Total != 3 {
		t.Fatalf("history %+v %v", page, err)
	}
	// A single authority: deleting one row changes all three views, with no replay/refresh store.
	if err = s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`DELETE FROM indexer_executions WHERE correlation_id='10'`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	stats, _ = s.Statistics(ctx, now, 72)
	states, _ = s.RecentExecutions(ctx, []string{"one"})
	page, _ = s.Diagnostics(ctx, DiagnosticFilter{Outcome: "FAILED", Limit: 100})
	if stats.Searches != 11 || stats.Succeeded != 10 || states["one"].Failures != 1 || page.Total != 2 {
		t.Fatal("views disagree after a record changes")
	}
}

func TestRetentionKeepsFullWindowAndProtectsEachConfiguredSite(t *testing.T) {
	s, m := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	r := executionRecord("busy", "seed", now.Add(-time.Hour), false)
	if err := s.Record(ctx, r); err != nil {
		t.Fatal(err)
	}
	// More than the former global budget, without 4100 test transactions.
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`WITH RECURSIVE seq(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM seq WHERE x<4100) INSERT INTO indexer_executions(message) SELECT json_set(message,'$.Report.CorrelationID','busy-'||x) FROM indexer_executions,seq WHERE correlation_id='seed'`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, site := range []string{"quiet", "removed"} {
		for i := 0; i < 12; i++ {
			if err := s.Record(ctx, executionRecord(site, fmt.Sprintf("%s-%d", site, i), now.Add(-20*24*time.Hour).Add(time.Duration(i)*time.Minute), i == 11)); err != nil {
				t.Fatal(err)
			}
		}
	}
	staleCached := executionRecord("quiet", "old-cache", now.Add(-16*24*time.Hour), false)
	staleCached.Report.CacheHit = true
	if err := s.Record(ctx, staleCached); err != nil {
		t.Fatal(err)
	}
	// Generic diagnostics cleanup must not touch the authoritative execution table.
	if _, err := m.PruneRuntime(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Prune(ctx, []string{"busy", "quiet"}, false, now); err != nil {
		t.Fatal(err)
	}
	stats, err := s.Statistics(ctx, now, 336)
	if err != nil || stats.Searches != 4101 || stats.Records != 4111 {
		t.Fatalf("retention corrupted window %+v %v", stats, err)
	}
	states, _ := s.RecentExecutions(ctx, []string{"quiet", "removed"})
	if states["quiet"].Count != 10 || !states["quiet"].LatestFailed || states["removed"].Count != 0 {
		t.Fatalf("site protection %+v", states)
	}
	page, _ := s.Diagnostics(ctx, DiagnosticFilter{InstanceID: "quiet", Limit: 100})
	if page.Total != 10 {
		t.Fatal("protected old records not visible in history")
	}
	since := now.Add(-14 * 24 * time.Hour)
	page, _ = s.Diagnostics(ctx, DiagnosticFilter{InstanceID: "quiet", Since: &since, Limit: 100})
	if page.Total != 0 {
		t.Fatal("history period ignored")
	}
}

func TestSelectedStatisticsWindowsAndMissingTiming(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, hours := range []int{350, 200, 48, 1} {
		r := executionRecord("one", fmt.Sprint(hours), now.Add(-time.Duration(hours)*time.Hour), false)
		if hours == 1 {
			r.ElapsedKnown = false
		}
		if err := s.Record(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	for hours, want := range map[int]int{24: 1, 72: 2, 336: 3} {
		stats, err := s.Statistics(ctx, now, hours)
		if err != nil || stats.Searches != want || stats.TimedSearches != want-1 || !stats.WindowStartedAt.Equal(now.Add(-time.Duration(hours)*time.Hour)) || !stats.WindowEndedAt.Equal(now) {
			t.Fatalf("period %d %+v %v", hours, stats, err)
		}
	}
	if _, err := s.Statistics(ctx, now, 48); err != ErrInvalidQuery {
		t.Fatal(err)
	}
}

func TestExecutionCountersExcludeCacheHitsAndCandidateInspectionTime(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	cache, policy, instance := testSearchCache(t)
	calls := 0
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		calls++
		return candidateBatch(instance), nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	service.SetDiagnosticRecorder(s)
	service.SetSearchCache(cache, func() SearchCachePolicy { return policy })
	for range 2 {
		_, err := service.SearchSequential(ctx, Query{Keywords: "ABCD-123"}, func(context.Context, Batch) (bool, error) { time.Sleep(60 * time.Millisecond); return true, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	stats, err := s.Statistics(ctx, time.Now().UTC(), 72)
	if err != nil || calls != 1 || stats.Searches != 1 || stats.Succeeded != 1 || stats.TotalElapsedMs >= 60 || stats.Records != 2 {
		t.Fatalf("cache/inspection counted %+v calls=%d %v", stats, calls, err)
	}
}

func TestConcurrentRecordsDeduplicateAndLateResultsKeepNewestStatus(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	errs := make(chan error, 64)
	for i := 0; i < 32; i++ {
		for range 2 {
			go func(i int) {
				errs <- s.Record(ctx, executionRecord("one", fmt.Sprint(i), now.Add(time.Duration(i-33)*time.Second), i%2 == 1))
			}(i)
		}
	}
	for range 64 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	stats, err := s.Statistics(ctx, now, 24)
	states, e := s.RecentExecutions(ctx, []string{"one"})
	if err != nil || e != nil || stats.Searches != 32 || stats.Succeeded != 16 || states["one"].Failures != 5 || !states["one"].LatestFailed {
		t.Fatalf("concurrent/late writes %+v %+v %v %v", stats, states, err, e)
	}
}

func TestUnconfirmedJackettProtectsOnlyItsLastTenExecutions(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, site := range []string{"jackett/quiet", "removed-builtin"} {
		for i := 0; i < 12; i++ {
			if err := s.Record(ctx, executionRecord(site, fmt.Sprint(i), now.Add(-30*24*time.Hour).Add(time.Duration(i)*time.Minute), true)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.Prune(ctx, nil, true, now); err != nil {
		t.Fatal(err)
	}
	page, err := s.Diagnostics(ctx, DiagnosticFilter{Limit: 100})
	if err != nil || page.Total != 10 {
		t.Fatalf("unknown catalog cleanup %+v %v", page, err)
	}
	if err = s.Prune(ctx, nil, false, now); err != nil {
		t.Fatal(err)
	}
	page, err = s.Diagnostics(ctx, DiagnosticFilter{Limit: 100})
	if err != nil || page.Total != 0 {
		t.Fatal("removed sites kept forever", page.Total, err)
	}
}

func TestConcurrentJoinedSearchCallersProduceOneCountedRecord(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	cache, policy, instance := testSearchCache(t)
	entered, release := make(chan struct{}), make(chan struct{})
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		close(entered)
		<-release
		return candidateBatch(instance), nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	service.SetDiagnosticRecorder(s)
	service.SetSearchCache(cache, func() SearchCachePolicy { return policy })
	errs := make(chan error, 2)
	for range 2 {
		go func() { _, err := service.Search(ctx, Query{Keywords: "ABCD-123"}); errs <- err }()
	}
	<-entered
	// Wait for both callers to join the actual flight, rather than becoming a later cache hit.
	deadline := time.Now().Add(time.Second)
	for {
		cache.mu.Lock()
		waiters := 0
		for _, slot := range cache.slots {
			for _, flight := range slot.flights {
				waiters += flight.waiters
			}
		}
		cache.mu.Unlock()
		if waiters == 2 {
			break
		}
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("callers did not join")
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	stats, err := s.Statistics(ctx, time.Now().UTC(), 24)
	page, e := s.Diagnostics(ctx, DiagnosticFilter{Limit: 20})
	if err != nil || e != nil || stats.Searches != 1 || page.Total != 1 {
		t.Fatalf("shared flight counted twice: %+v %+v %v %v", stats, page, err, e)
	}
}

func TestStatisticsRespectNanosecondWindowBoundaries(t *testing.T) {
	s, _ := executionStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second).Add(123 * time.Nanosecond)
	start := now.Add(-24 * time.Hour)
	for i, finished := range []time.Time{start.Add(-time.Nanosecond), start, now, now.Add(time.Nanosecond)} {
		r := executionRecord("one", fmt.Sprint(i), finished.Add(-time.Second), false)
		if err := s.Record(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := s.Statistics(ctx, now, 24)
	if err != nil || stats.Searches != 2 {
		t.Fatalf("boundary ordering %+v %v", stats, err)
	}
}
