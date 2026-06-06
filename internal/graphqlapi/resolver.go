package graphqlapi

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentClient interface {
	GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error)
	AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error
}

type Resolver struct {
	Tracker    tracker.Tracker
	Torrent    TorrentClient
	AppVersion string
}

func NewResolver(tr tracker.Tracker, torrent TorrentClient, version string) *Resolver {
	return &Resolver{
		Tracker:    tr,
		Torrent:    torrent,
		AppVersion: version,
	}
}
