package indexeradmin

import (
	"context"
	"sort"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type NetworkProfile struct {
	ID                     string
	ProxyConfigured        bool
	FlareSolverrConfigured bool
	UsedBy                 int
}
type NetworkSettings struct {
	Revision                                                           string
	RequestTimeoutSeconds, SiteTimeoutSeconds, ChallengeTimeoutSeconds int
	Profiles                                                           []NetworkProfile
}

func networkProjection(cfg *config.Config, revision string) NetworkSettings {
	result := NetworkSettings{Revision: revision, ChallengeTimeoutSeconds: cfg.Indexers.Builtin.ChallengeTimeoutSeconds, RequestTimeoutSeconds: cfg.Indexers.Builtin.RequestTimeoutSeconds, SiteTimeoutSeconds: cfg.Indexers.Builtin.SiteTimeoutSeconds, Profiles: []NetworkProfile{}}
	used := map[string]int{}
	for _, instance := range cfg.Indexers.Builtin.Instances {
		id := instance.NetworkProfile
		if id == "" {
			id = "direct"
		}
		used[id]++
	}
	for id, profile := range cfg.Indexers.Builtin.NetworkProfiles {
		result.Profiles = append(result.Profiles, NetworkProfile{ID: id, FlareSolverrConfigured: profile.FlareSolverrURL != "", ProxyConfigured: profile.ProxyURL != "", UsedBy: used[id]})
	}
	sort.Slice(result.Profiles, func(i, j int) bool { return result.Profiles[i].ID < result.Profiles[j].ID })
	return result
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
func (s *Service) UpdateNetworkTimeouts(ctx context.Context, expected string, patch config.IndexerNetworkTimeoutPatch) (NetworkSettings, error) {
	if s.store == nil {
		return NetworkSettings{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.UpdateIndexerNetworkTimeouts(ctx, expected, patch)
	if err != nil {
		return NetworkSettings{}, err
	}
	return networkProjection(cfg, revision), nil
}
func (s *Service) SaveNetworkProfile(ctx context.Context, expected, id string, create bool, patch config.IndexerNetworkProfilePatch) (NetworkSettings, error) {
	if s.store == nil {
		return NetworkSettings{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.SaveIndexerNetworkProfile(ctx, expected, id, create, patch)
	if err != nil {
		return NetworkSettings{}, err
	}
	return networkProjection(cfg, revision), nil
}
func (s *Service) RemoveNetworkProfile(ctx context.Context, expected, id string) (NetworkSettings, error) {
	if s.store == nil {
		return NetworkSettings{}, indexer.ErrNotConfigured
	}
	cfg, revision, err := s.store.RemoveIndexerNetworkProfile(ctx, expected, id)
	if err != nil {
		return NetworkSettings{}, err
	}
	return networkProjection(cfg, revision), nil
}
