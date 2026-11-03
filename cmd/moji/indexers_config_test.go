package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type indexerCatalogFunc func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error)

func (f indexerCatalogFunc) Catalog(ctx context.Context, instance indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
	return f(ctx, instance)
}
func indexerTestStore(t *testing.T) *config.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\nindexers:\n  jackett: {enabled: true, url: 'https://jackett.example', api_key: synthetic}\n  query_order: [jackett/one]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestIndexerRegistryRetainsConfirmedCatalogOnFailureWithoutCrossingAccounts(t *testing.T) {
	store := indexerTestStore(t)
	var calls atomic.Int32
	failed := false
	provider := &runtimeIndexerRegistry{configuration: indexerConfiguration{store: store}, external: indexerCatalogFunc(func(ctx context.Context, instance indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		calls.Add(1)
		if failed {
			return nil, indexer.ErrTransport
		}
		return []indexer.InstanceSnapshot{{ID: "jackett/one", RemoteID: "one", Name: "One"}}, nil
	})}
	snapshot, err := provider.Snapshot(context.Background())
	if err != nil || len(snapshot.Instances) != 1 {
		t.Fatal("catalog", err)
	}
	if !snapshot.JackettCatalog.Known || snapshot.JackettCatalog.Stale || snapshot.JackettCatalog.UpdatedAt == nil {
		t.Fatal("confirmed directory metadata missing")
	}
	confirmedAt := *snapshot.JackettCatalog.UpdatedAt
	revision := snapshot.Instances[0].ConfigRevision
	if _, err = store.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err != nil {
		t.Fatal(err)
	}
	again, err := provider.Snapshot(context.Background())
	if err != nil || calls.Load() != 1 || again.Instances[0].ConfigRevision != revision {
		t.Fatal("unrelated settings invalidated account")
	}
	provider.catalogAt = time.Now().Add(-10 * time.Minute)
	failed = true
	stale, err := provider.Snapshot(context.Background())
	if err != nil || len(stale.Instances) != 1 || len(stale.Diagnostics) != 1 || !stale.Diagnostics[0].Partial {
		t.Fatal("refresh failure discarded confirmed catalog", err)
	}
	if !stale.JackettCatalog.Known || !stale.JackettCatalog.Stale || stale.JackettCatalog.UpdatedAt == nil || !stale.JackettCatalog.UpdatedAt.Equal(confirmedAt) {
		t.Fatal("failed refresh discarded or advanced last confirmed directory time")
	}
	if _, err = provider.Snapshot(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatal("failed catalog had no bounded retry delay", err)
	}
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"api_key": "changed-synthetic"}}}); err != nil {
		t.Fatal(err)
	}
	changed, err := provider.Snapshot(context.Background())
	if err != nil || len(changed.Instances) != 0 || len(changed.Diagnostics) != 1 {
		t.Fatal("old account sites reused after credential change", err)
	}
	if changed.JackettCatalog.Known || changed.JackettCatalog.UpdatedAt != nil {
		t.Fatal("confirmed directory metadata crossed account boundary")
	}
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	disabled, err := provider.Snapshot(context.Background())
	if err != nil || len(disabled.Instances) != 0 || len(disabled.Diagnostics) != 0 || calls.Load() != before {
		t.Fatal("disabled Jackett contacted", err)
	}
}

func TestHomeIndexerSummaryNeverLoadsCatalogAndRetainsConfirmedEmptyOnFailure(t *testing.T) {
	store := indexerTestStore(t)
	var calls atomic.Int32
	failed := false
	provider := &runtimeIndexerRegistry{configuration: indexerConfiguration{store: store}, external: indexerCatalogFunc(func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		calls.Add(1)
		if failed {
			return nil, indexer.ErrTransport
		}
		return []indexer.InstanceSnapshot{}, nil
	})}
	admin := indexeradmin.NewService(store, nil, provider)
	unknown, err := admin.Summary(context.Background())
	if err != nil || calls.Load() != 0 || unknown.Total != nil || unknown.Jackett == nil || unknown.Jackett.Sites != nil {
		t.Fatal("Home loaded directory or invented zero", err)
	}
	if _, err = provider.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	confirmed, err := admin.Summary(context.Background())
	if err != nil || calls.Load() != 1 || confirmed.Total == nil || *confirmed.Total != 0 || confirmed.Jackett.Sites == nil || *confirmed.Jackett.Sites != 0 || confirmed.Jackett.Stale || confirmed.Jackett.UpdatedAt == nil {
		t.Fatal("successful empty directory remained unknown", err)
	}
	failed = true
	provider.catalogAt = time.Time{}
	if _, err = provider.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	stale, err := admin.Summary(context.Background())
	if err != nil || calls.Load() != 2 || stale.Total == nil || *stale.Total != 0 || !stale.Jackett.Stale || stale.Jackett.ErrorCode != indexer.ErrTransport.Error() || !stale.Jackett.UpdatedAt.Equal(*confirmed.Jackett.UpdatedAt) {
		t.Fatal("failed refresh lost last confirmed empty directory", err)
	}
}

func TestConfirmedCatalogSuccessClearsPriorFailureWhenCallerLeaves(t *testing.T) {
	store := indexerTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failed := true
	provider := &runtimeIndexerRegistry{configuration: indexerConfiguration{store: store}, external: indexerCatalogFunc(func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		if failed {
			return nil, indexer.ErrTransport
		}
		cancel()
		return []indexer.InstanceSnapshot{}, nil
	})}
	if _, err := provider.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	failed = false
	provider.attemptAt = time.Now().Add(-time.Minute)
	if _, err := provider.Snapshot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("departed caller did not receive cancellation", err)
	}
	snapshot, err := provider.CachedSnapshot(context.Background())
	if err != nil || !snapshot.JackettCatalog.Known || snapshot.JackettCatalog.Stale || len(snapshot.Diagnostics) != 0 {
		t.Fatal("previous failure attached to newly confirmed directory", err)
	}
}

func TestUnloadedCatalogRetainsCurrentAccountFailureInLocalSummary(t *testing.T) {
	store := indexerTestStore(t)
	bus := indexer.NewEvents()
	defer bus.Close()
	events, stop, err := bus.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	<-events
	provider := &runtimeIndexerRegistry{configuration: indexerConfiguration{store: store}, events: bus, external: indexerCatalogFunc(func(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error) {
		return nil, indexer.ErrAuthentication
	})}
	if _, err = provider.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Kind != indexer.EventCatalog {
			t.Fatal("catalog completion was not published")
		}
	default:
		t.Fatal("catalog failure was not published")
	}
	admin := indexeradmin.NewService(store, nil, provider)
	summary, err := admin.Summary(context.Background())
	if err != nil || summary.Jackett == nil || summary.Jackett.ErrorCode != indexer.ErrAuthentication.Error() || summary.Jackett.Sites != nil {
		t.Fatal("initial account failure hidden as an unloaded directory", err)
	}
	select {
	case <-events:
		t.Fatal("local read published a new directory event")
	default:
	}
	if _, err = store.UpdateJackett("https://jackett.example", "different-synthetic-key", ""); err != nil {
		t.Fatal(err)
	}
	summary, err = admin.Summary(context.Background())
	if err != nil || summary.Jackett == nil || summary.Jackett.ErrorCode != indexer.ErrCatalogNotLoaded.Error() {
		t.Fatal("another account inherited the prior authentication failure", err)
	}
}

func TestIndexerActivationUsesSavedConfigRevisionAndRejectsSettingChanges(t *testing.T) {
	store := indexerTestStore(t)
	enabled := true
	definition := "example"
	boolean := true
	id, _, err := store.UpdateBuiltinInstance(context.Background(), "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition, Enabled: &enabled, Settings: map[string]config.IndexerSettingChange{"checkbox": {Value: &config.IndexerSettingValue{Boolean: &boolean}}}})
	if err != nil {
		t.Fatal(err)
	}
	plan := &cardigann.Plan{Definition: cardigann.Definition{Links: []string{"https://example.invalid/"}, Settings: []cardigann.Setting{{Name: "checkbox", Type: "checkbox"}}}}
	validator := &indexerActivationValidator{configuration: indexerConfiguration{store: store}}
	guard, err := validator.Validate(context.Background(), &indexerdefs.Bundle{Plans: map[string]*cardigann.Plan{"example": plan}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err != nil {
		t.Fatal(err)
	}
	activated := false
	if err = guard.WithCurrent(context.Background(), func() error { activated = true; return nil }); !errors.Is(err, indexerdefs.ErrConflict) || activated {
		t.Fatal("stale rules activated", err)
	}
	missing := &indexerdefs.Bundle{Plans: map[string]*cardigann.Plan{}}
	if _, err = validator.Validate(context.Background(), missing); !errors.Is(err, indexerdefs.ErrIncompatible) {
		t.Fatal("active definition removal accepted", err)
	}
	changed := *plan
	changed.Definition.Settings = []cardigann.Setting{{Name: "checkbox", Type: "text"}}
	if _, err = validator.Validate(context.Background(), &indexerdefs.Bundle{Plans: map[string]*cardigann.Plan{"example": &changed}}); !errors.Is(err, indexerdefs.ErrIncompatible) {
		t.Fatal("incompatible saved setting accepted", err)
	}
	enabled = false
	if _, _, err = store.UpdateBuiltinInstance(context.Background(), id, store.Revision(), config.BuiltinInstancePatch{Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	if _, err = validator.Validate(context.Background(), missing); err != nil {
		t.Fatal("disabled orphan blocked library update", err)
	}
	old := *plan
	newPlan := *plan
	newPlan.Definition.Settings = append(append([]cardigann.Setting{}, plan.Definition.Settings...), cardigann.Setting{Name: "passkey", Type: "password"})
	instance := store.Config().Indexers.Builtin.Instances[id]
	if err = indexeradmin.ValidateSettings(&newPlan, &old, instance); err == nil {
		t.Fatal("new credential setting activated without review")
	}
}

func TestSavedNetworkChangesAffectNewRegistrySnapshotsAndKeepOldWorkFrozen(t *testing.T) {
	store := indexerTestStore(t)
	ctx := context.Background()
	proxy := "http://synthetic-user:old-secret@proxy.example:8080"
	if _, _, err := store.SaveIndexerNetworkProfile(ctx, store.Revision(), "private", true, config.IndexerNetworkProfilePatch{ProxyURL: &proxy}); err != nil {
		t.Fatal(err)
	}
	definition, profile := "example", "private"
	id, _, err := store.UpdateBuiltinInstance(ctx, "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition, NetworkProfile: &profile})
	if err != nil {
		t.Fatal(err)
	}
	registry := &runtimeIndexerRegistry{configuration: indexerConfiguration{store: store}}
	old, err := registry.CachedSnapshot(ctx)
	if err != nil || len(old.Instances) == 0 {
		t.Fatal(err)
	}
	var frozen indexer.InstanceSnapshot
	for _, instance := range old.Instances {
		if instance.ID == id {
			frozen = instance
		}
	}
	proxy = "socks5h://synthetic-user:new-secret@proxy.example:1080"
	if _, _, err = store.SaveIndexerNetworkProfile(ctx, store.Revision(), "private", false, config.IndexerNetworkProfilePatch{ProxyURL: &proxy}); err != nil {
		t.Fatal(err)
	}
	request, site := 45, 180
	if _, _, err = store.UpdateIndexerNetworkTimeouts(ctx, store.Revision(), config.IndexerNetworkTimeoutPatch{RequestTimeoutSeconds: &request, SiteTimeoutSeconds: &site}); err != nil {
		t.Fatal(err)
	}
	next, err := registry.CachedSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var current indexer.InstanceSnapshot
	for _, instance := range next.Instances {
		if instance.ID == id {
			current = instance
		}
	}
	if current.Network.ProxyURL != proxy || current.Network.Revision == frozen.Network.Revision || current.ConfigRevision == frozen.ConfigRevision || current.RequestTimeout != 45*time.Second || current.SiteTimeout != 180*time.Second || frozen.RequestTimeout != 30*time.Second || frozen.Network.ProxyURL == proxy {
		t.Fatal("network hot edit reused an old session namespace or changed frozen work")
	}
}
