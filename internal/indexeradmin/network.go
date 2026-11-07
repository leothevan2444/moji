package indexeradmin

import (
	"context"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type NetworkSettings struct {
	FlareSolverrMaxTimeoutMS                                   int
	Revision                                                   string
	RequestTimeoutSeconds, SiteTimeoutSeconds                  int
	ProxyEnabled, FlareSolverrEnabled, ProxyPasswordConfigured bool
	ProxyURL, ProxyUsername, FlareSolverrURL                   string
}

func networkProjection(cfg *config.Config, revision string) NetworkSettings {
	n := cfg.Indexers.Builtin.Network
	return NetworkSettings{FlareSolverrMaxTimeoutMS: n.FlareSolverr.MaxTimeoutMS, Revision: revision, RequestTimeoutSeconds: cfg.Indexers.Builtin.RequestTimeoutSeconds, SiteTimeoutSeconds: cfg.Indexers.Builtin.SiteTimeoutSeconds, ProxyEnabled: n.Proxy.Enabled, ProxyURL: n.Proxy.URL, ProxyUsername: n.Proxy.Username, ProxyPasswordConfigured: n.Proxy.Password != "", FlareSolverrEnabled: n.FlareSolverr.Enabled, FlareSolverrURL: n.FlareSolverr.URL}
}
func (s *Service) NetworkSettings(ctx context.Context) (NetworkSettings, error) {
	if err := ctx.Err(); err != nil {
		return NetworkSettings{}, err
	}
	if s.store == nil {
		return NetworkSettings{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	return networkProjection(cfg, revision), nil
}
func (s *Service) UpdateNetwork(ctx context.Context, expected string, patch config.IndexerNetworkPatch) (NetworkSettings, error) {
	if s.store == nil {
		return NetworkSettings{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.UpdateIndexerNetwork(ctx, expected, patch)
	if err != nil {
		return NetworkSettings{}, err
	}
	return networkProjection(cfg, revision), nil
}
