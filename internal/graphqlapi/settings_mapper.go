package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/stats"
)

func imageCacheStatusToModel(status imagecache.Status) *model.ImageCacheStatus {
	var cleanup *string
	if status.LastCleanupAt != nil {
		v := status.LastCleanupAt.UTC().Format(time.RFC3339)
		cleanup = &v
	}
	var lastErr *string
	if status.LastError != "" {
		v := status.LastError
		lastErr = &v
	}
	return &model.ImageCacheStatus{UsedBytes: status.UsedBytes, EntryCount: status.EntryCount, CacheDirectory: status.CacheDirectory, LastCleanupAt: cleanup, LastError: lastErr}
}

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

func stashLibrariesToModel(items []StashLibrarySnapshot) []*model.StashLibrary {
	if len(items) == 0 {
		return []*model.StashLibrary{}
	}
	out := make([]*model.StashLibrary, 0, len(items))
	for _, item := range items {
		out = append(out, &model.StashLibrary{Path: item.Path})
	}
	return out
}

func settingsSnapshotToModel(snapshot *SettingsSnapshot, appVersion string) *model.Settings {
	if snapshot == nil {
		return &model.Settings{
			Stash: &model.StashSettings{},
			Ingest: &model.IngestSettings{
				Downloads: &model.DownloadsIngestSettings{},
				Library:   &model.LibraryIngestSettings{},
				Transfer:  &model.TransferIngestSettings{},
			},
			Jackett:     &model.JackettSettings{},
			Qbittorrent: &model.QBittorrentSettings{},
			Automation:  &model.AutomationSettings{},
			System:      &model.SystemSettings{ImageCache: &model.ImageCacheSettings{}},
		}
	}

	return &model.Settings{
		Stash: &model.StashSettings{
			Configured:       snapshot.Stash.Configured,
			URL:              snapshot.Stash.URL,
			APIKeyConfigured: snapshot.Stash.APIKeyConfigured,
			APIKey:           snapshot.Stash.APIKey,
		},
		Ingest: &model.IngestSettings{
			DeliveryMode: snapshot.Ingest.DeliveryMode,
			Downloads: &model.DownloadsIngestSettings{
				QbRoot:   snapshot.Ingest.Downloads.QBRoot,
				MojiRoot: snapshot.Ingest.Downloads.MojiRoot,
			},
			Library: &model.LibraryIngestSettings{
				MojiRoot:  snapshot.Ingest.Library.MojiRoot,
				StashRoot: snapshot.Ingest.Library.StashRoot,
			},
			Transfer: &model.TransferIngestSettings{
				Action: snapshot.Ingest.Transfer.Action,
			},
		},
		Jackett: &model.JackettSettings{
			Configured:         snapshot.Jackett.Configured,
			URL:                snapshot.Jackett.URL,
			APIKeyConfigured:   snapshot.Jackett.APIKeyConfigured,
			APIKey:             snapshot.Jackett.APIKey,
			PasswordConfigured: snapshot.Jackett.PasswordConfigured,
			Password:           snapshot.Jackett.Password,
		},
		Qbittorrent: &model.QBittorrentSettings{
			Configured:         snapshot.QBittorrent.Configured,
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
			SubscriptionPollIntervalHours:   snapshot.Automation.SubscriptionPollIntervalHours,
			StashBoxEndpoints:               append([]string(nil), snapshot.Automation.StashBoxEndpoints...),
			SubscriptionReleasePolicy:       subscriptionReleasePolicyToModel(snapshot.Automation.SubscriptionReleasePolicy),
			TorrentSelection:                torrentSelectionSettingsToModel(snapshot.Automation.TorrentSelection),
		},
		System: &model.SystemSettings{
			TaskDeletePolicy: model.TaskDeletePolicy(snapshot.System.TaskDeletePolicy),
			ImageCache:       &model.ImageCacheSettings{Enabled: snapshot.System.ImageCache.Enabled, MaxSizeMb: snapshot.System.ImageCache.MaxSizeMB, RetentionDays: snapshot.System.ImageCache.RetentionDays},
		},
	}
}

func subscriptionReleasePolicyToModel(snapshot SubscriptionReleasePolicySnapshot) *model.SubscriptionReleasePolicy {
	return &model.SubscriptionReleasePolicy{
		SoloBehavior:           model.SubscriptionReleaseBehavior(snapshot.SoloBehavior),
		GroupBehavior:          model.SubscriptionReleaseBehavior(snapshot.GroupBehavior),
		CompilationBehavior:    model.SubscriptionReleaseBehavior(snapshot.CompilationBehavior),
		MaxGroupPerformerCount: snapshot.MaxGroupPerformerCount,
		ReleaseDateRange:       model.SubscriptionReleaseDateRange(snapshot.ReleaseDateRange),
	}
}

func torrentSelectionSettingsToModel(snapshot TorrentSelectionSettingsSnapshot) *model.TorrentSelectionSettings {
	out := &model.TorrentSelectionSettings{
		Enabled:                  snapshot.Enabled,
		InspectionCandidateLimit: snapshot.InspectionCandidateLimit,
		FastRules:                make([]*model.TorrentSelectionRule, 0, len(snapshot.FastRules)),
		TorrentRules:             make([]*model.TorrentSelectionRule, 0, len(snapshot.TorrentRules)),
	}
	for _, rule := range snapshot.FastRules {
		out.FastRules = append(out.FastRules, torrentSelectionRuleToModel(rule))
	}
	for _, rule := range snapshot.TorrentRules {
		out.TorrentRules = append(out.TorrentRules, torrentSelectionRuleToModel(rule))
	}
	return out
}

func torrentSelectionRuleToModel(rule TorrentSelectionRuleSnapshot) *model.TorrentSelectionRule {
	item := &model.TorrentSelectionRule{
		Type:    model.TorrentSelectionRuleType(rule.Type),
		Enabled: rule.Enabled,
		IndexerPreference: &model.IndexerPreferenceRule{
			TrackerIds: append([]string(nil), rule.IndexerPreference.TrackerIDs...),
		},
		TitleMatch: &model.TitleMatchRule{
			Clauses: make([]*model.TitleMatchClause, 0, len(rule.TitleMatch.Clauses)),
		},
		PublishDate: &model.DirectionRule{
			Direction: model.TorrentSelectionDirection(rule.PublishDate.Direction),
		},
		Seeders: &model.DirectionRule{
			Direction: model.TorrentSelectionDirection(rule.Seeders.Direction),
		},
		Size: &model.DirectionRule{
			Direction: model.TorrentSelectionDirection(rule.Size.Direction),
		},
		TorrentFileNameMatch: &model.TorrentFileNameMatchRule{
			Clauses: make([]*model.TorrentFileNameMatchClause, 0, len(rule.TorrentFileNameMatch.Clauses)),
		},
	}
	for _, clause := range rule.TitleMatch.Clauses {
		item.TitleMatch.Clauses = append(item.TitleMatch.Clauses, &model.TitleMatchClause{
			Pattern:     clause.Pattern,
			PatternMode: model.TitleMatchPatternMode(clause.PatternMode),
			Effect:      model.TitleMatchEffect(clause.Effect),
		})
	}
	for _, clause := range rule.TorrentFileNameMatch.Clauses {
		item.TorrentFileNameMatch.Clauses = append(item.TorrentFileNameMatch.Clauses, &model.TorrentFileNameMatchClause{
			Pattern:     clause.Pattern,
			PatternMode: model.TitleMatchPatternMode(clause.PatternMode),
			Effect:      model.TorrentFileMatchEffect(clause.Effect),
		})
	}
	return item
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
			StashLibraries:   []*model.StashLibrary{},
			StashStats:       emptyStashStatsModel(),
			JackettStats:     emptyJackettStatsModel(),
			QbittorrentStats: emptyQBittorrentStatsModel(),
		}
	}

	return &model.SettingsStatus{
		Stash: &model.ServiceStatus{
			Configured: snapshot.Stash.Configured,
			Ready:      snapshot.Stash.Ready,
		},
		Jackett: &model.ServiceStatus{
			Configured: snapshot.Jackett.Configured,
			Ready:      snapshot.Jackett.Ready,
		},
		Qbittorrent: &model.ServiceStatus{
			Configured: snapshot.QBittorrent.Configured,
			Ready:      snapshot.QBittorrent.Ready,
		},
		Automation: &model.AutomationStatus{
			TaskProgressSyncIntervalSeconds: snapshot.Automation.TaskProgressSyncIntervalSeconds,
			TaskProgressSyncEnabled:         snapshot.Automation.TaskProgressSyncEnabled,
			SubscriptionPollIntervalHours:   snapshot.Automation.SubscriptionPollIntervalHours,
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
		StashLibraries:          stashLibrariesToModel(snapshot.StashLibraries),
		StashLibrariesLoadError: subscriptionLoadErrorPtr(snapshot.StashLibrariesLoadError),
		StashStats:              emptyStashStatsModel(),
		JackettStats:            emptyJackettStatsModel(),
		QbittorrentStats:        emptyQBittorrentStatsModel(),
	}
}

// SettingsStatusWithStats is settingsStatusSnapshotToModel combined with the
// optional runtime-stats snapshot from the stats collector. When stats is nil
// (collector not wired, e.g. in tests), the stats fields are returned as
// zero-value placeholders so the GraphQL response remains valid.
//
// The per-service Ready field is recomputed in the stats branch by folding
// the config-only snapshot signal together with the most recent probe
// result. When stats is nil, Ready reflects the config-only signal.
func SettingsStatusWithStats(snapshot *SettingsStatusSnapshot, stats *stats.Snapshot) *model.SettingsStatus {
	out := settingsStatusSnapshotToModel(snapshot)
	if stats == nil {
		return out
	}
	now := time.Now()
	stashStats := stashStatsToModel(stats.Stash)
	jackettStats := jackettStatsToModel(stats.Jackett)
	qbittStats := qBittorrentStatsToModel(stats.QBitt)
	out.StashStats = stashStats
	out.JackettStats = jackettStats
	out.QbittorrentStats = qbittStats
	out.Stash.Ready = EvaluateServiceReadiness(
		snapshot.Stash.Ready, stats.Stash.OKAt, stats.Stash.LastError, now)
	out.Jackett.Ready = EvaluateServiceReadiness(
		snapshot.Jackett.Ready, stats.Jackett.OKAt, stats.Jackett.LastError, now)
	out.Qbittorrent.Ready = EvaluateServiceReadiness(
		snapshot.QBittorrent.Ready, stats.QBitt.OKAt, stats.QBitt.LastError, now)
	return out
}

func emptyStashStatsModel() *model.StashStats {
	return &model.StashStats{
		PendingMojiScanCount: 0,
		OkAt:                 nil,
	}
}

func emptyJackettStatsModel() *model.JackettStats {
	return &model.JackettStats{
		IndexerCount:           0,
		ConfiguredIndexerCount: 0,
		LastIndexerLatencyMs:   0,
		OkAt:                   nil,
	}
}

func emptyQBittorrentStatsModel() *model.QBittorrentStats {
	return &model.QBittorrentStats{
		DownloadSpeed:      0,
		UploadSpeed:        0,
		ActiveTorrentCount: 0,
		ConnectionStatus:   "unknown",
		OkAt:               nil,
	}
}

func stashStatsToModel(s stats.StashStats) *model.StashStats {
	out := &model.StashStats{
		PendingMojiScanCount: s.PendingMojiScanCount,
	}
	if okAt := formatOKAt(s.OKAt); okAt != nil {
		out.OkAt = okAt
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
	}
	if okAt := formatOKAt(s.OKAt); okAt != nil {
		out.OkAt = okAt
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
	}
	if okAt := formatOKAt(s.OKAt); okAt != nil {
		out.OkAt = okAt
	}
	if s.LastError != "" {
		e := s.LastError
		out.LastError = &e
	}
	return out
}

// formatOKAt returns an RFC3339 string for non-zero times, nil for the
// zero value. Used by every *Stats mapper so the GraphQL response can
// distinguish "never probed" from "probed N minutes ago".
func formatOKAt(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
