package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

func downloaderPolicyModel(p downloader.Policy) *model.DownloaderPolicy {
	return &model.DownloaderPolicy{AfterIngest: p.AfterIngest, Enabled: p.Enabled, Ratio: p.Ratio, SeedHours: p.SeedHours, MatchAll: p.MatchAll, DeleteFiles: p.DeleteFiles}
}
func managedTorrentModel(t downloader.Torrent) *model.ManagedTorrent {
	return &model.ManagedTorrent{Hash: t.Hash, Name: t.Name, State: t.State, Present: t.Present, TaskIds: t.TaskIDs, TaskCodes: t.TaskCodes, AllCompleted: t.AllCompleted, CanDeleteFiles: t.CanDeleteFiles, Progress: t.Progress, DownloadSpeed: t.DownloadSpeed, UploadSpeed: t.UploadSpeed, Size: t.Size, CompletedBytes: t.CompletedBytes, Downloaded: t.Downloaded, Uploaded: t.Uploaded, Ratio: t.Ratio, SeedingSeconds: t.SeedingSeconds, Eta: t.ETA, Seeds: t.Seeds, Peers: t.Peers, Availability: t.Availability, SavePath: t.SavePath, Category: t.Category, Tags: t.Tags, LastSeenAt: t.LastSeenAt, RemovedAt: t.RemovedAt, RemovalReason: t.RemovalReason, NeedsAttention: t.NeedsAttention(), CleanupError: t.CleanupError}
}
