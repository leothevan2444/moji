package torrentsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

type searchFunc func(context.Context, indexer.Query, func(context.Context, indexer.SearchProgress) error, func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error)

func (f searchFunc) SearchWithProgress(ctx context.Context, q indexer.Query, p func(context.Context, indexer.SearchProgress) error, a func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
	return f(ctx, q, p, a)
}
func fixture(t *testing.T, source Searcher) (*Service, *storage.Manager) {
	t.Helper()
	ctx := context.Background()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewService(ctx, m, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(); m.Close() })
	return s, m
}
func worker(t *testing.T, s *Service) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("search worker did not stop")
		}
	})
}
func await(t *testing.T, s *Service, id string, predicate func(Snapshot) bool) Snapshot {
	t.Helper()
	events, stop, err := s.Subscribe(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		snapshot, err := s.Get(context.Background(), id, Page{})
		if err != nil {
			t.Fatal(err)
		}
		if predicate(snapshot) {
			return snapshot
		}
		select {
		case <-events:
		case <-timeout.C:
			t.Fatal("search did not reach expected state")
		}
	}
}
func completedSource(count int) Searcher {
	return searchFunc(func(ctx context.Context, q indexer.Query, progress func(context.Context, indexer.SearchProgress) error, accept func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
		if err := progress(ctx, indexer.SearchProgress{Phase: "READY", Sources: []indexer.SearchSource{{ID: "one", Name: "One", Kind: indexer.Builtin}}}); err != nil {
			return indexer.SearchReport{}, err
		}
		if err := progress(ctx, indexer.SearchProgress{Phase: "STARTED", InstanceID: "one"}); err != nil {
			return indexer.SearchReport{}, err
		}
		batch := indexer.Batch{InstanceID: "one", FetchedAt: time.Now().UTC()}
		for i := range count {
			size := int64(i)
			batch.Releases = append(batch.Releases, indexer.Release{Title: fmt.Sprintf("ABCD-%d", i), InstanceID: "one", InstanceName: "One", Kind: indexer.Builtin, Size: &size, Resource: indexer.ResourceRef{ID: fmt.Sprintf("resource-%d", i), Locator: indexer.Locator{DownloadURL: "https://site.example/grab?passkey=synthetic-secret"}}})
		}
		stop, err := accept(ctx, batch)
		if err != nil {
			return indexer.SearchReport{}, err
		}
		report := indexer.SiteReport{InstanceID: "one", Results: count}
		err = progress(ctx, indexer.SearchProgress{Phase: "FINISHED", InstanceID: "one", Report: report, FetchedAt: batch.FetchedAt})
		return indexer.SearchReport{Sites: []indexer.SiteReport{report}, Truncated: stop}, err
	})
}

func TestNamedQueryQualificationPrecedesDurableAcceptance(t *testing.T) {
	service, manager := fixture(t, completedSource(0))
	for _, parameters := range []map[string]string{
		{"Q": "replacement"}, {"Limit": "100000"}, {"Year": "2026", "year": "2025"},
	} {
		result, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "ABCD-123", Parameters: parameters}})
		if !errors.Is(err, ErrInvalid) || result.ID != "" {
			t.Fatal("invalid parameter was durably accepted", result, err)
		}
	}
	var count int
	if err := manager.Runtime.Reader.Get(&count, `SELECT count(*) FROM torrent_search_runs`); err != nil || count != 0 {
		t.Fatal("rejected queries created runs", count, err)
	}
	result, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "ABCD-123", Parameters: map[string]string{"year": "002026"}}})
	if err != nil {
		t.Fatal(err)
	}
	var encoded string
	if err := manager.Runtime.Reader.Get(&encoded, `SELECT query_json FROM torrent_search_runs WHERE id=?`, result.ID); err != nil {
		t.Fatal(err)
	}
	var persisted indexer.Query
	if err := json.Unmarshal([]byte(encoded), &persisted); err != nil || persisted.Parameters["Year"] != "002026" || len(persisted.Parameters) != 1 {
		t.Fatal("accepted query did not freeze canonical field identity", persisted, err)
	}
}

func TestSearchPersistsBeforeExecutionSurvivesCallerAndPagesServerOwnedCandidates(t *testing.T) {
	service, manager := fixture(t, completedSource(130))
	caller, cancel := context.WithCancel(context.Background())
	run, err := service.StartSearch(caller, Input{Query: indexer.Query{Keywords: "ABCD"}})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if run.State != "QUEUED" || run.CreatedAt.IsZero() || run.UpdatedAt.IsZero() {
		t.Fatal("admission did not return persistent intent/timestamps")
	}
	worker(t, service)
	done := await(t, service, run.ID, func(s Snapshot) bool { return terminal(s.State) })
	if done.State != "COMPLETED" || done.Total != 130 || len(done.Candidates) != 100 || len(done.Sites) != 1 || done.Sites[0].State != "COMPLETED" || done.Sites[0].FetchedAt == nil {
		t.Fatal("search lifecycle/first page incorrect", done.State, done.Total)
	}
	page, err := service.Get(context.Background(), run.ID, Page{Offset: 100, Limit: 30})
	if err != nil || len(page.Candidates) != 30 || page.Revision != done.Revision {
		t.Fatal("durable paging incorrect", err)
	}
	sorted, err := service.Get(context.Background(), run.ID, Page{Limit: 1, Sort: "SIZE_DESC"})
	if err != nil || *sorted.Candidates[0].Size != 129 {
		t.Fatal("server sort/page incorrect", err)
	}
	resource, err := service.ResourceID(context.Background(), run.ID, done.Candidates[0].ID)
	if err != nil || resource != "resource-0" {
		t.Fatal("server membership lookup lost source", err)
	}
	other, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "OTHER"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ResourceID(context.Background(), other.ID, done.Candidates[0].ID); !errors.Is(err, ErrExpired) {
		t.Fatal("candidate crossed search ownership")
	}
	public, _ := json.Marshal(done)
	if strings.Contains(string(public), "synthetic-secret") || strings.Contains(string(public), "DownloadURL") {
		t.Fatal("public snapshot exposed private locator")
	}
	var persisted int
	if err = manager.Runtime.Reader.Get(&persisted, `SELECT count(*) FROM torrent_search_candidates WHERE search_id=?`, run.ID); err != nil || persisted != 130 {
		t.Fatal("results were not persisted", err)
	}
}
func TestCancelIsDurableBeforeWorkerStopsAndPreservesExistingResults(t *testing.T) {
	entered := make(chan struct{})
	finish := make(chan struct{})
	defer close(finish)
	source := searchFunc(func(ctx context.Context, q indexer.Query, p func(context.Context, indexer.SearchProgress) error, a func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
		_, err := a(ctx, indexer.Batch{Releases: []indexer.Release{{Title: "ABCD-123", Resource: indexer.ResourceRef{ID: "original"}}}})
		if err != nil {
			return indexer.SearchReport{}, err
		}
		close(entered)
		<-ctx.Done()
		<-finish
		return indexer.SearchReport{}, ctx.Err()
	})
	service, _ := fixture(t, source)
	worker(t, service)
	run, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "ABCD"}})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	cancelled, err := service.CancelSearch(context.Background(), run.ID)
	if err != nil || !cancelled.CancelRequested || cancelled.State != "RUNNING" || cancelled.Total != 1 {
		t.Fatal("cancel was reported terminal before worker exited", err, cancelled.State)
	}
	finish <- struct{}{}
	done := await(t, service, run.ID, func(s Snapshot) bool { return terminal(s.State) })
	if done.State != "CANCELLED" || done.Total != 1 {
		t.Fatal("cancel lost existing result")
	}
}
func TestQueuedCancelAndRestartDoNotReplayWork(t *testing.T) {
	var calls atomic.Int32
	source := searchFunc(func(context.Context, indexer.Query, func(context.Context, indexer.SearchProgress) error, func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
		calls.Add(1)
		return indexer.SearchReport{}, nil
	})
	service, manager := fixture(t, source)
	run, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "ABCD"}})
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.CancelSearch(context.Background(), run.ID)
	if err != nil || cancelled.State != "CANCELLED" {
		t.Fatal(err)
	}
	interrupted, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "OTHER"}})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := service.store.claim(context.Background(), interrupted.ID, time.Now()); err != nil || !claimed {
		t.Fatal(err)
	}
	restarted, err := NewService(context.Background(), manager, source)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if err = restarted.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot, err := restarted.Get(context.Background(), interrupted.ID, Page{})
	if err != nil || snapshot.State != "INTERRUPTED" || snapshot.ErrorCode != "TORRENT_SEARCH_INTERRUPTED" {
		t.Fatal("restart failed to classify abandoned work", err)
	}
	worker(t, restarted)
	if calls.Load() != 0 {
		t.Fatal("cancelled/interrupted work was replayed")
	}
}
func TestSearchBudgetsTruncateExplicitlyAndExpireWithoutRerunning(t *testing.T) {
	service, manager := fixture(t, completedSource(1100))
	worker(t, service)
	run, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "ABCD"}})
	if err != nil {
		t.Fatal(err)
	}
	done := await(t, service, run.ID, func(s Snapshot) bool { return terminal(s.State) })
	if done.State != "PARTIAL" || !done.Truncated || done.Total != 1000 {
		t.Fatal("oversized run masqueraded as complete", done.State, done.Total)
	}
	if err = manager.Runtime.WithWrite(context.Background(), func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE torrent_search_runs SET expires_at='2000-01-01T00:00:00.000000000Z' WHERE id=?`, run.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Get(context.Background(), run.ID, Page{}); !errors.Is(err, ErrExpired) {
		t.Fatal("expired run remained selectable", err)
	}
	if _, err = service.ResourceID(context.Background(), run.ID, done.Candidates[0].ID); !errors.Is(err, ErrExpired) {
		t.Fatal("expired candidate ownership survived", err)
	}
}
func TestRunCapacityAndInvalidQueriesAreRejectedBeforeAdmission(t *testing.T) {
	service, _ := fixture(t, completedSource(0))
	for i := range maxRuns {
		if _, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: fmt.Sprint(i)}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: "excess"}}); !errors.Is(err, ErrBusy) {
		t.Fatal("run capacity ignored", err)
	}
	if _, err := service.StartSearch(context.Background(), Input{Query: indexer.Query{Keywords: " "}}); !errors.Is(err, ErrInvalid) {
		t.Fatal("empty search admitted")
	}
}

func TestSharedSnapshotByteBudgetIncludesSitesAndQueries(t *testing.T) {
	service, manager := fixture(t, completedSource(0))
	ctx := context.Background()
	now := time.Now()
	create := func() string {
		id, err := service.store.create(ctx, Input{Query: indexer.Query{Keywords: "ABCD"}}, now)
		if err != nil {
			t.Fatal(err)
		}
		ok, err := service.store.claim(ctx, id, now)
		if err != nil || !ok {
			t.Fatal(err)
		}
		return id
	}
	first, second := create(), create()
	batch := indexer.Batch{}
	for range 1000 {
		batch.Releases = append(batch.Releases, indexer.Release{Title: strings.Repeat("a", 60000), Resource: indexer.ResourceRef{ID: "private-reference"}})
	}
	if truncated, err := service.store.appendBatch(ctx, first, batch, now); err != nil || truncated {
		t.Fatal("first run should fit the shared budget", err)
	}
	if truncated, err := service.store.appendBatch(ctx, second, batch, now); err != nil || !truncated {
		t.Fatal("second run exceeded shared byte budget without truncating", err)
	}
	var bytes int64
	if err := manager.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { var err error; bytes, err = snapshotBytes(ctx, tx); return err }); err != nil || bytes > maxBytes {
		t.Fatal("stored snapshots exceed budget", bytes, err)
	}
	sources := []indexer.SearchSource{}
	for i := range 200 {
		sources = append(sources, indexer.SearchSource{ID: fmt.Sprint(i), Name: strings.Repeat("n", 512), Kind: indexer.Builtin})
	}
	if err := service.store.progress(ctx, first, indexer.SearchProgress{Phase: "READY", Sources: sources}, now); !errors.Is(err, ErrBusy) {
		t.Fatal("source snapshots escaped shared byte accounting", err)
	}
	var sites int
	if err := manager.Runtime.Reader.Get(&sites, `SELECT count(*) FROM torrent_search_sites`); err != nil || sites != 0 {
		t.Fatal("oversized progress published uncommitted state", err)
	}
	if _, err := service.store.create(ctx, Input{Query: indexer.Query{Keywords: strings.Repeat("q", 100000)}}, now); !errors.Is(err, ErrBusy) {
		t.Fatal("query snapshot escaped shared byte accounting", err)
	}
}
func TestDetailsLinksNeverReturnCredentialQueriesOrUnsafeProtocols(t *testing.T) {
	for _, link := range []string{"magnet:?xt=secret", "https://user:password@site.example/details?id=1", "https://site.example/details?api_key=secret", "https://site.example/details?Authorization=secret", "https://site.example/details?signature=secret", "https://site.example/details?key=secret"} {
		if safeDetails(link) != "" {
			t.Fatal("unsafe detail link exposed")
		}
	}
	if got := safeDetails("https://site.example/details?id=123#private-fragment"); got != "https://site.example/details?id=123" {
		t.Fatal("public detail identity lost or fragment leaked", got)
	}
}

func TestRunTerminationLeavesNoSitesSearching(t *testing.T) {
	for _, scenario := range []string{"cancel", "restart", "truncated"} {
		t.Run(scenario, func(t *testing.T) {
			service, _ := fixture(t, completedSource(0))
			ctx := context.Background()
			now := time.Now()
			id, err := service.store.create(ctx, Input{Query: indexer.Query{Keywords: "ABCD"}}, now)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = service.store.claim(ctx, id, now); err != nil {
				t.Fatal(err)
			}
			if err = service.store.progress(ctx, id, indexer.SearchProgress{Phase: "READY", Sources: []indexer.SearchSource{{ID: "first", Name: "First", Kind: indexer.Builtin}, {ID: "second", Name: "Second", Kind: indexer.Jackett}}}, now); err != nil {
				t.Fatal(err)
			}
			if err = service.store.progress(ctx, id, indexer.SearchProgress{Phase: "STARTED", InstanceID: "first"}, now); err != nil {
				t.Fatal(err)
			}
			expected := "CANCELLED"
			switch scenario {
			case "cancel":
				if err = service.store.cancel(ctx, id, now); err == nil {
					err = service.store.finish(ctx, id, "CANCELLED", "", false, false, now)
				}
			case "restart":
				expected = "INTERRUPTED"
				err = service.store.interrupt(ctx, now)
			case "truncated":
				expected = "SKIPPED"
				if err = service.store.progress(ctx, id, indexer.SearchProgress{Phase: "FINISHED", InstanceID: "first"}, now); err == nil {
					err = service.store.finish(ctx, id, "PARTIAL", "", true, true, now)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := service.Get(ctx, id, Page{})
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Sites) != 2 || snapshot.Sites[1].State != expected || snapshot.Sites[0].State == "RUNNING" {
				t.Fatal("run terminated with misleading per-site progress", snapshot.Sites)
			}
		})
	}
}
func TestSubscriptionCapacityAndShutdownDoNotDependOnClientConsumption(t *testing.T) {
	service, _ := fixture(t, completedSource(0))
	ctx := context.Background()
	run, err := service.StartSearch(ctx, Input{Query: indexer.Query{Keywords: "ABCD"}})
	if err != nil {
		t.Fatal(err)
	}
	channels := []<-chan Event{}
	stops := []func(){}
	for range 128 {
		events, stop, err := service.Subscribe(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		channels = append(channels, events)
		stops = append(stops, stop)
	}
	if _, _, err = service.Subscribe(ctx, run.ID); !errors.Is(err, ErrBusy) {
		t.Fatal("subscriber capacity ignored", err)
	}
	service.CloseSubscriptions()
	for i, ch := range channels {
		<-ch
		if _, ok := <-ch; ok {
			t.Fatal("slow client prevented stream shutdown")
		}
		stops[i]()
	}
	if _, _, err = service.Subscribe(ctx, run.ID); !errors.Is(err, ErrNotRunning) {
		t.Fatal("new subscriptions admitted after stream shutdown", err)
	}
}

func TestTaskSearchAdmissionReusesExistingRunAcrossRefreshPolicyAndConcurrentMounts(t *testing.T) {
	service, _ := fixture(t, completedSource(0))
	ctx := context.Background()
	query := indexer.Query{Keywords: "ABCD-123"}
	first, err := service.StartSearch(ctx, Input{TaskID: "task-one", Query: query})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan string, 8)
	for range 8 {
		go func() {
			snapshot, err := service.StartSearch(ctx, Input{TaskID: "task-one", Query: query})
			if err != nil {
				results <- "error"
				return
			}
			results <- snapshot.ID
		}()
	}
	for range 8 {
		if id := <-results; id != first.ID {
			t.Fatal("concurrent task view duplicated search admission", id)
		}
	}
	query.CachePolicy = indexer.CacheRefresh
	fresh, err := service.StartSearch(ctx, Input{TaskID: "task-one", Query: query})
	if err != nil || fresh.ID == first.ID {
		t.Fatal("explicit task search did not admit fresh work", err)
	}
	query.CachePolicy = indexer.CacheDefault
	again, err := service.StartSearch(ctx, Input{TaskID: "task-one", Query: query})
	if err != nil || again.ID != fresh.ID {
		t.Fatal("reopened view ignored the refreshed run", err)
	}
	other, err := service.StartSearch(ctx, Input{TaskID: "task-two", Query: query})
	if err != nil || other.ID == fresh.ID {
		t.Fatal("search leaked across task ownership", err)
	}
}
