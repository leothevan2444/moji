package downloader

import (
	"context"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentDefaults struct {
	SavePath string
	Category string
	Tags     string
}

type DefaultingTorrentClient struct {
	client   TorrentClient
	defaults TorrentDefaults
}

func NewDefaultingTorrentClient(client TorrentClient, defaults TorrentDefaults) *DefaultingTorrentClient {
	return &DefaultingTorrentClient{
		client:   client,
		defaults: defaults,
	}
}

func (c *DefaultingTorrentClient) GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return c.client.GetTorrentList(ctx, options)
}

func (c *DefaultingTorrentClient) AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error {
	if opts.SavePath == nil && c.defaults.SavePath != "" {
		opts.SavePath = &c.defaults.SavePath
	}
	if opts.Category == nil && c.defaults.Category != "" {
		opts.Category = &c.defaults.Category
	}
	if opts.Tags == nil && c.defaults.Tags != "" {
		opts.Tags = &c.defaults.Tags
	}

	return c.client.AddNewTorrent(ctx, opts)
}
