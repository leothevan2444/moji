package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdownloads"
	"net/url"
	"time"
)

type TorrentDownloadService interface {
	Prepare(context.Context, string) (indexerdownloads.Receipt, error)
}

func (r *Resolver) prepareTorrentDownload(ctx context.Context, input model.PrepareTorrentDownloadInput) (*model.PreparedTorrentDownload, error) {
	if r.SearchRuns == nil || r.TorrentDownloads == nil {
		return nil, indexer.ErrNotConfigured
	}
	// Membership and expiry come from the persisted server-owned run. Neither
	// locators nor replacement source metadata are accepted from the browser.
	resource, err := r.SearchRuns.ResourceID(ctx, input.SearchID, input.CandidateID)
	if err != nil {
		return nil, err
	}
	receipt, err := r.TorrentDownloads.Prepare(ctx, resource)
	if err != nil {
		return nil, err
	}
	return &model.PreparedTorrentDownload{Kind: model.PreparedTorrentDownloadKind(receipt.Kind), DownloadURL: "/api/indexers/downloads/" + url.PathEscape(receipt.Handle), ExpiresAt: receipt.ExpiresAt.UTC().Format(time.RFC3339Nano)}, nil
}
