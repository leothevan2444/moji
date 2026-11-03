package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexerMigrationRetainsLegacyConnectionOrderAndUnrelatedSettings(t *testing.T) {
	for _, raw := range []string{
		"connection:\n  jackett: {url: 'http://jackett.example', api_key: synthetic-key}\nautomation:\n  torrent_selection:\n    fast_rules:\n      indexer_preference: {tracker_ids: [b, a]}\nextra: {retained: yes}\n",
		"config_version: 2\nindexers:\n  jackett: {url: 'http://jackett.example', api_key: synthetic-key}\n  query_order: [b, a]\nextra: {retained: yes}\n",
	} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		s, err := OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if !s.Config().Indexers.Jackett.Enabled {
			t.Fatal("existing Jackett disabled")
		}
		if got := strings.Join(s.Config().Indexers.QueryOrder, ","); got != "jackett/b,jackett/a" {
			t.Fatalf("query order: %s", got)
		}
		if _, err = s.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err != nil {
			t.Fatal(err)
		}
		reopened, err := OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg := reopened.Config()
		if !cfg.Indexers.Jackett.Enabled || cfg.Connection.Jackett.APIKey != "synthetic-key" {
			t.Fatal("connection changed")
		}
		if strings.Join(cfg.Indexers.QueryOrder, ",") != "jackett/b,jackett/a" {
			t.Fatal("order changed")
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "retained: yes") || !strings.Contains(string(data), "config_version: 3") {
			t.Fatal("migration lost unrelated fields")
		}
	}
}

func TestIndexerScopedEditsPreserveSecretsAndRejectStaleWrites(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	definition, name, secret := "example", "Example", "synthetic-password"
	enabled := true
	id, _, err := s.UpdateBuiltinInstance(ctx, "", s.Revision(), BuiltinInstancePatch{Definition: &definition, Name: &name, Enabled: &enabled, Settings: map[string]IndexerSettingChange{"password": {Value: &IndexerSettingValue{String: &secret}}}})
	if err != nil {
		t.Fatal(err)
	}
	stale := s.Revision()
	name = "Renamed"
	_, _, err = s.UpdateBuiltinInstance(ctx, id, stale, BuiltinInstancePatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.UpdateBuiltinInstance(ctx, id, stale, BuiltinInstancePatch{Enabled: &enabled}); !errors.Is(err, ErrConfigRevision) {
		t.Fatalf("stale write: %v", err)
	}
	if _, err = s.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err != nil {
		t.Fatal(err)
	}
	cfg := s.Config()
	instance := cfg.Indexers.Builtin.Instances[id]
	if instance.Name != "Renamed" || *instance.Settings["password"].String != secret {
		t.Fatal("unsubmitted secret changed")
	}
	*instance.Settings["password"].String = "outside mutation"
	if *s.Config().Indexers.Builtin.Instances[id].Settings["password"].String != secret {
		t.Fatal("configuration snapshot aliases stored secrets")
	}
	if _, _, err = s.UpdateBuiltinInstance(ctx, id, s.Revision(), BuiltinInstancePatch{Settings: map[string]IndexerSettingChange{"password": {Clear: true}}}); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Config().Indexers.Builtin.Instances[id].Settings["password"]; ok {
		t.Fatal("secret not cleared")
	}
	if _, err = s.ReorderIndexers(ctx, s.Revision(), []string{id, "jackett/a"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RemoveBuiltinInstance(ctx, id, s.Revision()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(s.Config().Indexers.QueryOrder, ",") != "jackett/a" {
		t.Fatal("removed instance still ordered")
	}
}

func TestIndexerProfileAndInvalidChangesAreAtomic(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	profile := IndexerNetworkProfile{ProxyURL: "http://proxy.example:8080"}
	if _, err := s.SetIndexerNetworkProfile(ctx, s.Revision(), "proxy", &profile); err != nil {
		t.Fatal(err)
	}
	definition, network := "example", "proxy"
	id, _, err := s.UpdateBuiltinInstance(ctx, "", s.Revision(), BuiltinInstancePatch{Definition: &definition, NetworkProfile: &network})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.Path())
	revision := s.Revision()
	if _, err = s.SetIndexerNetworkProfile(ctx, revision, "proxy", nil); err == nil {
		t.Fatal("removed a referenced profile")
	}
	if _, err = s.ApplyPatch(map[string]any{"indexers": map[string]any{"builtin": map[string]any{"instances": map[string]any{}}}}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("generic instance replacement: %v", err)
	}
	if _, _, err = s.UpdateBuiltinInstance(ctx, id, revision, BuiltinInstancePatch{Settings: map[string]IndexerSettingChange{"password": {Clear: true, Value: &IndexerSettingValue{}}}}); err == nil {
		t.Fatal("ambiguous clear accepted")
	}
	after, _ := os.ReadFile(s.Path())
	if string(before) != string(after) || s.Revision() != revision {
		t.Fatal("failed edit changed persistent configuration")
	}
	invoked := false
	if err = s.WithRevision(ctx, "stale", func() error { invoked = true; return nil }); !errors.Is(err, ErrConfigRevision) || invoked {
		t.Fatal("revision fence ran stale activation")
	}
}

func TestTrustedOriginsAreOriginsAndCannotBroadenPathGrants(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	definition := "example"
	origins := []string{"https://download.example", "https://auth.example:8443/"}
	id, _, err := s.UpdateBuiltinInstance(ctx, "", s.Revision(), BuiltinInstancePatch{Definition: &definition, TrustedOrigins: &origins})
	if err != nil {
		t.Fatal(err)
	}
	ack := s.Revision()
	before, _ := os.ReadFile(s.Path())
	for _, raw := range []string{"https://download.example/path", "https://download.example/.", "https://user:secret@download.example", "https://download.example?token=secret", "https://download.example?", "https://download.example#", "https://*.example", "*.example"} {
		invalid := []string{raw}
		if _, _, err = s.UpdateBuiltinInstance(ctx, id, ack, BuiltinInstancePatch{TrustedOrigins: &invalid}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("non-origin grant accepted", err)
		}
	}
	after, _ := os.ReadFile(s.Path())
	if string(before) != string(after) {
		t.Fatal("invalid origins changed saved settings")
	}
	empty := []string{}
	if _, _, err = s.UpdateBuiltinInstance(ctx, id, ack, BuiltinInstancePatch{TrustedOrigins: &empty}); err != nil || len(s.Config().Indexers.Builtin.Instances[id].TrustedOrigins) != 0 {
		t.Fatal("explicit revocation failed", err)
	}
}
