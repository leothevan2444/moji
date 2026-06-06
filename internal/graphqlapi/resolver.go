package graphqlapi

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"

	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentClient interface {
	GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error)
	AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error
}

type StashService interface {
	MetadataScan(ctx context.Context, req stashsync.ScanRequest) (string, error)
	FindJob(ctx context.Context, id string) (*stashsync.Job, error)
}

type Resolver struct {
	Tracker    tracker.Tracker
	Torrent    TorrentClient
	Stash      StashService
	AppVersion string
}

func NewResolver(tr tracker.Tracker, torrent TorrentClient, stash StashService, version string) *Resolver {
	return &Resolver{
		Tracker:    tr,
		Torrent:    torrent,
		Stash:      stash,
		AppVersion: version,
	}
}
