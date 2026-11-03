package indexeradmin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

type currentBuiltinHealthRegistry struct{ store *config.Store }

func (r currentBuiltinHealthRegistry) CachedSnapshot(context.Context) (indexer.Registry, error) {
	cfg, revision := r.store.Snapshot()
	out := indexer.Registry{Revision: revision}
	for id, source := range cfg.Indexers.Builtin.Instances {
		out.Instances = append(out.Instances, BuiltinSnapshot(cfg, id, source))
	}
	return out, nil
}
func (r currentBuiltinHealthRegistry) Refresh(ctx context.Context) (indexer.Registry, error) {
	return r.CachedSnapshot(ctx)
}

func TestBuiltinHomeHealthUsesCurrentAccountAndActiveRuleNamespace(t *testing.T) {
	ctx := context.Background()
	service, store, defs, _ := managementFixture(t)
	if _, e := store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); e != nil {
		t.Fatal(e)
	}
	service.registry = currentBuiltinHealthRegistry{store: store}
	manager, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer manager.Close()
	health, e := indexer.NewHealthStore(manager)
	if e != nil {
		t.Fatal(e)
	}
	service.SetHealthReader(health)
	definition, enabled, password := "example", true, "synthetic-initial"
	save, e := service.SaveBuiltin(ctx, "", store.Revision(), BuiltinPatch{Definition: &definition, Enabled: &enabled, Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &password}}}})
	if e != nil {
		t.Fatal(e)
	}
	cfg, _ := store.Snapshot()
	frozen := BuiltinSnapshot(cfg, save.Instance.ID, cfg.Indexers.Builtin.Instances[save.Instance.ID])
	frozen.BundleRevision = defs.bundle.Manifest.Revision
	correlation := uuid.NewString()
	if _, e = health.Observe(ctx, frozen, indexer.HealthSearch, time.Now().Add(-time.Second), indexer.ErrAuthentication, correlation); e != nil {
		t.Fatal(e)
	}
	summary, e := service.Summary(ctx)
	if e != nil || summary.Attention != 1 || len(summary.Issues) != 1 || summary.Issues[0].Observation.CorrelationID != correlation || summary.BuiltinUnverified != 0 {
		t.Fatal("current auth failure missing", e)
	}
	changed := "synthetic-other-account"
	if _, e = service.SaveBuiltin(ctx, save.Instance.ID, store.Revision(), BuiltinPatch{Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &changed}}}}); e != nil {
		t.Fatal(e)
	}
	summary, e = service.Summary(ctx)
	if e != nil || summary.Attention != 0 || len(summary.Issues) != 0 || summary.BuiltinUnverified != 1 {
		t.Fatal("new account inherited previous auth failure", e)
	}
	if _, e = service.SaveBuiltin(ctx, save.Instance.ID, store.Revision(), BuiltinPatch{Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &password}}}}); e != nil {
		t.Fatal(e)
	}
	summary, e = service.Summary(ctx)
	if e != nil || len(summary.Issues) != 1 {
		t.Fatal("original namespace not preserved", e)
	}
	defs.bundle.Manifest.Revision = strings.Repeat("b", 40)
	summary, e = service.Summary(ctx)
	if e != nil || summary.Attention != 0 || summary.BuiltinUnverified != 1 || len(summary.Issues) != 0 {
		t.Fatal("active rule revision reused old auth failure", e)
	}
	// Observations remain in the old namespace; reading the new rule did not erase
	// them or fabricate new verification.
	states, e := health.Snapshot(ctx, []indexer.HealthKey{indexer.HealthScope(frozen)})
	if e != nil || states[indexer.HealthScope(frozen)].Search == nil || states[indexer.HealthScope(frozen)].Search.CorrelationID != correlation {
		t.Fatal("namespace switch destroyed durable observation", e)
	}
}
