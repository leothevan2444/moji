package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreUpdateQBittorrentPreservesUnmodeledSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `database:
  dsn: "user:password@tcp(localhost:3306)/media_library?parseTime=true"
connection:
  qbittorrent:
    url: "http://localhost:8080"
    username: "admin"
    password: "secret"
tasks:
  progress_sync_interval_seconds: 60
automation:
  task_progress_sync_interval_seconds: 90
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	cfg, err := store.UpdateQBittorrent("http://localhost:8081", "operator", "", "/downloads", "moji", "auto")
	if err != nil {
		t.Fatalf("update qbittorrent: %v", err)
	}
	if cfg.Connection.QBittorrent.Password != "" {
		t.Fatalf("expected password cleared by empty input, got %q", cfg.Connection.QBittorrent.Password)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(updated)
	if !strings.Contains(text, `database:`) || !strings.Contains(text, `dsn: "user:password@tcp(localhost:3306)/media_library?parseTime=true"`) {
		t.Fatalf("expected database section preserved, got:\n%s", text)
	}

	reloaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.Automation.TaskProgressSyncIntervalSeconds != 90 {
		t.Fatalf("expected automation interval preserved, got %d", reloaded.Automation.TaskProgressSyncIntervalSeconds)
	}
	if reloaded.Connection.QBittorrent.Username != "operator" {
		t.Fatalf("expected updated username, got %q", reloaded.Connection.QBittorrent.Username)
	}
}

func TestStoreUpdateAutomationPersistsTorrentSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `connection:
  jackett:
    url: "http://localhost:9117"
    api_key: "secret"
    password: "pw"
automation:
  task_progress_sync_interval_seconds: 60
  subscription_poll_interval_hours: 1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	cfg, err := store.UpdateAutomation(60, 1, []string{"https://javstash.example.org/graphql"}, TorrentSelectionConfig{
		Enabled: true,
		Rules: []TorrentSelectionRule{
			{
				ID:        "pref",
				Name:      "Preferred Indexers",
				Type:      TorrentSelectionRuleTypeIndexerPreference,
				Enabled:   true,
				Direction: TorrentSelectionDirectionDesc,
				IndexerPreference: IndexerPreferenceRuleConfig{
					TrackerIDs: []string{"alpha", "beta"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("update automation: %v", err)
	}
	if len(cfg.Automation.TorrentSelection.Rules) != 1 {
		t.Fatalf("unexpected torrent selection: %+v", cfg.Automation.TorrentSelection)
	}
	if len(cfg.Automation.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected stash-box endpoints: %+v", cfg.Automation.StashBoxEndpoints)
	}

	reloaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	got := reloaded.Automation.TorrentSelection.Effective()
	if !got.Enabled || len(got.Rules) != 1 || len(got.Rules[0].IndexerPreference.TrackerIDs) != 2 {
		t.Fatalf("expected persisted torrent selection, got %+v", got)
	}
	if got.Rules[0].Name != "Preferred Indexers" {
		t.Fatalf("expected persisted rule name, got %+v", got.Rules[0])
	}
}

func TestLoadFromPathUsesDirectStashURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `connection:
  stash:
    url: "http://stash.example"
    api_key: "secret"
ingest:
  delivery_mode: "PATH_MAP"
  path_map:
    qbittorrent_path_prefix: "/downloads"
    stash_path_prefix: "/library"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Connection.Stash.GraphQLEndpoint() != "http://stash.example/graphql" {
		t.Fatalf("expected derived graphql endpoint, got %q", cfg.Connection.Stash.GraphQLEndpoint())
	}
}

func TestStoreUpdateSystemPersistsTaskDeletePolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `system:
  task_delete_policy: "KEEP_ONLY"
automation:
  selected_stash_box_endpoints: []
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	cfg, err := store.UpdateSystem(TaskDeletePolicyRemoveTorrentAndFiles)
	if err != nil {
		t.Fatalf("update system: %v", err)
	}
	if cfg.System.EffectiveTaskDeletePolicy() != TaskDeletePolicyRemoveTorrentAndFiles {
		t.Fatalf("unexpected system config: %+v", cfg.System)
	}

	reloaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.System.EffectiveTaskDeletePolicy() != TaskDeletePolicyRemoveTorrentAndFiles {
		t.Fatalf("expected persisted task delete policy, got %q", reloaded.System.EffectiveTaskDeletePolicy())
	}
}
