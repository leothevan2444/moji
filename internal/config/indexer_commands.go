package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type IndexerSettingChange struct {
	Value *IndexerSettingValue
	Clear bool
}
type IndexerCachePatch struct {
	Enabled                                *bool
	TTLSeconds, EmptyTTLSeconds, MaxSizeMB *int
}

func (s *Store) UpdateIndexerSearchCache(ctx context.Context, expected string, patch IndexerCachePatch) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	cfg := s.cfg.Indexers.Clone()
	if patch.Enabled != nil {
		cfg.SearchCache.Enabled = *patch.Enabled
	}
	if patch.TTLSeconds != nil {
		cfg.SearchCache.TTLSeconds = *patch.TTLSeconds
	}
	if patch.EmptyTTLSeconds != nil {
		cfg.SearchCache.EmptyTTLSeconds = *patch.EmptyTTLSeconds
	}
	if patch.MaxSizeMB != nil {
		cfg.SearchCache.MaxSizeMB = *patch.MaxSizeMB
	}
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error { setNodeValue(indexers, "search_cache", cfg.SearchCache); return nil })
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}

type BuiltinInstancePatch struct {
	Definition, Name, BaseURL, NetworkProfile *string
	Enabled                                   *bool
	TrustedOrigins                            *[]string
	Settings                                  map[string]IndexerSettingChange
}

// UpdateBuiltinInstance applies only submitted fields under the same lock as
// patchSettings. Empty ID creates a stable UUID; empty/mismatched revision never
// authorizes a write. Secrets absent from Settings remain exactly as stored.
func (s *Store) UpdateBuiltinInstance(ctx context.Context, id, expected string, patch BuiltinInstancePatch) (string, *Config, error) {
	return s.UpdateBuiltinInstanceGuarded(ctx, id, expected, patch, nil)
}

// Guard inspects the compiled active revision only. Both writers and rule
// activation acquire configuration before definitions; no I/O belongs in it.
func (s *Store) UpdateBuiltinInstanceGuarded(ctx context.Context, id, expected string, patch BuiltinInstancePatch, guard func() error) (string, *Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return "", nil, ErrConfigRevision
	}
	cfg := s.cfg.Indexers.Clone()
	instance, exists := cfg.Builtin.Instances[id]
	if id == "" {
		id = uuid.NewString()
		instance.NetworkProfile = "direct"
	} else if !exists {
		return "", nil, ErrConfigRevision
	}
	var err error
	instance, err = MergeBuiltinInstance(instance, patch)
	if err != nil {
		return "", nil, err
	}
	cfg.Builtin.Instances[id] = instance
	if err := cfg.Validate(); err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	if guard != nil {
		if err := guard(); err != nil {
			return "", nil, err
		}
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		builtin := ensureMapValue(mapValue(indexers, "builtin"))
		instances := ensureMapValue(mapValue(builtin, "instances"))
		setNodeValue(instances, id, instance)
		return nil
	})
	return id, next, err
}

// MergeBuiltinInstance applies an explicit patch to a private copy, allowing
// the application to validate exactly what the locked writer will save.
func MergeBuiltinInstance(instance BuiltinIndexerInstance, patch BuiltinInstancePatch) (BuiltinIndexerInstance, error) {
	settings := map[string]IndexerSettingValue{}
	for key, value := range instance.Settings {
		settings[key] = value.clone()
	}
	instance.Settings = settings
	instance.TrustedOrigins = append([]string(nil), instance.TrustedOrigins...)
	if patch.Definition != nil {
		instance.Definition = *patch.Definition
	}
	if patch.Name != nil {
		instance.Name = *patch.Name
	}
	if patch.BaseURL != nil {
		instance.BaseURL = *patch.BaseURL
	}
	if patch.NetworkProfile != nil {
		instance.NetworkProfile = *patch.NetworkProfile
	}
	if patch.Enabled != nil {
		instance.Enabled = *patch.Enabled
	}
	if patch.TrustedOrigins != nil {
		instance.TrustedOrigins = append([]string(nil), (*patch.TrustedOrigins)...)
	}
	if instance.Settings == nil {
		instance.Settings = map[string]IndexerSettingValue{}
	}
	for key, change := range patch.Settings {
		if change.Clear && change.Value != nil {
			return BuiltinIndexerInstance{}, fmt.Errorf("%w: setting cannot be replaced and cleared together", ErrInvalidSettings)
		}
		if change.Clear {
			delete(instance.Settings, key)
		} else if change.Value != nil {
			if err := change.Value.Validate(); err != nil {
				return BuiltinIndexerInstance{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
			}
			instance.Settings[key] = change.Value.clone()
		}
	}
	return instance, nil
}

func (s *Store) RemoveBuiltinInstance(ctx context.Context, id, expected string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, ErrConfigRevision
	}
	if _, ok := s.cfg.Indexers.Builtin.Instances[id]; !ok {
		return nil, ErrConfigRevision
	}
	return s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		deleteMapKey(mapValue(mapValue(indexers, "builtin"), "instances"), id)
		if order := existingMapValue(indexers, "query_order"); order != nil {
			var kept []*yaml.Node
			for _, node := range order.Content {
				if node.Value != id {
					kept = append(kept, node)
				}
			}
			order.Content = kept
		}
		return nil
	})
}

func (s *Store) ReorderIndexers(ctx context.Context, expected string, order []string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, ErrConfigRevision
	}
	if len(order) > 1024 {
		return nil, fmt.Errorf("%w: too many indexers", ErrInvalidSettings)
	}
	seen := map[string]bool{}
	for _, id := range order {
		if id == "" || len(id) > 256 || seen[id] {
			return nil, fmt.Errorf("%w: invalid indexer order", ErrInvalidSettings)
		}
		seen[id] = true
	}
	cfg := s.cfg.Indexers.Clone()
	cfg.QueryOrder = append([]string(nil), order...)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	return s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error { setNodeValue(indexers, "query_order", order); return nil })
}

// SetIndexerNetworkProfile replaces one profile. A referenced profile cannot be
// removed; validation runs before either the file or runtime view changes.
func (s *Store) SetIndexerNetworkProfile(ctx context.Context, expected, id string, profile *IndexerNetworkProfile) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, ErrConfigRevision
	}
	cfg := s.cfg.Indexers.Clone()
	if profile == nil {
		delete(cfg.Builtin.NetworkProfiles, id)
	} else {
		cfg.Builtin.NetworkProfiles[id] = *profile
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	return s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		builtin := ensureMapValue(mapValue(indexers, "builtin"))
		profiles := ensureMapValue(mapValue(builtin, "network_profiles"))
		if profile == nil {
			deleteMapKey(profiles, id)
		} else {
			setNodeValue(profiles, id, *profile)
		}
		return nil
	})
}

func (s *Store) mutateIndexerNodeLocked(mutate func(*yaml.Node) error) (*Config, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(s.saved, &root); err != nil {
		return nil, err
	}
	top := documentNode(&root)
	if existingMapValue(top, "config_version") == nil {
		canonicalizeConfigNode(top)
	} else {
		normalizeIndexerV3Node(top)
	}
	if err := mutate(ensureMapValue(mapValue(top, "indexers"))); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(&root)
	if err != nil {
		return nil, err
	}
	next, err := parseStore(s.path, data)
	if err != nil {
		return nil, err
	}
	if err := s.writeValidatedConfigLocked(next, data); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) SetJackettIndexerEnabled(ctx context.Context, expected, id string, enabled bool) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, ErrConfigRevision
	}
	if id == "jackett" {
		return s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
			jackett := ensureMapValue(mapValue(indexers, "jackett"))
			setNodeValue(jackett, "enabled", enabled)
			return nil
		})
	}
	if !strings.HasPrefix(id, "jackett/") || len(id) <= len("jackett/") || len(id) > 256 {
		return nil, fmt.Errorf("%w: invalid Jackett instance ID", ErrInvalidSettings)
	}
	var excluded []string
	for _, current := range s.cfg.Indexers.Jackett.ExcludedIndexers {
		if current != id && current != strings.TrimPrefix(id, "jackett/") {
			excluded = append(excluded, current)
		}
	}
	if !enabled {
		excluded = append(excluded, id)
	}
	return s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		jackett := ensureMapValue(mapValue(indexers, "jackett"))
		setNodeValue(jackett, "excluded_indexers", excluded)
		return nil
	})
}
