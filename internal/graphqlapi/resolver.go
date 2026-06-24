package graphqlapi

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/stats"
	"github.com/leothevan2444/moji/internal/subscription"
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
	CurrentConfig() stashsync.IntegrationConfig
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
	StatusSnapshot() *SettingsStatusSnapshot
	UpdateStashSettings(input UpdateStashSettingsInput) (*SettingsSnapshot, error)
	UpdateIngestSettings(input UpdateIngestSettingsInput) (*SettingsSnapshot, error)
	UpdateJackettSettings(input UpdateJackettSettingsInput) (*SettingsSnapshot, error)
	UpdateQBittorrentSettings(input UpdateQBittorrentSettingsInput) (*SettingsSnapshot, error)
	UpdateAutomationSettings(input UpdateAutomationSettingsInput) (*SettingsSnapshot, error)
	UpdateSubscriptionSettings(input UpdateSubscriptionSettingsInput) (*SettingsSnapshot, error)
}

type UpdateStashSettingsInput struct {
	URL    string
	APIKey string
}

type UpdateIngestSettingsInput struct {
	DeliveryMode     string
	StashLibraryPath string
	Transfer         TransferIngestSettingsSnapshot
}

type UpdateJackettSettingsInput struct {
	URL      string
	APIKey   string
	Password string
}

type UpdateQBittorrentSettingsInput struct {
	URL             string
	Username        string
	Password        string
	DefaultSavePath string
	Category        string
	Tags            string
}

type UpdateSubscriptionSettingsInput struct {
	StashBoxEndpoints []string
}

type UpdateAutomationSettingsInput struct {
	TaskProgressSyncIntervalSeconds int
	SubscriptionPollIntervalSeconds int
}

type SettingsSnapshot struct {
	Stash        StashSettingsSnapshot
	Ingest       IngestSettingsSnapshot
	Jackett      JackettSettingsSnapshot
	QBittorrent  QBittorrentSettingsSnapshot
	Automation   AutomationSettingsSnapshot
	Subscription SubscriptionSettingsSnapshot
}

type StashSettingsSnapshot struct {
	Configured       bool
	URL              string
	APIKeyConfigured bool
	APIKey           string
}

type IngestSettingsSnapshot struct {
	DeliveryMode     string
	StashLibraryPath string
	Transfer         TransferIngestSettingsSnapshot
}

type TransferIngestSettingsSnapshot struct {
	Action         string
	MojiSourceRoot string
	MojiTargetRoot string
}

type JackettSettingsSnapshot struct {
	Configured         bool
	URL                string
	APIKeyConfigured   bool
	APIKey             string
	PasswordConfigured bool
	Password           string
}

type QBittorrentSettingsSnapshot struct {
	Configured         bool
	URL                string
	Username           string
	UsernameConfigured bool
	PasswordConfigured bool
	Password           string
	DefaultSavePath    string
	Category           string
	Tags               string
}

type AutomationSettingsSnapshot struct {
	TaskProgressSyncIntervalSeconds int
	SubscriptionPollIntervalSeconds int
}

type SubscriptionSettingsSnapshot struct {
	StashBoxEndpoints []string
}

type StashLibrarySnapshot struct {
	Path string
}

type StashBoxEndpointSnapshot struct {
	Name             string
	Endpoint         string
	APIKeyConfigured bool
}

type SettingsStatusSnapshot struct {
	Stash                   ServiceStatusSnapshot
	Jackett                 ServiceStatusSnapshot
	QBittorrent             ServiceStatusSnapshot
	Automation              AutomationStatusSnapshot
	Subscription            SubscriptionStatusSnapshot
	Ingest                  IngestStatusSnapshot
	StashLibraries          []StashLibrarySnapshot
	StashLibrariesLoadError string
}

type IngestStatusSnapshot struct {
	Configured bool
}

type ServiceStatusSnapshot struct {
	Configured bool
	Ready      bool
}

type AutomationStatusSnapshot struct {
	TaskProgressSyncIntervalSeconds int
	TaskProgressSyncEnabled         bool
	SubscriptionPollIntervalSeconds int
	SubscriptionPollEnabled         bool
}

type SubscriptionStatusSnapshot struct {
	StashBoxes          []StashBoxEndpointSnapshot
	StashBoxesLoaded    bool
	StashBoxesLoadError string
}

type StashPerformerPage struct {
	Items       []subscription.Performer
	Page        int
	PageSize    int
	TotalCount  int
	TotalPages  int
	HasPrevPage bool
	HasNextPage bool
}

type StatsProvider interface {
	SnapshotView() stats.Snapshot
}

type SubscriptionService interface {
	ListStashPerformers(ctx context.Context, search string) ([]subscription.Performer, error)
	ListSubscribedPerformers(ctx context.Context) ([]subscription.SubscribedPerformer, error)
	SubscribePerformer(ctx context.Context, performerID string) (subscription.SubscribedPerformer, error)
	UnsubscribePerformer(ctx context.Context, performerID string) error
	RefreshSubscribedPerformer(ctx context.Context, performerID string) (subscription.SubscribedPerformer, error)
	RefreshAll(ctx context.Context) ([]subscription.SubscribedPerformer, error)
	RefreshStashBoxes(ctx context.Context) error
	SnapshotState() (endpoints []subscription.StashBoxEndpoint, state subscription.LoadState)
}

type LogReader interface {
	Entries(limit int, minLevel string) []logging.Entry
}

type Resolver struct {
	Tracker         tracker.Tracker
	Torrent         TorrentClient
	Downloader      DownloaderService
	Stash           StashService
	Subscription    SubscriptionService
	LogReader       LogReader
	SettingsEditor  SettingsEditor
	RuntimeSettings *SettingsSnapshot
	RuntimeStatus   *SettingsStatusSnapshot
	Stats           StatsProvider
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
