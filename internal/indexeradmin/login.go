package indexeradmin

import (
	"context"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
	"github.com/leothevan2444/moji/internal/indexerlogin"
)

// PrepareLogin uses saved-site preparation and retains the exact compiled rule.
// Login does not accept a local draft or mutate saved credentials.
func (s *Service) PrepareLogin(ctx context.Context, input indexerlogin.Input) (indexerlogin.Prepared, error) {
	if input.InstanceID == "" {
		return indexerlogin.Prepared{}, indexer.ErrInvalidScope
	}
	checked, err := s.PrepareCheck(ctx, indexercheck.Input{InstanceID: input.InstanceID, ExpectedRevision: input.ExpectedRevision})
	if err != nil {
		return indexerlogin.Prepared{}, err
	}
	if checked.Release != nil {
		defer checked.Release()
	}
	if checked.Instance.Kind != indexer.Builtin {
		return indexerlogin.Prepared{}, indexer.ErrUnsupported
	}
	lease, err := s.definitions.Acquire(ctx, checked.RuleRevision)
	if err != nil {
		return indexerlogin.Prepared{}, err
	}
	plan := lease.Bundle.Plans[checked.Instance.DefinitionID]
	if plan == nil {
		lease.Release()
		return indexerlogin.Prepared{}, indexer.ErrUnsupported
	}
	return indexerlogin.Prepared{Instance: checked.Instance, ConfigRevision: checked.ConfigRevision, RuleRevision: checked.RuleRevision, Plan: plan, Release: lease.Release}, nil
}
func (s *Service) ValidateLogin(ctx context.Context, revision string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.store == nil {
		return indexer.ErrNotConfigured
	}
	_, current := s.store.Snapshot()
	if revision == "" || current != revision {
		return config.ErrConfigRevision
	}
	return nil
}
