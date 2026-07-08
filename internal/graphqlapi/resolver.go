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
	DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error
}

type StashService interface {
	MetadataScan(ctx context.Context, req stashsync.ScanRequest) (string, error)
	FindJob(ctx context.Context, id string) (*stashsync.Job, error)
	CurrentConfig() stashsync.IntegrationConfig
}

type DownloaderService interface {
	AddTorrentContext(ctx context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error)
	DownloadMediaContext(ctx context.Context, req downloader.DownloadRequest) (*downloader.Task, error)
	PreviewJackettSelectionContext(ctx context.Context, req downloader.PreviewJackettSelectionRequest) (*downloader.CandidateSelectionPreview, error)
	FindTask(ctx context.Context, id string) (*downloader.Task, error)
	ListTasks(ctx context.Context) ([]*downloader.Task, error)
	DeleteTask(ctx context.Context, id string) (*downloader.Task, error)
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
	UpdateSystemSettings(input UpdateSystemSettingsInput) (*SettingsSnapshot, error)
}

type UpdateStashSettingsInput struct {
	URL    string
	APIKey string
}

type UpdateIngestSettingsInput struct {
	DeliveryMode string
	Downloads    DownloadsIngestSettingsSnapshot
	Library      LibraryIngestSettingsSnapshot
	Transfer     TransferIngestSettingsSnapshot
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

type UpdateAutomationSettingsInput struct {
	TaskProgressSyncIntervalSeconds int
	SubscriptionPollIntervalHours   int
	StashBoxEndpoints               []string
	SubscriptionReleasePolicy       SubscriptionReleasePolicySnapshot
	TorrentSelection                TorrentSelectionSettingsSnapshot
}

type UpdateSystemSettingsInput struct {
	TaskDeletePolicy string
}

type SettingsSnapshot struct {
	Stash       StashSettingsSnapshot
	Ingest      IngestSettingsSnapshot
	Jackett     JackettSettingsSnapshot
	QBittorrent QBittorrentSettingsSnapshot
	Automation  AutomationSettingsSnapshot
	System      SystemSettingsSnapshot
}

type StashSettingsSnapshot struct {
	Configured       bool
	URL              string
	APIKeyConfigured bool
	APIKey           string
}

type IngestSettingsSnapshot struct {
	DeliveryMode string
	Downloads    DownloadsIngestSettingsSnapshot
	Library      LibraryIngestSettingsSnapshot
	Transfer     TransferIngestSettingsSnapshot
}

type DownloadsIngestSettingsSnapshot struct {
	QBRoot   string
	MojiRoot string
}

type LibraryIngestSettingsSnapshot struct {
	MojiRoot  string
	StashRoot string
}

type TransferIngestSettingsSnapshot struct {
	Action string
}

type JackettSettingsSnapshot struct {
	Configured         bool
	URL                string
	APIKeyConfigured   bool
	APIKey             string
	PasswordConfigured bool
	Password           string
}

type TorrentSelectionSettingsSnapshot struct {
	Enabled                  bool
	InspectionCandidateLimit int
	FastRules                []TorrentSelectionRuleSnapshot
	TorrentRules             []TorrentSelectionRuleSnapshot
}

type TorrentSelectionRuleSnapshot struct {
	Type                 string
	Enabled              bool
	IndexerPreference    IndexerPreferenceRuleSnapshot
	TitleMatch           TitleMatchRuleSnapshot
	PublishDate          DirectionRuleSnapshot
	Seeders              DirectionRuleSnapshot
	Size                 DirectionRuleSnapshot
	TorrentFileNameMatch TorrentFileNameMatchRuleSnapshot
}

type DirectionRuleSnapshot struct {
	Direction string
}

type IndexerPreferenceRuleSnapshot struct {
	TrackerIDs []string
}

type TitleMatchRuleSnapshot struct {
	Clauses []TitleMatchClauseSnapshot
}

type TitleMatchClauseSnapshot struct {
	Pattern     string
	PatternMode string
	Effect      string
}

type TorrentFileNameMatchRuleSnapshot struct {
	Clauses []TorrentFileNameMatchClauseSnapshot
}

type TorrentFileNameMatchClauseSnapshot struct {
	Pattern     string
	PatternMode string
	Effect      string
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
	SubscriptionPollIntervalHours   int
	StashBoxEndpoints               []string
	SubscriptionReleasePolicy       SubscriptionReleasePolicySnapshot
	TorrentSelection                TorrentSelectionSettingsSnapshot
}

type SubscriptionReleasePolicySnapshot struct {
	SoloBehavior           string
	GroupBehavior          string
	CompilationBehavior    string
	MaxGroupPerformerCount int
	ReleaseDateRange       string
}

type SystemSettingsSnapshot struct {
	TaskDeletePolicy string
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
	SubscriptionPollIntervalHours   int
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
	SearchPreferredStashBoxScenes(ctx context.Context, query string, limit int, sortBy subscription.DiscoverSort) (subscription.DiscoverScenePage, error)
	QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*downloader.Task, error)
	GetPerformerDetail(ctx context.Context, performerID string) (subscription.PerformerDetail, error)
	ListPerformerScenes(ctx context.Context, performerID string, query subscription.PerformerSceneQuery) (subscription.PerformerScenePage, error)
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
