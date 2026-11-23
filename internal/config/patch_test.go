package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func patchStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte("config_version: 2\nindexers:\n  jackett:\n    api_key: secret\n  query_order: [a, b]\nsystem:\n  language: auto\n  developer_mode: true\n  image_cache:\n    enabled: true\n    max_size_mb: 1024\n    retention_days: 30\nextra:\n  retained: yes\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
func TestPatchMergesExplicitFalseEmptyListAndPreservesOtherFields(t *testing.T) {
	s := patchStore(t)
	_, err := s.ApplyPatch(map[string]any{"system": map[string]any{"developer_mode": false}, "indexers": map[string]any{"query_order": []any{}}})
	if err != nil {
		t.Fatal(err)
	}
	next, err := OpenStore(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	cfg := next.Config()
	if cfg.System.DeveloperMode || cfg.Connection.Jackett.APIKey != "secret" || cfg.Automation.TorrentSelection.Version != 7 {
		t.Fatalf("patch replaced untouched or explicit fields")
	}
	raw, _ := os.ReadFile(s.Path())
	if !strings.Contains(string(raw), "retained: yes") {
		t.Fatal("lost extension field")
	}
}
func TestConcurrentPatchesMergeLatestUnderOneLock(t *testing.T) {
	s := patchStore(t)
	var wg sync.WaitGroup
	for _, patch := range []map[string]any{{"system": map[string]any{"language": "en"}}, {"system": map[string]any{"developer_mode": false}}} {
		wg.Add(1)
		go func(p map[string]any) {
			defer wg.Done()
			if _, err := s.ApplyPatch(p); err != nil {
				t.Error(err)
			}
		}(patch)
	}
	wg.Wait()
	cfg := s.Config()
	if cfg.System.Language != "en" || cfg.System.DeveloperMode {
		t.Fatal("concurrent updates lost")
	}
}
func TestInvalidOrFailedPatchDoesNotPublishMemoryOrDisk(t *testing.T) {
	s := patchStore(t)
	before, _ := os.ReadFile(s.Path())
	for _, patch := range []map[string]any{{"system": map[string]any{"language": "bogus"}}, {"system": map[string]any{"image_cache": map[string]any{"retention_days": 0}}}, {"tasks": map[string]any{"torrent_selection": map[string]any{"inspection_candidate_limit": 0}}}} {
		if _, err := s.ApplyPatch(patch); err == nil {
			t.Fatal("invalid patch accepted")
		}
		after, _ := os.ReadFile(s.Path())
		if string(after) != string(before) {
			t.Fatal("invalid patch changed disk")
		}
	}
	oldPath := s.path
	s.path = filepath.Join(t.TempDir(), "missing", "config.yaml")
	if _, err := s.ApplyPatch(map[string]any{"system": map[string]any{"language": "en"}}); err == nil {
		t.Fatal("expected write failure")
	}
	if s.Config().System.Language != "auto" {
		t.Fatal("failed write published memory")
	}
	s.path = oldPath
}

func TestConflictingResourceFiltersDoNotChangeSavedSettings(t *testing.T) {
	s := patchStore(t)
	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ApplyPatch(map[string]any{"tasks": map[string]any{"torrent_selection": map[string]any{
		"version": 7, "active_group": "main",
		"rules":  []any{map[string]any{"id": "shared", "name": "Shared", "conditions": []any{map[string]any{"scope": "TITLE", "value": "label"}}}},
		"groups": []any{map[string]any{"id": "main", "name": "Main", "require": []any{"shared"}, "exclude": []any{"shared"}, "fallback": true, "tiers": []any{}}},
	}}})
	if err == nil {
		t.Fatal("conflicting pre-filter saved")
	}
	after, readErr := os.ReadFile(s.Path())
	if readErr != nil || string(before) != string(after) || len(s.Config().Automation.TorrentSelection.Groups) != 0 {
		t.Fatal("invalid patch changed settings", readErr)
	}
}

func TestEmptyResourceRulePersistsWithoutConditions(t *testing.T) {
	s := patchStore(t)
	_, err := s.ApplyPatch(map[string]any{"tasks": map[string]any{"torrent_selection": map[string]any{
		"version": 7, "active_group": "", "groups": []any{},
		"rules": []any{map[string]any{"id": "draft", "name": "Draft", "conditions": []any{}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFromPath(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	rules := loaded.Automation.TorrentSelection.Rules
	if len(rules) != 1 || len(rules[0].Conditions) != 0 {
		t.Fatal("empty condition reappeared", rules)
	}
}

func TestEmptyPriorityTierPersists(t *testing.T) {
	s := patchStore(t)
	_, err := s.ApplyPatch(map[string]any{"tasks": map[string]any{"torrent_selection": map[string]any{
		"version": 7, "active_group": "main", "rules": []any{},
		"groups": []any{map[string]any{"id": "main", "name": "Main", "fallback": false, "require": []any{}, "exclude": []any{}, "tiers": []any{map[string]any{"id": "empty", "name": "Empty", "rules": []any{}}}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFromPath(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	tiers := loaded.Automation.TorrentSelection.Groups[0].Tiers
	if len(tiers) != 1 || tiers[0].ID != "empty" || len(tiers[0].Rules) != 0 {
		t.Fatal("empty tier lost", tiers)
	}
}
