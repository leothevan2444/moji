package config

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
)

type IndexerDefinitionPolicyPatch struct {
	CheckIntervalHours *int
	AutoUpdate         *bool
}

// UpdateIndexerDefinitionPolicy only changes user policy. Installed revisions
// and jobs remain owned by the rule library, while source/schema stay fixed.
func (s *Store) UpdateIndexerDefinitionPolicy(ctx context.Context, expected string, patch IndexerDefinitionPolicyPatch) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	if patch.CheckIntervalHours == nil && patch.AutoUpdate == nil {
		return nil, "", ErrInvalidSettings
	}
	cfg := s.cfg.Indexers.Clone()
	if patch.CheckIntervalHours != nil {
		cfg.Builtin.Definitions.CheckIntervalHours = *patch.CheckIntervalHours
	}
	if patch.AutoUpdate != nil {
		cfg.Builtin.Definitions.AutoUpdate = *patch.AutoUpdate
	}
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("%w: invalid definition update policy", ErrInvalidSettings)
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		definitions := ensureMapValue(mapValue(ensureMapValue(mapValue(indexers, "builtin")), "definitions"))
		if patch.CheckIntervalHours != nil {
			setNodeValue(definitions, "check_interval_hours", *patch.CheckIntervalHours)
		}
		if patch.AutoUpdate != nil {
			setNodeValue(definitions, "auto_update", *patch.AutoUpdate)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}
