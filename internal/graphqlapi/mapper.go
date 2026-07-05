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
	source := task.Source
	if source == "" {
		source = downloader.TaskSourceManual
	}

	return &model.Task{
		ID:                  task.ID,
		Source:              model.TaskSource(source),
		Query:               task.Query,
		Code:                task.Code,
		Status:              string(task.Status),
		Candidate:           candidateToModel(task.Candidate),
		TorrentURL:          task.TorrentURL,
		SavePath:            task.SavePath,
		Category:            task.Category,
		Tags:                task.Tags,
		TorrentHash:         task.TorrentHash,
		TorrentName:         task.TorrentName,
		Progress:            task.Progress,
		QbittorrentState:    task.QBittorrentState,
		ContentPath:         task.ContentPath,
		CompletedAt:         formatOptionalTime(task.CompletedAt),
		StashMode:           task.StashMode,
		StashSourcePath:     task.StashSourcePath,
		StashTransferAction: task.StashTransferAction,
		StashTransferPath:   task.StashTransferPath,
		StashTransferStatus: task.StashTransferStatus,
		StashTransferError:  task.StashTransferError,
		StashJobID:          task.StashJobID,
		StashScanPath:       task.StashScanPath,
		StashScanStatus:     task.StashScanStatus,
		StashScanError:      task.StashScanError,
		StashScanHint:       task.StashScanHint,
		StashScanStartedAt:  formatOptionalTime(task.StashScanStartedAt),
		Error:               task.Error,
		CreatedAt:           formatTime(task.CreatedAt),
		UpdatedAt:           formatTime(task.UpdatedAt),
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
