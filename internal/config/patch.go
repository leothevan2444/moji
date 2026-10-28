package config

import (
	"fmt"
	"math"

	"github.com/leothevan2444/moji/internal/durablefile"
	"gopkg.in/yaml.v3"
)

// ApplyPatch merges a typed API patch into the latest persisted configuration
// while holding the same lock used by all other settings writers. Lists replace.
// It publishes neither memory nor disk state until validation and durable write succeed.
func (s *Store) ApplyPatch(patch map[string]any) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var root yaml.Node
	if err := yaml.Unmarshal(s.saved, &root); err != nil {
		return nil, err
	}
	top := documentNode(&root)
	if existingMapValue(top, "config_version") == nil {
		canonicalizeConfigNode(top)
	}
	raw, err := yaml.Marshal(patch)
	if err != nil {
		return nil, err
	}
	var delta yaml.Node
	if err := yaml.Unmarshal(raw, &delta); err != nil {
		return nil, err
	}
	mergePatchNode(top, documentNode(&delta))
	data, err := yaml.Marshal(&root)
	if err != nil {
		return nil, err
	}
	next, err := parseStore(s.path, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	if err := validatePatch(patch, next.cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	if err := durablefile.Write(s.path, data, 0o600); err != nil {
		return nil, err
	}
	s.cfg, s.root, s.saved = next.cfg, next.root, next.saved
	result := *s.cfg
	return &result, nil
}

func mergePatchNode(target, patch *yaml.Node) {
	for i := 0; i+1 < len(patch.Content); i += 2 {
		key, value := patch.Content[i].Value, patch.Content[i+1]
		if value.Tag == "!!null" {
			continue
		}
		current := mapValue(target, key)
		if value.Kind == yaml.MappingNode {
			mergePatchNode(ensureMapValue(current), value)
		} else {
			*current = *value
		}
	}
}

func patchValue(patch map[string]any, keys ...string) (any, bool) {
	for i, key := range keys {
		value, ok := patch[key]
		if !ok || value == nil {
			return nil, false
		}
		if i == len(keys)-1 {
			return value, true
		}
		patch, ok = value.(map[string]any)
		if !ok {
			return nil, false
		}
	}
	return nil, false
}

func validatePatch(patch map[string]any, cfg *Config) error {
	checks := []struct {
		path     []string
		min, max int
	}{
		{[]string{"tasks", "sourcing_retry_interval_hours"}, 1, 168},
		{[]string{"tasks", "subscription_release_policy", "max_group_performer_count"}, 1, math.MaxInt},
		{[]string{"tasks", "torrent_selection", "inspection_candidate_limit"}, 1, math.MaxInt},
		{[]string{"stash", "data_cache", "ttl_hours"}, 12, 360},
		{[]string{"system", "image_cache", "max_size_mb"}, 64, 20480},
		{[]string{"system", "image_cache", "retention_days"}, 1, 365},
	}
	for _, check := range checks {
		if _, ok := patchValue(patch, check.path...); ok && (patchNumber(patch, check.path...) < check.min || patchNumber(patch, check.path...) > check.max) {
			return fmt.Errorf("%v must be between %d and %d", check.path, check.min, check.max)
		}
	}
	if _, ok := patchValue(patch, "tasks", "task_progress_sync_interval_seconds"); ok {
		switch cfg.Automation.TaskProgressSyncIntervalSeconds {
		case 30, 60, 120, 300:
		default:
			return fmt.Errorf("task interval must be 30, 60, 120 or 300 seconds")
		}
	}
	if action := cfg.Ingest.Transfer.Action; action != "" && action != "COPY" && action != "MOVE" {
		return fmt.Errorf("transfer action must be COPY or MOVE")
	}
	return ValidateLanguage(cfg.System.Language)
}

func patchNumber(patch map[string]any, keys ...string) int {
	value, _ := patchValue(patch, keys...)
	switch n := value.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
