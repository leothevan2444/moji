package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func downloaderPolicyModel(p downloader.Policy) *model.DownloaderPolicy {
	return &model.DownloaderPolicy{AfterIngest: p.AfterIngest, Enabled: p.Enabled, CapacityEnabled: p.CapacityEnabled, MinFreeGb: p.MinFreeGB, CapacityOrder: model.CapacityCleanupOrder(p.CapacityOrder), DeleteFiles: p.DeleteFiles}
}
func managedTorrentModel(t downloader.Torrent) *model.ManagedTorrent {
	return &model.ManagedTorrent{Hash: t.Hash, Name: t.Name, State: t.State, Present: t.Present, TaskIds: t.TaskIDs, TaskCodes: t.TaskCodes, AllCompleted: t.AllCompleted, CanDeleteFiles: t.CanDeleteFiles, Progress: t.Progress, DownloadSpeed: t.DownloadSpeed, UploadSpeed: t.UploadSpeed, Size: t.Size, CompletedBytes: t.CompletedBytes, Downloaded: t.Downloaded, Uploaded: t.Uploaded, Ratio: t.Ratio, SeedingSeconds: t.SeedingSeconds, Eta: t.ETA, Seeds: t.Seeds, Peers: t.Peers, Availability: t.Availability, SavePath: t.SavePath, Category: t.Category, Tags: t.Tags, LastSeenAt: t.LastSeenAt, RemovedAt: t.RemovedAt, RemovalReason: t.RemovalReason, NeedsAttention: t.NeedsAttention(), CleanupError: t.CleanupError}
}

func transferPreferencesModel(p *qbittorrent.TorrentPreferences) *model.DownloaderTransferPreferences {
	return &model.DownloaderTransferPreferences{QueueingEnabled: p.QueueingEnabled, MaxActiveDownloads: p.MaxActiveDownloads, MaxActiveUploads: p.MaxActiveUploads, MaxActiveTorrents: p.MaxActiveTorrents, DontCountSlowTorrents: p.DontCountSlowTorrents, SlowTorrentDlRateThreshold: p.SlowTorrentDLRateThreshold, SlowTorrentUlRateThreshold: p.SlowTorrentULRateThreshold, SlowTorrentInactiveTimer: p.SlowTorrentInactiveTimer, MaxRatioEnabled: p.MaxRatioEnabled, MaxRatio: p.MaxRatio, MaxRatioAction: p.MaxRatioAction, MaxSeedingTimeEnabled: p.MaxSeedingTimeEnabled, MaxSeedingTime: p.MaxSeedingTime, MaxInactiveSeedingTimeEnabled: p.MaxInactiveSeedingTimeEnabled, MaxInactiveSeedingTime: p.MaxInactiveSeedingTime, ShareLimitMode: p.ShareLimitMode}
}

func downloaderCleanupModel(v downloader.CleanupStatus) *model.DownloaderCleanupStatus {
	phase := v.Phase
	if phase == "" {
		phase = "IDLE"
	}
	return &model.DownloaderCleanupStatus{Enabled: v.Enabled, Phase: model.CleanupPhase(phase), Reason: v.Reason, Hash: v.Hash, Name: v.Name, UpdatedAt: v.UpdatedAt}
}
