package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/stats"
)

func subscriptionStashBoxesToModel(items []StashBoxEndpointSnapshot) []*model.StashBoxEndpoint {
	if len(items) == 0 {
		return []*model.StashBoxEndpoint{}
	}
	out := make([]*model.StashBoxEndpoint, 0, len(items))
	for _, item := range items {
		out = append(out, &model.StashBoxEndpoint{
			Name:             item.Name,
			Endpoint:         item.Endpoint,
			APIKeyConfigured: item.APIKeyConfigured,
		})
	}
	return out
}

func subscriptionLoadErrorPtr(message string) *string {
	if message == "" {
		return nil
	}
	copy := message
	return &copy
}

func settingsSnapshotToModel(snapshot *SettingsSnapshot, appVersion string) *model.Settings {
	if snapshot == nil {
		return &model.Settings{
			Stash: &model.StashSettings{},
			Ingest: &model.IngestSettings{
				SharedStorage: &model.SharedStorageIngestSettings{},
				FileTransfer:  &model.FileTransferIngestSettings{},
				LibraryScan:   &model.LibraryScanIngestSettings{},
			},
			Jackett:      &model.JackettSettings{},
			Qbittorrent:  &model.QBittorrentSettings{},
			Automation:   &model.AutomationSettings{},
			Subscription: &model.SubscriptionSettings{},
		}
	}

	return &model.Settings{
		Stash: &model.StashSettings{
			Configured:       snapshot.Stash.Configured,
			Enabled:          snapshot.Stash.Enabled,
			URL:              snapshot.Stash.URL,
			APIKeyConfigured: snapshot.Stash.APIKeyConfigured,
			APIKey:           snapshot.Stash.APIKey,
		},
		Ingest: &model.IngestSettings{
			Mode: snapshot.Ingest.Mode,
			SharedStorage: &model.SharedStorageIngestSettings{
				QbittorrentPathPrefix: snapshot.Ingest.SharedStorage.QBittorrentPathPrefix,
				StashPathPrefix:       snapshot.Ingest.SharedStorage.StashPathPrefix,
			},
			FileTransfer: &model.FileTransferIngestSettings{
				Action:     snapshot.Ingest.FileTransfer.Action,
				TargetPath: snapshot.Ingest.FileTransfer.TargetPath,
			},
			LibraryScan: &model.LibraryScanIngestSettings{
				LibraryPath: snapshot.Ingest.LibraryScan.LibraryPath,
			},
		},
		Jackett: &model.JackettSettings{
			Configured:         snapshot.Jackett.Configured,
			Enabled:            snapshot.Jackett.Enabled,
			URL:                snapshot.Jackett.URL,
			APIKeyConfigured:   snapshot.Jackett.APIKeyConfigured,
			APIKey:             snapshot.Jackett.APIKey,
			PasswordConfigured: snapshot.Jackett.PasswordConfigured,
			Password:           snapshot.Jackett.Password,
		},
		Qbittorrent: &model.QBittorrentSettings{
			Configured:         snapshot.QBittorrent.Configured,
			Enabled:            snapshot.QBittorrent.Enabled,
			URL:                snapshot.QBittorrent.URL,
			Username:           snapshot.QBittorrent.Username,
			UsernameConfigured: snapshot.QBittorrent.UsernameConfigured,
			PasswordConfigured: snapshot.QBittorrent.PasswordConfigured,
			Password:           snapshot.QBittorrent.Password,
			DefaultSavePath:    snapshot.QBittorrent.DefaultSavePath,
			Category:           snapshot.QBittorrent.Category,
			Tags:               snapshot.QBittorrent.Tags,
		},
		Automation: &model.AutomationSettings{
			TaskProgressSyncIntervalSeconds: snapshot.Automation.TaskProgressSyncIntervalSeconds,
			SubscriptionPollIntervalSeconds: snapshot.Automation.SubscriptionPollIntervalSeconds,
		},
		Subscription: &model.SubscriptionSettings{
			StashBoxEndpoints: append([]string(nil), snapshot.Subscription.StashBoxEndpoints...),
		},
	}
}

func settingsStatusSnapshotToModel(snapshot *SettingsStatusSnapshot) *model.SettingsStatus {
	if snapshot == nil {
		return &model.SettingsStatus{
			Stash:            &model.ServiceStatus{},
			Jackett:          &model.ServiceStatus{},
			Qbittorrent:      &model.ServiceStatus{},
			Automation:       &model.AutomationStatus{},
			Subscription:     &model.SubscriptionStatus{},
			Ingest:           &model.IngestStatus{},
			StashStats:       emptyStashStatsModel(),
			JackettStats:     emptyJackettStatsModel(),
			QbittorrentStats: emptyQBittorrentStatsModel(),
		}
	}

	return &model.SettingsStatus{
		Stash: &model.ServiceStatus{
			Configured: snapshot.Stash.Configured,
			Enabled:    snapshot.Stash.Enabled,
		},
		Jackett: &model.ServiceStatus{
			Configured: snapshot.Jackett.Configured,
			Enabled:    snapshot.Jackett.Enabled,
		},
		Qbittorrent: &model.ServiceStatus{
			Configured: snapshot.QBittorrent.Configured,
			Enabled:    snapshot.QBittorrent.Enabled,
		},
		Automation: &model.AutomationStatus{
			TaskProgressSyncIntervalSeconds: snapshot.Automation.TaskProgressSyncIntervalSeconds,
			TaskProgressSyncEnabled:         snapshot.Automation.TaskProgressSyncEnabled,
			SubscriptionPollIntervalSeconds: snapshot.Automation.SubscriptionPollIntervalSeconds,
			SubscriptionPollEnabled:         snapshot.Automation.SubscriptionPollEnabled,
		},
		Subscription: &model.SubscriptionStatus{
			StashBoxes:          subscriptionStashBoxesToModel(snapshot.Subscription.StashBoxes),
			StashBoxesLoaded:    snapshot.Subscription.StashBoxesLoaded,
			StashBoxesLoadError: subscriptionLoadErrorPtr(snapshot.Subscription.StashBoxesLoadError),
		},
		Ingest: &model.IngestStatus{
			Configured: snapshot.Ingest.Configured,
		},
		StashStats:       emptyStashStatsModel(),
		JackettStats:     emptyJackettStatsModel(),
		QbittorrentStats: emptyQBittorrentStatsModel(),
	}
}

// SettingsStatusWithStats is settingsStatusSnapshotToModel combined with the
// optional runtime-stats snapshot from the stats collector. When stats is nil
// (collector not wired, e.g. in tests), the stats fields are returned as
// zero-value placeholders so the GraphQL response remains valid.
func SettingsStatusWithStats(snapshot *SettingsStatusSnapshot, stats *stats.Snapshot) *model.SettingsStatus {
	out := settingsStatusSnapshotToModel(snapshot)
	if stats == nil {
		return out
	}
	out.StashStats = stashStatsToModel(stats.Stash)
	out.JackettStats = jackettStatsToModel(stats.Jackett)
	out.QbittorrentStats = qBittorrentStatsToModel(stats.QBitt)
	return out
}

func emptyStashStatsModel() *model.StashStats {
	return &model.StashStats{
		PendingMojiScanCount: 0,
		OkAt:                 time.Time{}.UTC().Format(time.RFC3339),
	}
}

func emptyJackettStatsModel() *model.JackettStats {
	return &model.JackettStats{
		IndexerCount:           0,
		ConfiguredIndexerCount: 0,
		LastIndexerLatencyMs:   0,
		OkAt:                   time.Time{}.UTC().Format(time.RFC3339),
	}
}

func emptyQBittorrentStatsModel() *model.QBittorrentStats {
	return &model.QBittorrentStats{
		DownloadSpeed:      0,
		UploadSpeed:        0,
		ActiveTorrentCount: 0,
		ConnectionStatus:   "unknown",
		OkAt:               time.Time{}.UTC().Format(time.RFC3339),
	}
}

func stashStatsToModel(s stats.StashStats) *model.StashStats {
	out := &model.StashStats{
		PendingMojiScanCount: s.PendingMojiScanCount,
		OkAt:                 s.OKAt.UTC().Format(time.RFC3339),
	}
	if s.Version != "" {
		v := s.Version
		out.Version = &v
	}
	if s.SceneCount != nil {
		n := *s.SceneCount
		out.SceneCount = &n
	}
	if s.LastError != "" {
		e := s.LastError
		out.LastError = &e
	}
	return out
}

func jackettStatsToModel(s stats.JackettStats) *model.JackettStats {
	out := &model.JackettStats{
		IndexerCount:           s.IndexerCount,
		ConfiguredIndexerCount: s.ConfiguredIndexerCount,
		LastIndexerLatencyMs:   s.LastIndexerLatencyMs,
		OkAt:                   s.OKAt.UTC().Format(time.RFC3339),
	}
	if s.LastIndexerError != "" {
		e := s.LastIndexerError
		out.LastIndexerError = &e
	}
	if s.LastIndexerSearchAt != nil {
		t := s.LastIndexerSearchAt.UTC().Format(time.RFC3339)
		out.LastIndexerSearchAt = &t
	}
	if s.LastError != "" {
		e := s.LastError
		out.LastError = &e
	}
	return out
}

func qBittorrentStatsToModel(s stats.QBittorrentStats) *model.QBittorrentStats {
	conn := s.ConnectionStatus
	if conn == "" {
		conn = "unknown"
	}
	out := &model.QBittorrentStats{
		DownloadSpeed:        int(s.DownloadSpeed),
		UploadSpeed:          int(s.UploadSpeed),
		ActiveTorrentCount:   s.ActiveTorrentCount,
		ConnectionStatus:     conn,
		AltSpeedLimitEnabled: s.AltSpeedLimitEnabled,
		OkAt:                 s.OKAt.UTC().Format(time.RFC3339),
	}
	if s.LastError != "" {
		e := s.LastError
		out.LastError = &e
	}
	return out
}
