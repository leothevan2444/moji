package indexeradmin

import (
	"context"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type DefinitionPolicy struct {
	Revision, Source                  string
	SchemaVersion, CheckIntervalHours int
	AutoUpdate                        bool
}

func definitionPolicyProjection(cfg *config.Config, revision string) DefinitionPolicy {
	p := cfg.Indexers.Builtin.Definitions
	return DefinitionPolicy{Revision: revision, Source: p.Source, SchemaVersion: p.SchemaVersion, CheckIntervalHours: p.CheckIntervalHours, AutoUpdate: p.AutoUpdate}
}
func (s *Service) DefinitionPolicy(ctx context.Context) (DefinitionPolicy, error) {
	if err := ctx.Err(); err != nil {
		return DefinitionPolicy{}, err
	}
	if s.store == nil {
		return DefinitionPolicy{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	return definitionPolicyProjection(cfg, revision), nil
}
func (s *Service) UpdateDefinitionPolicy(ctx context.Context, expected string, patch config.IndexerDefinitionPolicyPatch) (DefinitionPolicy, error) {
	if s.store == nil {
		return DefinitionPolicy{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.UpdateIndexerDefinitionPolicy(ctx, expected, patch)
	if err != nil {
		return DefinitionPolicy{}, err
	}
	return definitionPolicyProjection(cfg, revision), nil
}
