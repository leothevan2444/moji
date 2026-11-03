package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type IndexerNetworkService interface {
	NetworkSettings(context.Context) (indexeradmin.NetworkSettings, error)
	UpdateNetworkTimeouts(context.Context, string, config.IndexerNetworkTimeoutPatch) (indexeradmin.NetworkSettings, error)
	SaveNetworkProfile(context.Context, string, string, bool, config.IndexerNetworkProfilePatch) (indexeradmin.NetworkSettings, error)
	RemoveNetworkProfile(context.Context, string, string) (indexeradmin.NetworkSettings, error)
}

func indexerNetworkModel(snapshot indexeradmin.NetworkSettings, err error) (*model.IndexerNetworkSettings, error) {
	if err != nil {
		return nil, err
	}
	result := &model.IndexerNetworkSettings{Revision: snapshot.Revision, ChallengeTimeoutSeconds: snapshot.ChallengeTimeoutSeconds, RequestTimeoutSeconds: snapshot.RequestTimeoutSeconds, SiteTimeoutSeconds: snapshot.SiteTimeoutSeconds, Profiles: []*model.IndexerNetworkProfile{}}
	for _, profile := range snapshot.Profiles {
		result.Profiles = append(result.Profiles, &model.IndexerNetworkProfile{ID: profile.ID, FlareSolverrConfigured: profile.FlareSolverrConfigured, ProxyConfigured: profile.ProxyConfigured, UsedBy: profile.UsedBy})
	}
	return result, nil
}
func (r *Resolver) indexerNetworkSettings(ctx context.Context) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.NetworkSettings(ctx))
}
func (r *Resolver) updateIndexerNetworkTimeouts(ctx context.Context, expected string, input model.IndexerNetworkTimeoutPatchInput) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.UpdateNetworkTimeouts(ctx, expected, config.IndexerNetworkTimeoutPatch{RequestTimeoutSeconds: input.RequestTimeoutSeconds, SiteTimeoutSeconds: input.SiteTimeoutSeconds, ChallengeTimeoutSeconds: input.ChallengeTimeoutSeconds}))
}
func (r *Resolver) saveIndexerNetworkProfile(ctx context.Context, id, expected string, create bool, input model.IndexerNetworkProfilePatchInput) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.SaveNetworkProfile(ctx, expected, id, create, config.IndexerNetworkProfilePatch{FlareSolverrURL: input.FlareSolverrURL, ClearFlareSolverr: input.ClearFlareSolverr != nil && *input.ClearFlareSolverr, ProxyURL: input.ProxyURL, ClearProxy: input.ClearProxy != nil && *input.ClearProxy}))
}
func (r *Resolver) removeIndexerNetworkProfile(ctx context.Context, id, expected string) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.RemoveNetworkProfile(ctx, expected, id))
}
