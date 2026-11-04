package indexer

import "context"

// FetchCurrentReference admits a new user operation against the current source.
// FetchReference/FetchRecord retain snapshots for work already in progress.
func (s *Service) FetchCurrentReference(ctx context.Context, id string) (Artifact, error) {
	lease, err := s.ResolveReference(ctx, id)
	if err != nil {
		return Artifact{}, err
	}
	defer lease.Release()
	return s.FetchRecord(ctx, lease.Record)
}

func (s *Service) validateCurrentResource(ctx context.Context, record ResourceRecord) error {
	var current InstanceSnapshot
	if provider, ok := s.provider.(interface {
		ReferenceSnapshot(context.Context, string) (InstanceSnapshot, error)
	}); ok {
		var err error
		current, err = provider.ReferenceSnapshot(ctx, record.Ref.InstanceID)
		if err != nil {
			return err
		}
	} else {
		var registry Registry
		var err error
		if provider, ok := s.provider.(interface {
			CachedSnapshot(context.Context) (Registry, error)
		}); ok {
			registry, err = provider.CachedSnapshot(ctx)
		} else {
			registry, err = s.provider.Snapshot(ctx)
		}
		if err != nil {
			return err
		}
		instances, err := orderedInstances(registry, []string{record.Ref.InstanceID})
		if err != nil || len(instances) != 1 {
			return ErrReference
		}
		current = instances[0]
	}
	if !current.Enabled || current.ID != record.Instance.ID || current.Kind != record.Instance.Kind || current.ConfigRevision != record.Ref.ConfigRevision || current.DefinitionID != record.Instance.DefinitionID || current.RemoteID != record.Instance.RemoteID {
		return ErrReference
	}
	if current.Kind == Builtin {
		if s.bundles == nil {
			return ErrNotConfigured
		}
		lease, err := s.bundles.Acquire(ctx, "")
		if err != nil {
			return err
		}
		defer lease.Release()
		plan := lease.Bundle.Plans[current.DefinitionID]
		if plan == nil || plan.Digest != record.Ref.DefinitionDigest {
			return ErrReference
		}
	}
	return ctx.Err()
}
