package indexeradmin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

func TestDefinitionCatalogFullV11CorpusPagination(t *testing.T) {
	path := os.Getenv("MOJI_INDEXER_CORPUS_ARCHIVE")
	if path == "" {
		t.Skip("opt-in fixed community corpus")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "b266646ab2662c260ac6b0fad5f99794b200784f3ac865b521820e8e6814a472" {
		t.Fatal("corpus archive differs from pinned snapshot")
	}
	bundle, err := indexerdefs.StageArchive(context.Background(), t.TempDir(), "5f5db4e58d06be18619c3a858b9a91ade6f341ca", bytes.NewReader(data), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, &testDefinitions{bundle: bundle}, nil)
	seen := map[string]bool{}
	for offset := 0; offset < len(bundle.Plans); offset += 100 {
		page, err := service.Definitions(context.Background(), DefinitionFilter{Offset: offset, Limit: 100})
		if err != nil || page.Total != len(bundle.Plans) || page.Revision != bundle.Manifest.Revision {
			t.Fatal("corpus catalog unavailable", err)
		}
		for _, definition := range page.Items {
			if seen[definition.ID] {
				t.Fatal("duplicate page result")
			}
			seen[definition.ID] = true
		}
	}
	if len(seen) != 579 || len(seen) != len(bundle.Plans) {
		t.Fatal("catalog did not expose complete fixed corpus")
	}
}

type testDefinitions struct {
	bundle      *indexerdefs.Bundle
	require     func() error
	unavailable bool
}

func (d *testDefinitions) Acquire(context.Context, string) (*indexerdefs.Lease, error) {
	if d.unavailable {
		return nil, indexerdefs.ErrNotInstalled
	}
	return &indexerdefs.Lease{Bundle: d.bundle}, nil
}
func (d *testDefinitions) RequireActive(context.Context, string) error {
	if d.require != nil {
		return d.require()
	}
	return nil
}

type testRegistry struct {
	store     *config.Store
	refreshes int
}

func (r *testRegistry) CachedSnapshot(context.Context) (indexer.Registry, error) {
	cfg, revision := r.store.Snapshot()
	enabled := true
	for _, id := range cfg.Indexers.Jackett.ExcludedIndexers {
		if id == "jackett/one" {
			enabled = false
		}
	}
	return indexer.Registry{Revision: revision, Instances: []indexer.InstanceSnapshot{{ID: "jackett/one", Kind: indexer.Jackett, Name: "One", Enabled: enabled, SiteURL: "https://user:synthetic-jackett-key@tracker.example/path?apikey=synthetic-site-password#synthetic-private-cookie"}}}, nil
}
func (r *testRegistry) Refresh(ctx context.Context) (indexer.Registry, error) {
	r.refreshes++
	return r.CachedSnapshot(ctx)
}
func managementFixture(t *testing.T) (*Service, *config.Store, *testDefinitions, *testRegistry) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\nindexers:\n  jackett: {enabled: true, url: 'https://jackett.example', api_key: synthetic-jackett-key}\nextra: {retained: yes}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	definition := cardigann.Definition{ID: "example", Name: "Example", Language: "en-US", Type: "private", Links: []string{"https://example.invalid/"}, Caps: cardigann.Capabilities{Modes: map[string][]string{"search": {"q"}}}, Settings: []cardigann.Setting{{Name: "password", Type: "password"}, {Name: "cookie", Type: "text"}, {Name: "filter", Type: "text"}, {Name: "enabled-filter", Type: "checkbox"}}}
	defs := &testDefinitions{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": {Definition: definition, Digest: "digest"}}}}
	registry := &testRegistry{store: store}
	return NewService(store, defs, registry), store, defs, registry
}

func TestManagementCatalogAndInstanceProjectionNeverExposeCredentials(t *testing.T) {
	service, store, _, registry := managementFixture(t)
	definition := "example"
	secret, cookie, filter := "synthetic-site-password", "synthetic-private-cookie", "1080p"
	enabled := true
	saved, err := service.SaveBuiltin(context.Background(), "", store.Revision(), BuiltinPatch{Definition: &definition, Enabled: &enabled, Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &secret}}, "cookie": {Value: &config.IndexerSettingValue{String: &cookie}}, "filter": {Value: &config.IndexerSettingValue{String: &filter}}}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.Instances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	serialized, _ := json.Marshal(page)
	for _, credential := range []string{secret, cookie, "synthetic-jackett-key"} {
		if strings.Contains(string(serialized), credential) {
			t.Fatal("credential exposed by instance projection")
		}
	}
	instance := page.Items[0]
	if instance.ID != saved.Instance.ID {
		t.Fatal("missing instance")
	}
	foundFilter := false
	for _, setting := range instance.Settings {
		if setting.Name == "password" || setting.Name == "cookie" {
			if !setting.Secret || !setting.Configured || setting.Value != nil || setting.Default != nil {
				t.Fatal("secret mask lost configured status")
			}
		}
		if setting.Name == "filter" {
			foundFilter = setting.Value != nil && *setting.Value.String == filter
		}
	}
	if !foundFilter || registry.refreshes != 0 {
		t.Fatal("public settings lost or query refreshed remote catalog")
	}
	name := "Renamed"
	if _, err = service.SaveBuiltin(context.Background(), saved.Instance.ID, store.Revision(), BuiltinPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if *store.Config().Indexers.Builtin.Instances[saved.Instance.ID].Settings["password"].String != secret {
		t.Fatal("name edit erased unsubmitted secret")
	}
	if _, err = service.SaveBuiltin(context.Background(), saved.Instance.ID, store.Revision(), BuiltinPatch{Settings: map[string]config.IndexerSettingChange{"password": {Clear: true}}}); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.Config().Indexers.Builtin.Instances[saved.Instance.ID].Settings["password"]; exists {
		t.Fatal("explicit clear ignored")
	}
	catalog, err := service.Definitions(context.Background(), DefinitionFilter{Privacy: "private", Language: "en-US", Capability: "search", Limit: 1})
	if err != nil || catalog.Total != 1 || len(catalog.Items) != 1 {
		t.Fatal("catalog filters", err)
	}
	if _, err = service.Definitions(context.Background(), DefinitionFilter{Limit: 101}); !errors.Is(err, indexer.ErrInvalidQuery) {
		t.Fatal("unbounded catalog accepted", err)
	}
	raw, _ := os.ReadFile(store.Path())
	if !strings.Contains(string(raw), "retained: yes") {
		t.Fatal("unrelated configuration lost")
	}
}

func TestManagementRuleSwitchAndConcurrentEditsCannotPublishStaleConfiguration(t *testing.T) {
	service, store, defs, _ := managementFixture(t)
	definition := "example"
	before, _ := os.ReadFile(store.Path())
	defs.require = func() error { return indexerdefs.ErrConflict }
	if _, err := service.SaveBuiltin(context.Background(), "", store.Revision(), BuiltinPatch{Definition: &definition}); !errors.Is(err, indexerdefs.ErrConflict) {
		t.Fatal("superseded rules accepted", err)
	}
	after, _ := os.ReadFile(store.Path())
	if string(before) != string(after) {
		t.Fatal("rule conflict changed configuration")
	}
	defs.require = nil
	expected := store.Revision()
	var wg sync.WaitGroup
	var mu sync.Mutex
	success, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.SaveBuiltin(context.Background(), "", expected, BuiltinPatch{Definition: &definition})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else if errors.Is(err, config.ErrConfigRevision) {
				conflicts++
			} else {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if success != 1 || conflicts != 1 || len(store.Config().Indexers.Builtin.Instances) != 1 {
		t.Fatal("stale concurrent write was not fenced")
	}
}

func TestManagementCanDisableBrokenSourceAndPreservesRemoteOwnership(t *testing.T) {
	service, store, defs, registry := managementFixture(t)
	definition := "example"
	enabled := true
	id, _, err := store.UpdateBuiltinInstance(context.Background(), "", store.Revision(), BuiltinPatch{Definition: &definition, Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	defs.unavailable = true
	if _, err = service.SetEnabled(context.Background(), id, store.Revision(), false); err != nil {
		t.Fatal("cannot disable source while rules unavailable", err)
	}
	if _, err = service.SetEnabled(context.Background(), "jackett/one", store.Revision(), false); err != nil {
		t.Fatal(err)
	}
	if len(store.Config().Indexers.Jackett.ExcludedIndexers) != 1 {
		t.Fatal("remote exclusion not persisted")
	}
	if _, err = service.SetEnabled(context.Background(), "jackett/unknown", store.Revision(), true); !errors.Is(err, indexer.ErrInvalidScope) {
		t.Fatal("unknown remote granted", err)
	}
	if _, err = service.RefreshJackett(context.Background()); err != nil || registry.refreshes != 1 {
		t.Fatal("explicit refresh not routed", err)
	}
}

func TestLoginCapabilityFollowsDeclaredFlowInsteadOfPrivacy(t *testing.T) {
	for _, privacy := range []string{"public", "private", "semi-private"} {
		for _, login := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/login=%t", privacy, login), func(t *testing.T) {
				service, store, definitions, _ := managementFixture(t)
				plan := definitions.bundle.Plans["example"]
				plan.Definition.Type = privacy
				if login {
					plan.Definition.Login = &cardigann.Login{Method: "cookie"}
				}
				catalog, err := service.Definitions(context.Background(), DefinitionFilter{Limit: 10})
				if err != nil || len(catalog.Items) != 1 || catalog.Items[0].LoginSupported != login {
					t.Fatalf("catalog login capability: %+v, %v", catalog, err)
				}
				definition := "example"
				saved, err := service.SaveBuiltin(context.Background(), "", store.Revision(), BuiltinPatch{Definition: &definition})
				if err != nil || saved.Instance.LoginSupported != login {
					t.Fatalf("configured login capability: %+v, %v", saved, err)
				}
			})
		}
	}
}
