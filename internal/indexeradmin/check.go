package indexeradmin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
)

// PrepareCheck merges the actual draft with saved secrets under one revision
// snapshot, validates without writing, and pins the rules used by the check.
func (s *Service) PrepareCheck(ctx context.Context, input indexercheck.Input) (indexercheck.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return indexercheck.Prepared{}, err
	}
	if s.store == nil {
		return indexercheck.Prepared{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	selective := input.ConnectionOnly && input.ExpectedConnectionRevision != "" && input.Draft == nil && input.InstanceID != ""
	if input.ExpectedRevision == "" || !selective && input.ExpectedRevision != revision {
		return indexercheck.Prepared{}, config.ErrConfigRevision
	}
	if strings.HasPrefix(input.InstanceID, "jackett/") {
		if input.Draft != nil {
			return indexercheck.Prepared{}, indexer.ErrUnsupported
		}
		if s.registry == nil {
			return indexercheck.Prepared{}, indexer.ErrNotConfigured
		}
		registry, err := s.registry.CachedSnapshot(ctx)
		if err != nil {
			return indexercheck.Prepared{}, err
		}
		if registry.Revision != revision {
			return indexercheck.Prepared{}, config.ErrConfigRevision
		}
		for _, instance := range registry.Instances {
			if instance.ID == input.InstanceID && instance.Kind == indexer.Jackett {
				if selective && input.ExpectedConnectionRevision != s.connectionRevision(instance.ConfigRevision, "") {
					return indexercheck.Prepared{}, config.ErrConfigRevision
				}
				return indexercheck.Prepared{Instance: instance, ConfigRevision: revision}, nil
			}
		}
		return indexercheck.Prepared{}, indexer.ErrInvalidScope
	}
	candidate, exists := cfg.Indexers.Builtin.Instances[input.InstanceID]
	if input.InstanceID != "" && !exists {
		return indexercheck.Prepared{}, indexer.ErrInvalidScope
	}
	var err error
	if input.Draft != nil {
		candidate, err = config.MergeBuiltinInstance(candidate, *input.Draft)
		if err != nil {
			return indexercheck.Prepared{}, err
		}
	}
	id := input.InstanceID
	if id == "" {
		id = uuid.NewString()
	}
	proposed := cfg.Indexers.Clone()
	proposed.Builtin.Instances[id] = candidate
	if err = proposed.Validate(); err != nil {
		return indexercheck.Prepared{}, fmt.Errorf("%w: invalid diagnostic settings", config.ErrInvalidSettings)
	}
	if s.definitions == nil {
		return indexercheck.Prepared{}, indexer.ErrNotConfigured
	}
	lease, err := s.definitions.Acquire(ctx, "")
	if err != nil {
		return indexercheck.Prepared{}, err
	}
	retained := false
	defer func() {
		if !retained {
			lease.Release()
		}
	}()
	plan := lease.Bundle.Plans[candidate.Definition]
	if plan == nil {
		return indexercheck.Prepared{}, indexer.ErrUnsupported
	}
	instance := BuiltinSnapshot(cfg, id, candidate)
	instance.BundleRevision = lease.Bundle.Manifest.Revision
	if selective && input.ExpectedConnectionRevision != s.connectionRevision(instance.ConfigRevision, plan.Digest) {
		return indexercheck.Prepared{}, config.ErrConfigRevision
	}
	if err = ValidateSettings(plan, nil, candidate); err != nil {
		return indexercheck.Prepared{}, fmt.Errorf("%w: invalid diagnostic definition settings", config.ErrInvalidSettings)
	}
	retained = true
	return indexercheck.Prepared{Instance: instance, ConfigRevision: revision, RuleRevision: instance.BundleRevision, AuthenticationRequired: plan.Definition.Login != nil, Release: lease.Release}, nil
}
