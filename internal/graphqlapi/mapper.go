package graphqlapi

import (
	"sort"
	"time"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func jackettSearchResultToModel(result jackett.SearchResult) *model.JackettSearchResult {
	return &model.JackettSearchResult{
		Title:        result.Title,
		Size:         result.Size,
		Seeders:      result.Seeders,
		Peers:        result.Peers,
		Tracker:      result.Tracker,
		TrackerID:    result.TrackerID,
		CategoryDesc: result.CategoryDesc,
		PublishDate:  result.PublishDate,
		Details:      result.Details,
		Link:         result.Link,
		MagnetURI:    result.MagnetURI,
		InfoHash:     result.InfoHash,
	}
}

func jackettSearchResultsToModel(results []jackett.SearchResult) []*model.JackettSearchResult {
	out := make([]*model.JackettSearchResult, 0, len(results))
	for _, result := range results {
		out = append(out, jackettSearchResultToModel(result))
	}
	return out
}

func previewJackettSelectionCandidatesFromModel(input []*model.PreviewJackettSelectionCandidateInput) []jackett.SearchResult {
	out := make([]jackett.SearchResult, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		out = append(out, jackett.SearchResult{
			Title:        item.Title,
			Size:         item.Size,
			Seeders:      item.Seeders,
			Peers:        item.Peers,
			Tracker:      item.Tracker,
			TrackerID:    item.TrackerID,
			CategoryDesc: item.CategoryDesc,
			PublishDate:  item.PublishDate,
			Details:      item.Details,
			Link:         item.Link,
			MagnetURI:    item.MagnetURI,
			InfoHash:     item.InfoHash,
		})
	}
	return out
}

func candidateSelectionPreviewToModel(preview *downloader.CandidateSelectionPreview) *model.PreviewJackettSelectionResult {
	if preview == nil {
		return &model.PreviewJackettSelectionResult{
			Results:     []*model.JackettSearchResult{},
			PreviewMeta: &model.PreviewJackettSelectionMeta{},
		}
	}
	return &model.PreviewJackettSelectionResult{
		Results: jackettSearchResultsToModel(preview.Results),
		PreviewMeta: &model.PreviewJackettSelectionMeta{
			AppliedFastRules: preview.Meta.AppliedFastRules,
			AppliedFileRules: preview.Meta.AppliedFileRules,
			InspectedCount:   preview.Meta.InspectedCount,
			InspectableCount: preview.Meta.InspectableCount,
		},
	}
}

func jackettIndexerToModel(indexer jackett.Indexer) *model.JackettIndexer {
	return &model.JackettIndexer{
		ID:      indexer.ID,
		Name:    indexer.Name,
		Enabled: indexer.Configured,
	}
}

// sortJackettResults sorts in place. RELEVANCE preserves Jackett's native order.
func sortJackettResults(results []*model.JackettSearchResult, sortBy *model.JackettSortBy) {
	if sortBy == nil {
		return
	}
	switch *sortBy {
	case model.JackettSortByRelevance:
		return
	case model.JackettSortBySeedersDesc:
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Seeders > results[j].Seeders
		})
	case model.JackettSortBySizeDesc:
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Size > results[j].Size
		})
	case model.JackettSortByDateDesc:
		sort.SliceStable(results, func(i, j int) bool {
			a, b := results[i].PublishDate, results[j].PublishDate
			if a == b {
				return false
			}
			if a == "" {
				return false
			}
			if b == "" {
				return true
			}
			return a > b
		})
	}
}

func qbTorrentToModel(torrent qbittorrent.Torrent) *model.QBTorrent {
	return &model.QBTorrent{
		Hash:     torrent.Hash,
		Name:     torrent.Name,
		Progress: torrent.Progress,
		State:    string(torrent.State),
		Dlspeed:  torrent.DLSpeed,
		Upspeed:  torrent.UPSpeed,
		Size:     torrent.Size,
		Eta:      torrent.ETA,
		Category: torrent.Category,
		Tags:     torrent.Tags,
		AddedOn:  torrent.AddedOn,
	}
}

func taskToModel(task *downloader.Task) *model.Task {
	if task == nil {
		return nil
	}
	task = downloaderTaskForView(task)
	source := task.Source
	if source == "" {
		source = downloader.TaskSourceManual
	}

	return &model.Task{
		ID:                  task.ID,
		Source:              model.TaskSource(source),
		Query:               task.Query,
		Code:                task.Code,
		Stage:               model.TaskStage(task.Stage),
		StageStatus:         model.TaskStageStatus(task.StageStatus),
		StageLabel:          task.StageLabel,
		StageStatusLabel:    task.StageStatusLabel,
		StageErrorCode:      nilIfEmpty(task.StageErrorCode),
		StageErrorMessage:   nilIfEmpty(task.StageErrorMessage),
		Candidate:           candidateToModel(task.Candidate),
		TorrentURL:          task.TorrentURL,
		SavePath:            nilIfEmpty(task.SavePath),
		Category:            nilIfEmpty(task.Category),
		Tags:                nilIfEmpty(task.Tags),
		TorrentHash:         nilIfEmpty(task.TorrentHash),
		TorrentName:         nilIfEmpty(task.TorrentName),
		Progress:            task.Progress,
		QbittorrentState:    nilIfEmpty(task.QBittorrentState),
		ContentPath:         nilIfEmpty(task.ContentPath),
		DownloadCompletedAt: formatOptionalTime(task.DownloadCompletedAt),
		DeliveryMode:        nilIfEmpty(task.DeliveryMode),
		MojiSourcePath:      nilIfEmpty(task.MojiSourcePath),
		TransferAction:      nilIfEmpty(task.TransferAction),
		MojiTransferPath:    nilIfEmpty(task.MojiTransferPath),
		TransferError:       nilIfEmpty(task.TransferError),
		StashScanJobID:      nilIfEmpty(task.StashScanJobID),
		StashScanPath:       nilIfEmpty(task.StashScanPath),
		StashScanError:      nilIfEmpty(task.StashScanError),
		StashScanHint:       nilIfEmpty(task.StashScanHint),
		StashScanStartedAt:  formatOptionalTime(task.StashScanStartedAt),
		CreatedAt:           formatTime(task.CreatedAt),
		UpdatedAt:           formatTime(task.UpdatedAt),
	}
}

func downloaderTaskForView(task *downloader.Task) *downloader.Task {
	if task == nil {
		return nil
	}
	cp := *task
	refreshTaskLike(&cp)
	return &cp
}

func refreshTaskLike(task *downloader.Task) {
	if task == nil {
		return
	}
	if task.Stage == "" && task.Status != "" {
		switch task.Status {
		case downloader.TaskStatusPending:
			task.Stage = downloader.TaskStageSourcing
			task.StageStatus = downloader.TaskStageStatusRunning
		case downloader.TaskStatusAdded, downloader.TaskStatusDownloading:
			task.Stage = downloader.TaskStageDownloading
			task.StageStatus = downloader.TaskStageStatusRunning
		case downloader.TaskStatusCompleted:
			task.Stage = downloader.TaskStagePendingIngest
			task.StageStatus = downloader.TaskStageStatusPending
		case downloader.TaskStatusFailed:
			task.Stage = downloader.TaskStageDownloading
			task.StageStatus = downloader.TaskStageStatusBlocked
		}
	}
	if task.StageLabel == "" {
		switch task.Stage {
		case downloader.TaskStageSourcing:
			task.StageLabel = "待选种"
		case downloader.TaskStageDownloading:
			task.StageLabel = "下载中"
		case downloader.TaskStagePendingIngest:
			task.StageLabel = "待入库"
		case downloader.TaskStageTransferring:
			task.StageLabel = "搬运中"
		case downloader.TaskStageScanning:
			task.StageLabel = "扫描中"
		case downloader.TaskStageCompleted:
			task.StageLabel = "已完成"
		}
	}
	if task.StageStatusLabel == "" {
		switch task.StageStatus {
		case downloader.TaskStageStatusPending:
			task.StageStatusLabel = "待处理"
		case downloader.TaskStageStatusRunning:
			task.StageStatusLabel = "进行中"
		case downloader.TaskStageStatusBlocked:
			task.StageStatusLabel = "受阻"
		case downloader.TaskStageStatusDone:
			task.StageStatusLabel = "已完成"
		}
	}
	if task.DownloadCompletedAt == nil && task.CompletedAt != nil {
		task.DownloadCompletedAt = task.CompletedAt
	}
	if task.DeliveryMode == "" {
		task.DeliveryMode = task.StashMode
	}
	if task.MojiSourcePath == "" {
		task.MojiSourcePath = task.StashSourcePath
	}
	if task.TransferAction == "" {
		task.TransferAction = task.StashTransferAction
	}
	if task.MojiTransferPath == "" {
		task.MojiTransferPath = task.StashTransferPath
	}
	if task.TransferError == "" {
		task.TransferError = task.StashTransferError
	}
	if task.StashScanJobID == "" {
		task.StashScanJobID = task.StashJobID
	}
}

func candidateToModel(candidate downloader.Candidate) *model.DownloadCandidate {
	return &model.DownloadCandidate{
		Title:     candidate.Title,
		Tracker:   candidate.Tracker,
		InfoHash:  candidate.InfoHash,
		Link:      candidate.Link,
		MagnetURI: candidate.MagnetURI,
		Size:      candidate.Size,
		Seeders:   candidate.Seeders,
		Peers:     candidate.Peers,
	}
}

func stashJobToModel(job *stashsync.Job) *model.StashJob {
	if job == nil {
		return nil
	}

	return &model.StashJob{
		ID:          job.ID,
		Status:      job.Status,
		Description: job.Description,
		Progress:    job.Progress,
		StartTime:   formatOptionalTime(job.StartTime),
		EndTime:     formatOptionalTime(job.EndTime),
		AddTime:     formatTime(job.AddTime),
		Error:       job.Error,
		SubTasks:    job.SubTasks,
	}
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}

	formatted := formatTime(*t)
	return &formatted
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
