package indexeradmin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestSummaryCountsCurrentEnabledSites(t *testing.T) {
	page := InstancePage{JackettEnabled: true, JackettConfigured: true, Items: []Instance{{ID: "builtin", Kind: indexer.Builtin, Enabled: true}, {ID: "off", Kind: indexer.Builtin}, {ID: "jackett/one", Kind: indexer.Jackett, Enabled: true}}}
	if summarize(page).Total != 2 {
		t.Fatal("current enabled sites were not counted")
	}
	page.JackettCatalog.Known = true
	if total := summarize(page).Total; total != 2 {
		t.Fatal("enabled count lost")
	}
	page.Items = nil
	if total := summarize(page).Total; total != 0 {
		t.Fatal("confirmed empty became unknown")
	}
	page.JackettCatalog.Known = false
	page.JackettEnabled = false
	if total := summarize(page).Total; total != 0 {
		t.Fatal("disabled gateway made total unknown")
	}
}
func TestSummaryUsesLocalSnapshotWithoutRefreshingJackett(t *testing.T) {
	service, _, _, registry := managementFixture(t)
	result, err := service.Summary(context.Background(), 72)
	if err != nil || registry.refreshes != 0 || result.Total != 1 {
		t.Fatal("Home refreshed remote state or fabricated count", err)
	}
}

func TestManagementHistoryAndSummaryUseSameExecution(t *testing.T) {
	ctx := context.Background()
	service, cfg, _, _ := managementFixture(t)
	if _, err := cfg.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	service.registry = currentBuiltinHealthRegistry{store: cfg}
	definition, enabled := "example", true
	saved, err := service.SaveBuiltin(ctx, "", cfg.Revision(), BuiltinPatch{Definition: &definition, Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	history, _ := indexer.NewDiagnosticStore(manager)
	service.SetStatisticsReader(history)
	at := time.Now().UTC().Add(-time.Hour)
	if err = history.Record(ctx, indexer.Diagnostic{Executed: true, InstanceID: saved.Instance.ID, Name: "Example", Operation: indexer.HealthSearch, StartedAt: at, FinishedAt: at.Add(time.Second), Report: indexer.SiteReport{CorrelationID: "previous-search", Code: indexer.ErrSolver.Error(), Partial: true}}); err != nil {
		t.Fatal(err)
	}
	page, err := service.Instances(ctx)
	if err != nil || len(page.Items) != 1 || page.Items[0].Recent.Count != 1 || !page.Items[0].Recent.LatestFailed {
		t.Fatalf("existing failure invisible to station UI: %+v %v", page, err)
	}
	summary, err := service.Summary(ctx, 72)
	if err != nil || summary.Total != 1 || summary.Hours != 72 || summary.Statistics == nil || summary.Statistics.Searches != 1 {
		t.Fatalf("current sites tied to missing samples: %+v %v", summary, err)
	}
}
func TestInvalidStatisticsPeriodFailsBeforeReadingSources(t *testing.T) {
	service := &Service{}
	if _, err := service.Summary(context.Background(), 48); !errors.Is(err, indexer.ErrInvalidQuery) {
		t.Fatal("invalid period was accepted", err)
	}
}
