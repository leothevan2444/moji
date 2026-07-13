package graphqlapi

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"

	"github.com/leothevan2444/moji/internal/discovery"
	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/stats"
	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
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

type TaskRuntimeService interface {
	AddTorrentContext(ctx context.Context, req taskruntime.AddTorrentRequest) (*taskruntime.Task, error)
	DownloadMediaContext(ctx context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error)
	PreviewJackettSelectionContext(ctx context.Context, req taskruntime.PreviewJackettSelectionRequest) (*taskruntime.CandidateSelectionPreview, error)
	FindTask(ctx context.Context, id string) (*taskruntime.Task, error)
	ListTasks(ctx context.Context) ([]*taskruntime.Task, error)
	DeleteTask(ctx context.Context, id string) (*taskruntime.Task, error)
	RetryTask(ctx context.Context, id string, scanner taskruntime.StashScanner) (*taskruntime.Task, error)
	ResolveBlockedSourcingTask(ctx context.Context, id string, req taskruntime.ResolveBlockedSourcingRequest) (*taskruntime.Task, error)
	SyncProgress(ctx context.Context) ([]*taskruntime.Task, error)
	TriggerTaskStashScan(ctx context.Context, id string, scanner taskruntime.StashScanner) (*taskruntime.Task, error)
	TriggerStashScans(ctx context.Context, scanner taskruntime.StashScanner) ([]*taskruntime.Task, error)
}

// TaskFlowService is the only GraphQL seam allowed to create new tasks. Querying
// existing tasks and advancing task execution stages still belongs to the
// task runtime execution service.
type TaskFlowService interface {
	CreateFromManualTorrent(ctx context.Context, input taskflow.CreateFromManualTorrentInput) (*taskruntime.Task, error)
	CreateFromSearchCode(ctx context.Context, input taskflow.CreateFromSearchCodeInput) (*taskruntime.Task, error)
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

type ImageCacheService interface {
	Status(context.Context) (imagecache.Status, error)
	Clear(context.Context) (imagecache.Status, error)
	Cleanup(context.Context) error
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
	ImageCache       ImageCacheSettingsSnapshot
}

type ImageCacheSettingsSnapshot struct {
	Enabled       bool
	MaxSizeMB     int
	RetentionDays int
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
	ImageCache       ImageCacheSettingsSnapshot
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
	StashBox                StashBoxStatusSnapshot
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

type StashBoxStatusSnapshot struct {
	StashBoxes          []StashBoxEndpointSnapshot
	StashBoxesLoaded    bool
	StashBoxesLoadError string
}

type StashPerformerPage struct {
	Items       []performer.Performer
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

type PerformerService interface {
	List(ctx context.Context, search string) ([]performer.Performer, error)
	QueuePerformerScenes(ctx context.Context, performerID string, selections []performer.QueueSceneSelection) (performer.QueueScenesResult, error)
	GetPerformerDetail(ctx context.Context, performerID string) (performer.Detail, error)
	ListPerformerScenes(ctx context.Context, performerID string, query performer.SceneQuery) (performer.ScenePage, error)
}

type DiscoveryService interface {
	Search(ctx context.Context, query string, limit int, sortBy discovery.Sort) (discovery.Page, error)
	Queue(ctx context.Context, sceneID string, stashBoxEndpoint string) (*taskruntime.Task, error)
}

type SubscriptionService interface {
	ListSubscribedPerformers(ctx context.Context) ([]subscription.SubscribedPerformer, error)
	SubscribePerformer(ctx context.Context, performerID string) (subscription.SubscribedPerformer, error)
	UnsubscribePerformer(ctx context.Context, performerID string) error
	RefreshSubscribedPerformer(ctx context.Context, performerID string) (subscription.SubscribedPerformer, error)
	RefreshAll(ctx context.Context) ([]subscription.SubscribedPerformer, error)
}

type StashBoxService interface {
	RefreshStashBoxes(ctx context.Context) error
	SnapshotState() (endpoints []metadata.StashBoxEndpoint, state metadata.LoadState)
}

type LogReader interface {
	Entries(limit int, minLevel string) []logging.Entry
}

type Resolver struct {
	Tracker                  tracker.Tracker
	Torrent                  TorrentClient
	TaskRuntime              TaskRuntimeService
	TaskFlow                 TaskFlowService
	Stash                    StashService
	Performer                PerformerService
	Discovery                DiscoveryService
	PerformerSubscription    SubscriptionService
	TaskEventSource          taskruntime.TaskEventSource
	ServiceStatusEventSource stats.ServiceStatusEventSource
	StashBox                 StashBoxService
	LogReader                LogReader
	SettingsEditor           SettingsEditor
	RuntimeSettings          *SettingsSnapshot
	RuntimeStatus            *SettingsStatusSnapshot
	Stats                    StatsProvider
	ImageCache               ImageCacheService
	AppVersion               string
}

func NewResolver(tr tracker.Tracker, torrent TorrentClient, taskRuntime TaskRuntimeService, stash StashService, version string) *Resolver {
	resolver := &Resolver{
		Tracker:     tr,
		Torrent:     torrent,
		TaskRuntime: taskRuntime,
		Stash:       stash,
		AppVersion:  version,
	}
	if taskRuntime != nil {
		resolver.TaskFlow = taskflow.NewService(taskRuntime)
	}
	return resolver
}
