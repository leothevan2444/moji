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
	UpdateNetwork(context.Context, string, config.IndexerNetworkPatch) (indexeradmin.NetworkSettings, error)
}

func indexerNetworkModel(snapshot indexeradmin.NetworkSettings, err error) (*model.IndexerNetworkSettings, error) {
	if err != nil {
		return nil, err
	}
	result := &model.IndexerNetworkSettings{FlareSolverrMaxTimeoutMs: snapshot.FlareSolverrMaxTimeoutMS, Revision: snapshot.Revision, RequestTimeoutSeconds: snapshot.RequestTimeoutSeconds, SiteTimeoutSeconds: snapshot.SiteTimeoutSeconds, ProxyEnabled: snapshot.ProxyEnabled, ProxyURL: snapshot.ProxyURL, ProxyUsername: snapshot.ProxyUsername, ProxyPasswordConfigured: snapshot.ProxyPasswordConfigured, FlareSolverrEnabled: snapshot.FlareSolverrEnabled, FlareSolverrURL: snapshot.FlareSolverrURL}
	return result, nil
}
func (r *Resolver) indexerNetworkSettings(ctx context.Context) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.NetworkSettings(ctx))
}
func (r *Resolver) updateIndexerNetwork(ctx context.Context, expected string, input model.IndexerNetworkPatchInput) (*model.IndexerNetworkSettings, error) {
	if r.IndexerNetwork == nil {
		return nil, indexer.ErrNotConfigured
	}
	return indexerNetworkModel(r.IndexerNetwork.UpdateNetwork(ctx, expected, config.IndexerNetworkPatch{FlareSolverrMaxTimeoutMS: input.FlareSolverrMaxTimeoutMs, RequestTimeoutSeconds: input.RequestTimeoutSeconds, SiteTimeoutSeconds: input.SiteTimeoutSeconds, ProxyEnabled: input.ProxyEnabled, ProxyURL: input.ProxyURL, ProxyUsername: input.ProxyUsername, ProxyPassword: input.ProxyPassword, ClearProxyPassword: input.ClearProxyPassword != nil && *input.ClearProxyPassword, FlareSolverrEnabled: input.FlareSolverrEnabled, FlareSolverrURL: input.FlareSolverrURL}))
}
