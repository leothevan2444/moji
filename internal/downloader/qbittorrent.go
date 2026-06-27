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

// DefaultsProvider supplies the latest qBittorrent defaults at the moment of
// each AddNewTorrent call. Returning the values lazily (instead of capturing
// them at construction time) lets Web UI edits take effect on the next
// download without restarting the service.
type DefaultsProvider func() TorrentDefaults

type DefaultingTorrentClient struct {
	client           TorrentClient
	defaultsProvider DefaultsProvider
}

func NewDefaultingTorrentClient(client TorrentClient, defaultsProvider DefaultsProvider) *DefaultingTorrentClient {
	return &DefaultingTorrentClient{
		client:           client,
		defaultsProvider: defaultsProvider,
	}
}

func (c *DefaultingTorrentClient) GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return c.client.GetTorrentList(ctx, options)
}

func (c *DefaultingTorrentClient) AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error {
	if c.defaultsProvider != nil {
		defaults := c.defaultsProvider()
		if opts.SavePath == nil && defaults.SavePath != "" {
			savePath := defaults.SavePath
			opts.SavePath = &savePath
		}
		if opts.Category == nil && defaults.Category != "" {
			category := defaults.Category
			opts.Category = &category
		}
		if opts.Tags == nil && defaults.Tags != "" {
			tags := defaults.Tags
			opts.Tags = &tags
		}
	}

	return c.client.AddNewTorrent(ctx, opts)
}

func (c *DefaultingTorrentClient) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	return c.client.DeleteTorrents(ctx, hashes, deleteFiles)
}
