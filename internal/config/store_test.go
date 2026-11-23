package config

import (
	"github.com/leothevan2444/moji/internal/resourceselection"
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
  task_processing_frequency: MEDIUM
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
	if reloaded.Automation.TaskProcessingFrequency != "MEDIUM" {
		t.Fatalf("expected automation frequency preserved, got %s", reloaded.Automation.TaskProcessingFrequency)
	}
	if reloaded.Connection.QBittorrent.Username != "operator" {
		t.Fatalf("expected updated username, got %q", reloaded.Connection.QBittorrent.Username)
	}
}

func TestResourceSearchIntervalPersistsAndRejectsInvalidValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("automation: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Config().Automation.EffectiveSourcingRetryIntervalHours() != 6 {
		t.Fatal("legacy configuration lost the default interval")
	}
	if _, err := store.UpdateAutomation("MEDIUM", nil, DefaultSubscriptionReleasePolicyConfig(), DefaultTorrentSelectionConfig(), 12); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil || reopened.Config().Automation.EffectiveSourcingRetryIntervalHours() != 12 {
		t.Fatalf("resource interval not persisted: %v", err)
	}
	for _, value := range []int{0, -1, 169} {
		if _, err := store.UpdateAutomation("MEDIUM", nil, DefaultSubscriptionReleasePolicyConfig(), DefaultTorrentSelectionConfig(), value); err == nil {
			t.Fatalf("invalid interval %d accepted", value)
		}
	}
	if store.Config().Automation.EffectiveSourcingRetryIntervalHours() != 12 {
		t.Fatal("invalid update modified configuration")
	}
}

func TestVRReleasePolicyDefaultsAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("automation: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Config().Automation.SubscriptionReleasePolicy.Effective().VRBehavior; got != SubscriptionReleaseBehaviorReview {
		t.Fatalf("legacy configuration VR behavior = %s, want REVIEW", got)
	}
	for _, behavior := range []SubscriptionReleaseBehavior{SubscriptionReleaseBehaviorDownload, SubscriptionReleaseBehaviorReview, SubscriptionReleaseBehaviorBlock} {
		policy := DefaultSubscriptionReleasePolicyConfig()
		policy.VRBehavior = behavior
		if _, err := store.UpdateAutomation("MEDIUM", nil, policy, DefaultTorrentSelectionConfig(), 6); err != nil {
			t.Fatal(err)
		}
		reloaded, err := OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := reloaded.Config().Automation.SubscriptionReleasePolicy.VRBehavior; got != behavior {
			t.Fatalf("VR behavior = %s, want %s", got, behavior)
		}
	}
}

func TestStoreUpdateAutomationPersistsTorrentSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("automation: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	p := DefaultTorrentSelectionConfig()
	p.Rules = []resourceselection.Rule{{ID: "kept", Name: "Kept", Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: "label"}}}}
	if _, err = store.UpdateAutomation("MEDIUM", nil, DefaultSubscriptionReleasePolicyConfig(), p); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Automation.TorrentSelection.Rules) != 1 {
		t.Fatalf("policy lost: %+v", got.Automation.TorrentSelection)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "fast_rules:") || !strings.Contains(string(data), "version: 4") {
		t.Fatal("retired config persisted")
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
  downloads:
    qb_root: "/downloads"
    moji_root: "/moji/downloads"
  library:
    moji_root: "/moji/library"
    stash_root: "/library"
  transfer:
    action: COPY
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

func TestLoadFromPathRejectsRetiredResourceRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `automation:
  torrent_selection:
    enabled: true
    inspection_candidate_limit: 5
    fast_rule_order:
      - "SEEDERS"
      - "SEEDERS"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadFromPath(path)
	if err == nil || !strings.Contains(err.Error(), "is retired") {
		t.Fatalf("expected explicit manual conversion error, got %v", err)
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

	enabled := false
	cfg, err := store.UpdateSystem(TaskDeletePolicyRemoveTorrentAndFiles, ImageCacheConfig{Enabled: &enabled, MaxSizeMB: 512, RetentionDays: 14})
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
	if reloaded.System.ImageCache.EffectiveEnabled() || reloaded.System.ImageCache.MaxSizeMB != 512 || reloaded.System.ImageCache.RetentionDays != 14 {
		t.Fatalf("expected persisted image cache settings, got %+v", reloaded.System.ImageCache)
	}
}

func TestStoreUpdateSystemPersistsStashBoxCacheTTL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("system:\n  task_delete_policy: KEEP_ONLY\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	if _, err := store.UpdateSystemWithDataCache(TaskDeletePolicyKeepOnly, ImageCacheConfig{Enabled: &enabled, MaxSizeMB: 1024, RetentionDays: 30}, StashBoxDataCacheConfig{TTLHours: 36}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.System.StashBoxDataCache.TTLHours != 36 {
		t.Fatalf("TTL hours = %d, want 36", reloaded.System.StashBoxDataCache.TTLHours)
	}
}

func TestStashBoxDataCacheConfigNormalizeBounds(t *testing.T) {
	for _, test := range []struct {
		input int
		want  int
	}{{0, 24}, {11, 12}, {12, 12}, {360, 360}, {361, 360}} {
		if got := (StashBoxDataCacheConfig{TTLHours: test.input}).Normalize().TTLHours; got != test.want {
			t.Errorf("Normalize(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}

func TestIngestSubtitleSwitchPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("ingest:\n  keep_subtitles: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := store.UpdateIngest(DownloadsIngestConfig{}, LibraryIngestConfig{}, TransferIngestConfig{Action: "COPY"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Ingest.KeepSubtitles {
		t.Fatal("subtitle setting was not set")
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Config().Ingest.KeepSubtitles {
		t.Fatal("subtitle setting was not persisted")
	}
}

func TestIngestTranscodeSwitchDefaultsOffAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("ingest:\n  keep_subtitles: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Config().Ingest.GenerateTranscodes {
		t.Fatal("old configurations must default transcodes to off")
	}
	for _, enabled := range []bool{true, false} {
		cfg, err := store.UpdateIngest(DownloadsIngestConfig{}, LibraryIngestConfig{}, TransferIngestConfig{Action: "COPY"}, true, enabled)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Ingest.GenerateTranscodes != enabled || !cfg.Ingest.KeepSubtitles {
			t.Fatalf("incorrect ingest settings: %+v", cfg.Ingest)
		}
		store, err = OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if store.Config().Ingest.GenerateTranscodes != enabled {
			t.Fatalf("transcode setting %t did not survive reopening", enabled)
		}
	}
}

func TestIngestRejectsUnsupportedActionsWithoutChangingConfig(t *testing.T) {
	for _, action := range []string{"", "SYMLINK", "UNKNOWN"} {
		t.Run(action, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			original := "ingest:\n  transfer:\n    action: COPY\n"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			store, err := OpenStore(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.UpdateIngest(DownloadsIngestConfig{}, LibraryIngestConfig{}, TransferIngestConfig{Action: action}, false, false); err == nil {
				t.Fatal("expected unsupported delivery action to be rejected")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != original || store.Config().Ingest.Transfer.Action != "COPY" {
				t.Fatal("rejected settings changed the persisted or in-memory configuration")
			}
		})
	}
}

func TestLoadRejectsUnsupportedIngestAction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("ingest:\n  transfer:\n    action: SYMLINK\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromPath(path); err == nil || !strings.Contains(err.Error(), "COPY or MOVE") {
		t.Fatalf("expected invalid delivery action error, got %v", err)
	}
}

func TestTaskProcessingIntervalPersistsAndRejectsUnsupportedIntervals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, interval := range []string{"HIGH", "MEDIUM", "LOW"} {
		cfg, err := store.UpdateAutomation(interval, nil, DefaultSubscriptionReleasePolicyConfig(), DefaultTorrentSelectionConfig(), 6)
		if err != nil || cfg.Automation.TaskProcessingFrequency != interval {
			t.Fatal("interval lost", interval, err)
		}
		reopened, err := OpenStore(path)
		if err != nil || reopened.Config().Automation.TaskProcessingFrequency != interval {
			t.Fatal("interval was not persisted", err)
		}
	}
	for _, interval := range []string{"", "FAST", "30", "medium"} {
		if _, err := store.UpdateAutomation(interval, nil, DefaultSubscriptionReleasePolicyConfig(), DefaultTorrentSelectionConfig(), 12); err == nil {
			t.Fatal("invalid interval accepted", interval)
		}
		if store.Config().Automation.TaskProcessingFrequency != "LOW" || store.Config().Automation.EffectiveSourcingRetryIntervalHours() != 6 {
			t.Fatal("invalid save changed configuration")
		}
	}
}

func TestFailedDurableWriteDoesNotActivateNewConfiguration(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "settings")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "config.yaml")
	original := []byte("connection:\n  stash:\n    url: http://old.invalid/graphql\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	oldURL := store.Config().Connection.Stash.URL
	offline := parent + ".offline"
	if err := os.Rename(parent, offline); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(offline, parent)
	if _, err := store.UpdateStash("http://new.invalid/graphql", ""); err == nil {
		t.Fatal("write to unavailable storage succeeded")
	}
	if store.Config().Connection.Stash.URL != oldURL {
		t.Fatal("failed setting was activated in memory")
	}
	raw, err := os.ReadFile(filepath.Join(offline, "config.yaml"))
	if err != nil || string(raw) != string(original) {
		t.Fatal("failed write changed the previous file", err)
	}
}

func TestDeveloperModePersistsAndUnrelatedSystemUpdatesPreserveIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("system:\n  developer_mode: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateSystemWithDataCache(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), StashBoxDataCacheConfig{TTLHours: 24}, true); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil || !reopened.Config().System.DeveloperMode {
		t.Fatal("developer mode was not persisted", err)
	}
	if _, err = store.UpdateSystem(TaskDeletePolicyRemoveTorrent); err != nil {
		t.Fatal(err)
	}
	reopened, err = OpenStore(path)
	if err != nil || !reopened.Config().System.DeveloperMode {
		t.Fatal("unrelated update reset developer mode", err)
	}
	if _, err = store.UpdateSystemWithDataCache(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), StashBoxDataCacheConfig{TTLHours: 24}, false); err != nil {
		t.Fatal(err)
	}
	reopened, err = OpenStore(path)
	if err != nil || reopened.Config().System.DeveloperMode {
		t.Fatal("developer mode was not disabled", err)
	}
}

func TestDeveloperModeFailedSaveDoesNotEnableItInMemory(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "settings")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "config.yaml")
	if err := os.WriteFile(path, []byte("system:\n  developer_mode: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(parent, parent+".offline"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateSystemWithDataCache(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), StashBoxDataCacheConfig{TTLHours: 24}, true); err == nil {
		t.Fatal("save to missing directory succeeded")
	}
	if store.Config().System.DeveloperMode {
		t.Fatal("failed save enabled developer mode")
	}
}
