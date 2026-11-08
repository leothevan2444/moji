package indexer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/storage"
)

func TestDiagnosticsKeepSiteReportsContextAcrossSuccessAndRestart(t *testing.T) {
	ctx := WithDiagnosticContext(context.Background(), DiagnosticContext{Trigger: "AUTOMATIC", TaskID: "task-one", Code: "ABCD-123"})
	root := t.TempDir()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if manager != nil {
			manager.Close()
		}
	}()
	records, _ := NewDiagnosticStore(manager)
	instance := InstanceSnapshot{ID: "one", Name: "Synthetic site", Kind: Builtin, Enabled: true, ConfigRevision: "account", Settings: map[string]SettingValue{"password": {String: func() *string { s := "secret-password"; return &s }()}}}
	failed := true
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		if failed {
			return Batch{}, fmt.Errorf("%w: https://private.example/?apikey=secret-key", ErrSolver)
		}
		return Batch{IssueCount: 1, Partial: true}, nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	service.SetDiagnosticRecorder(records)
	for range 2 {
		if _, err = service.Search(ctx, Query{Keywords: "ABCD-123", Parameters: map[string]string{"year": "secret-input"}}); !errors.Is(err, ErrIncomplete) {
			t.Fatal(err)
		}
		failed = false
	}
	service.adapters[Builtin] = fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) { return Batch{}, nil }}
	if _, err = service.Search(ctx, Query{Keywords: "ABCD-123"}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Fetch(ctx, instance, ResourceRef{Version: 1, ID: "opaque", InstanceID: instance.ID, Kind: instance.Kind, ConfigRevision: instance.ConfigRevision}); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	manager.Close()
	manager = nil
	manager, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	records, _ = NewDiagnosticStore(manager)
	stats, statsErr := records.Statistics(ctx, time.Now().UTC(), 72)
	if statsErr != nil || stats.Searches != 3 || stats.Succeeded != 1 || stats.TimedSearches != 3 || stats.Records != 4 {
		t.Fatalf("restart changed derived statistics: %+v %v", stats, statsErr)
	}
	reports, err := diagnosticItems(records, ctx, DiagnosticFilter{InstanceID: "one", Limit: 20, Before: 0})
	if err != nil || len(reports) != 4 {
		t.Fatalf("reports lost on restart: %+v %v", reports, err)
	}
	if reports[3].Report.Code != ErrSolver.Error() || reports[2].Report.IssueCount != 1 || !reports[2].Report.Partial || reports[1].Report.Code != "" || reports[1].Report.Partial || reports[0].Operation != HealthFetch || reports[0].ResourceID != "opaque" {
		t.Fatalf("reports lost outcomes: %+v", reports)
	}
	for _, r := range reports {
		if r.TaskID != "task-one" || r.Code != "ABCD-123" || r.Trigger != "AUTOMATIC" || r.ConfigRevision != "account" || r.Report.CorrelationID == "" {
			t.Fatalf("context lost: %+v", r)
		}
	}
	var messages []string
	if err = manager.Runtime.Reader.Select(&messages, `SELECT message FROM indexer_executions`); err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		for _, secret := range []string{"secret-password", "secret-key", "private.example", "secret-input"} {
			if strings.Contains(message, secret) {
				t.Fatal("secret leaked", secret)
			}
		}
	}
	first, err := diagnosticItems(records, ctx, DiagnosticFilter{InstanceID: "one", Limit: 1, Before: 0})
	if err != nil || len(first) != 1 {
		t.Fatal(err)
	}
	next, err := diagnosticItems(records, ctx, DiagnosticFilter{InstanceID: "one", Limit: 1, Before: mustDiagnosticID(t, first[0].ID)})
	if err != nil || len(next) != 1 || next[0].ID == first[0].ID {
		t.Fatal("cursor repeated report", err)
	}
	other, err := diagnosticItems(records, ctx, DiagnosticFilter{InstanceID: "other", Limit: 20, Before: 0})
	if err != nil || len(other) != 0 {
		t.Fatal("site scope widened", err)
	}
}

func mustDiagnosticID(t *testing.T, id string) int64 {
	t.Helper()
	var n int64
	if _, err := fmt.Sscan(id, &n); err != nil {
		t.Fatal(err)
	}
	return n
}

type failingDiagnostics struct{}

func (failingDiagnostics) Record(context.Context, Diagnostic) error {
	return errors.New("synthetic write failure")
}
func TestDiagnosticFailurePreventsSuccessfulOperationAndTimeoutIsRecorded(t *testing.T) {
	instance := InstanceSnapshot{ID: "one", Kind: Builtin, Enabled: true, ConfigRevision: "account"}
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) { return Batch{}, nil }}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	service.SetDiagnosticRecorder(failingDiagnostics{})
	if _, err := service.Search(context.Background(), Query{}); !errors.Is(err, ErrStateUnavailable) {
		t.Fatal("report persistence failure hidden", err)
	}
	manager, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store, _ := NewDiagnosticStore(manager)
	service.SetDiagnosticRecorder(store)
	adapter.search = func(ctx context.Context, _ InstanceSnapshot, _ Query) (Batch, error) {
		<-ctx.Done()
		return Batch{}, ctx.Err()
	}
	instance.SiteTimeout = time.Millisecond
	service.adapters[Builtin] = adapter
	if _, err := service.Search(context.Background(), Query{}); !errors.Is(err, ErrIncomplete) {
		t.Fatal(err)
	}
	reports, err := diagnosticItems(store, context.Background(), DiagnosticFilter{InstanceID: "one", Limit: 20, Before: 0})
	if err != nil || len(reports) != 1 || reports[0].Report.Code != "INDEXER_TIMEOUT" {
		t.Fatalf("timeout report lost: %+v %v", reports, err)
	}
}

func TestSearchCacheReportsKeepEachTaskContext(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store, _ := NewDiagnosticStore(manager)
	cache, policy, instance := testSearchCache(t)
	calls := 0
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		calls++
		return candidateBatch(instance), nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	service.SetDiagnosticRecorder(store)
	service.SetSearchCache(cache, func() SearchCachePolicy { return policy })
	for _, id := range []string{"task-one", "task-two"} {
		caller := WithDiagnosticContext(ctx, DiagnosticContext{Trigger: "AUTOMATIC", TaskID: id, Code: "ABCD-123"})
		if _, err = service.Search(caller, Query{Keywords: "ABCD-123"}); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := diagnosticItems(store, ctx, DiagnosticFilter{InstanceID: instance.ID, Limit: 20})
	if err != nil || len(reports) != 2 || reports[0].TaskID != "task-two" || reports[1].TaskID != "task-one" || !reports[0].Report.CacheHit || reports[1].Report.CacheHit || calls != 1 {
		t.Fatalf("cache lost caller context: %+v calls=%d err=%v", reports, calls, err)
	}
}

func diagnosticItems(store *DiagnosticStore, ctx context.Context, filter DiagnosticFilter) ([]Diagnostic, error) {
	page, err := store.Diagnostics(ctx, filter)
	return page.Items, err
}

func TestDiagnosticFiltersSearchBeforePaginationAndPreserveMatchingCount(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store, _ := NewDiagnosticStore(manager)
	now := time.Now().UTC()
	for i := 0; i < 6; i++ {
		record := Diagnostic{DiagnosticContext: DiagnosticContext{Code: fmt.Sprintf("ABCD-%d", i)}, InstanceID: "one", Name: "Site one", Operation: HealthSearch, StartedAt: now, FinishedAt: now, Report: SiteReport{CorrelationID: fmt.Sprint(i)}}
		if i%2 == 0 {
			record.Report.Code = ErrSolver.Error()
		}
		if i == 5 {
			record.InstanceID = "two"
			record.Name = "Other site"
			record.Operation = HealthFetch
		}
		if err = store.Record(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	filter := DiagnosticFilter{InstanceID: "one", Outcome: "FAILED", Limit: 2}
	page, err := store.Diagnostics(ctx, filter)
	if err != nil || page.Total != 3 || len(page.Items) != 2 || !page.HasMore || len(page.Sites) != 2 {
		t.Fatalf("filtered count: %+v %v", page, err)
	}
	filter.Before = mustDiagnosticID(t, page.Items[1].ID)
	page, err = store.Diagnostics(ctx, filter)
	if err != nil || page.Total != 3 || len(page.Items) != 1 || page.HasMore {
		t.Fatalf("filtered cursor: %+v %v", page, err)
	}
	for _, term := range []string{"abcd-4", "SITE ONE", "INDEXER_SOLVER_FAILED", "%"} {
		filter = DiagnosticFilter{Query: term, Limit: 20}
		page, err = store.Diagnostics(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string]int{"abcd-4": 1, "SITE ONE": 5, "INDEXER_SOLVER_FAILED": 3, "%": 0}[term]
		if page.Total != expected {
			t.Fatalf("literal search %q: %+v", term, page)
		}
	}
	filter = DiagnosticFilter{Operation: "FETCH", Limit: 20}
	page, err = store.Diagnostics(ctx, filter)
	if err != nil || page.Total != 1 || page.Items[0].InstanceID != "two" {
		t.Fatal("operation filter widened", err)
	}
	since := now.Add(time.Hour)
	filter = DiagnosticFilter{Since: &since, Limit: 20}
	page, err = store.Diagnostics(ctx, filter)
	if err != nil || page.Total != 0 {
		t.Fatal("time filter ignored", err)
	}
}
