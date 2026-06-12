package graphqlapi

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/following"
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

type DownloaderService interface {
	AddTorrentContext(ctx context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error)
	DownloadMediaContext(ctx context.Context, req downloader.DownloadRequest) (*downloader.Task, error)
	FindTask(ctx context.Context, id string) (*downloader.Task, error)
	ListTasks(ctx context.Context) ([]*downloader.Task, error)
	SyncProgress(ctx context.Context) ([]*downloader.Task, error)
	TriggerTaskStashScan(ctx context.Context, id string, scanner downloader.StashScanner) (*downloader.Task, error)
	TriggerStashScans(ctx context.Context, scanner downloader.StashScanner) ([]*downloader.Task, error)
}

type SettingsEditor interface {
	Snapshot() *SettingsSnapshot
	UpdateStashSettings(input UpdateStashSettingsInput) (*SettingsSnapshot, error)
	UpdateJackettSettings(input UpdateJackettSettingsInput) (*SettingsSnapshot, error)
	UpdateQBittorrentSettings(input UpdateQBittorrentSettingsInput) (*SettingsSnapshot, error)
}

type UpdateStashSettingsInput struct {
	URL         string
	APIKey      *string
	LibraryPath string
}

type UpdateJackettSettingsInput struct {
	URL    string
	APIKey *string
}

type UpdateQBittorrentSettingsInput struct {
	URL             string
	Username        string
	Password        *string
	DefaultSavePath string
	Category        string
	Tags            string
}

type SettingsSnapshot struct {
	Stash       StashSettingsSnapshot
	Jackett     JackettSettingsSnapshot
	QBittorrent QBittorrentSettingsSnapshot
	Tasks       TaskSettingsSnapshot
	System      SystemSettingsSnapshot
}

type StashSettingsSnapshot struct {
	Configured       bool
	Enabled          bool
	URL              string
	APIKeyConfigured bool
	LibraryPath      string
}

type JackettSettingsSnapshot struct {
	Configured       bool
	Enabled          bool
	URL              string
	APIKeyConfigured bool
}

type QBittorrentSettingsSnapshot struct {
	Configured         bool
	Enabled            bool
	URL                string
	Username           string
	UsernameConfigured bool
	PasswordConfigured bool
	DefaultSavePath    string
	Category           string
	Tags               string
}

type TaskSettingsSnapshot struct {
	Store                       string
	JSONPath                    string
	ProgressSyncIntervalSeconds int
	ProgressSyncEnabled         bool
}

type SystemSettingsSnapshot struct {
	AppVersion string
}

type FollowingService interface {
	ListStashPerformers(ctx context.Context, search string) ([]following.Performer, error)
	ListFollowingPerformers(ctx context.Context) ([]following.FollowingPerformer, error)
	FollowPerformer(ctx context.Context, performerID string) (following.FollowingPerformer, error)
	UnfollowPerformer(ctx context.Context, performerID string) error
	RefreshPerformer(ctx context.Context, performerID string) (following.FollowingPerformer, error)
	RefreshAll(ctx context.Context) ([]following.FollowingPerformer, error)
}

type Resolver struct {
	Tracker         tracker.Tracker
	Torrent         TorrentClient
	Downloader      DownloaderService
	Stash           StashService
	Following       FollowingService
	SettingsEditor  SettingsEditor
	RuntimeSettings *SettingsSnapshot
	AppVersion      string
}

func NewResolver(tr tracker.Tracker, torrent TorrentClient, downloader DownloaderService, stash StashService, version string) *Resolver {
	return &Resolver{
		Tracker:    tr,
		Torrent:    torrent,
		Downloader: downloader,
		Stash:      stash,
		AppVersion: version,
	}
}
