package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutMigrationRoundTripAndScopedSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := `connection:
  stash: {url: 'http://stash', api_key: stash-secret}
  jackett: {url: 'http://jackett', api_key: jackett-secret, password: dashboard}
  qbittorrent: {url: 'http://qb', username: user, password: qb-secret}
ingest:
  keep_subtitles: true
  generate_transcodes: false
  downloads: {qb_root: /q, moji_root: /m}
  library: {moji_root: /lib, stash_root: /stash}
  transfer: {action: COPY}
automation:
  task_processing_enabled: false
  task_progress_sync_interval_seconds: 120
  subscription_poll_interval_hours: 24
  selected_stash_box_endpoints: [second, first]
  torrent_selection:
    enabled: true
    inspection_candidate_limit: 3
    fast_rules:
      indexer_preference: {enabled: true, tracker_ids: [b, a]}
      title_match:
        enabled: true
        clauses: [{pattern: sample, pattern_mode: PLAIN, effect: AVOID}]
    torrent_rules:
      torrent_file_name_match:
        enabled: true
        clauses: [{pattern: movie, pattern_mode: PLAIN, effect: LOCK}]
system: {developer_mode: true, stash_box_data_cache: {ttl_hours: 36, max_size_mb: 99}}
logging: {level: debug, file_path: /tmp/moji.log, max_entries: 88}
custom: {untouched: true}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	before := store.Config()
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"config_version: 2", "indexers:", "query_order:", "downloader:", "metadata_priority:", "tasks:", "DEMOTE", "LOCK", "logging:", "untouched: true"} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("missing canonical field %s", expected)
		}
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	after := reopened.Config()
	if before.Connection != after.Connection || before.Ingest != after.Ingest || before.Logging != after.Logging || after.Automation.EffectiveTaskProcessingEnabled() || after.System.StashBoxDataCache.TTLHours != 36 {
		t.Fatalf("migration lost settings: connection=%v ingest=%v logging=%v processing=%v cache=%v", before.Connection == after.Connection, before.Ingest == after.Ingest, before.Logging == after.Logging, after.Automation.EffectiveTaskProcessingEnabled(), after.System.StashBoxDataCache.TTLHours)
	}
	if _, err := reopened.UpdateAutomationSection("METADATA", AutomationConfig{StashBoxEndpoints: []string{"third"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.UpdateAutomationSection("RESOURCES", AutomationConfig{TorrentSelection: DefaultTorrentSelectionConfig()}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Automation.StashBoxEndpoints[0] != "third" || got.Automation.TorrentSelection.FastRules.IndexerPreference.TrackerIDs[0] != "b" || got.Automation.TaskProgressSyncIntervalSeconds != 120 || got.Automation.EffectiveTaskProcessingEnabled() {
		t.Fatal("section save overwrote another section")
	}
	// Subsequent writes must also preserve logging and unrelated fields.
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "untouched: true") || got.Logging != before.Logging {
		t.Fatal("repeat save lost unrelated settings")
	}
}

func TestLayoutRejectsMixedAndUnknownVersions(t *testing.T) {
	for _, raw := range []string{"config_version: 3\n", "config_version: 2\nconnection: {}\n", "indexers: {}\n"} {
		var cfg Config
		if err := yaml.Unmarshal([]byte(raw), &cfg); err == nil {
			t.Fatalf("accepted ambiguous config %q", raw)
		}
	}
}

func TestSectionFailureAndSystemSavePreserveCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("automation: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateStashBoxCache(72); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSystemSection(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), nil, true); err != nil {
		t.Fatal(err)
	}
	if store.Config().System.StashBoxDataCache.TTLHours != 72 {
		t.Fatal("system save replaced Stash cache settings")
	}
	before, _ := os.ReadFile(path)
	bad := DefaultTorrentSelectionConfig()
	bad.FastRules.TitleMatch = TitleMatchRuleSettings{Enabled: true, Clauses: []TitleMatchClause{{Pattern: "[", PatternMode: TitleMatchPatternModeRegex, Effect: TitleMatchEffectExclude}}}
	if _, err := store.UpdateAutomationSection("RESOURCES", AutomationConfig{TorrentSelection: bad}); err == nil {
		t.Fatal("accepted invalid hard rule regex")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid rule modified persisted config")
	}
}

func TestLanguageAndDeletionPolicyHaveIndependentOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 2\nsystem:\n  task_delete_policy: REMOVE_TORRENT\n  language: en\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	tasks := document["tasks"].(map[string]any)
	system := document["system"].(map[string]any)
	if tasks["task_delete_policy"] != "REMOVE_TORRENT" || system["task_delete_policy"] != nil || system["language"] != "en" {
		t.Fatal("incorrect configuration ownership")
	}
	if _, err := store.UpdateTaskDeletePolicy(TaskDeletePolicyRemoveTorrentAndFiles); err != nil {
		t.Fatal(err)
	}
	language := "zh-CN"
	if _, err := store.UpdateSystemPreferences(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), nil, false, &language, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.System.EffectiveTaskDeletePolicy() != TaskDeletePolicyRemoveTorrentAndFiles || loaded.System.EffectiveLanguage() != "zh-CN" {
		t.Fatal("system save overwrote task policy")
	}
	bad := "invalid"
	if _, err := store.UpdateSystemPreferences(TaskDeletePolicyKeepOnly, DefaultImageCacheConfig(), nil, false, &bad, true); err == nil {
		t.Fatal("accepted invalid language")
	}
	if store.Config().System.EffectiveLanguage() != "zh-CN" {
		t.Fatal("invalid update changed language")
	}
}

func TestLanguageValidationAndDeletionConflict(t *testing.T) {
	for _, raw := range []string{
		"config_version: 2\nsystem: {language: ar-XB}\n",
		"config_version: 2\nsystem: {task_delete_policy: KEEP_ONLY}\ntasks: {task_delete_policy: REMOVE_TORRENT}\n",
	} {
		var cfg Config
		if err := yaml.Unmarshal([]byte(raw), &cfg); err == nil {
			t.Fatal("accepted invalid configuration")
		}
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte("system: {}\n"), &cfg); err != nil || cfg.System.EffectiveLanguage() != "auto" {
		t.Fatal("missing language did not default to auto")
	}
}
