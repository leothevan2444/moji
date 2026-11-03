package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func taskEventToModel(event *taskruntime.TaskEvent, includeTask ...bool) *model.TaskEvent {
	if event == nil {
		return nil
	}
	var legacyTask *model.Task
	if len(includeTask) == 0 || includeTask[0] {
		legacyTask = taskToModel(event.Task)
	}
	result := &model.TaskEvent{
		StreamID: event.StreamID, ListChanged: event.ListChanged, Summary: taskSummaryToModel(event.Task),
		Sequence: event.Sequence,
		Type:     model.TaskEventType(event.Type),
		TaskID:   event.TaskID,
		Task:     legacyTask,
		DashboardStats: &model.DashboardStats{
			Attention: event.DashboardStats.Attention, Waiting: event.DashboardStats.Waiting,
			Total:        event.DashboardStats.Total,
			Active:       event.DashboardStats.Active,
			Completed:    event.DashboardStats.Completed,
			Downloading:  event.DashboardStats.Downloading,
			PendingScans: event.DashboardStats.PendingScans,
			Failed:       event.DashboardStats.Failed,
		},
	}
	if event.Type == taskruntime.TaskEventReady {
		result.DashboardStats = nil
	}
	return result
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

func taskToModel(task *taskruntime.Task) *model.Task {
	task = taskruntime.PublicTaskSnapshot(task)
	if task == nil {
		return nil
	}
	source := task.Source
	if source == "" {
		source = taskruntime.TaskSourceManual
	}

	files := []*model.TaskFile{}
	for _, file := range task.FilePlan().Files {
		files = append(files, &model.TaskFile{Index: file.Index, Path: file.Path, Size: file.Size, Kind: string(file.Kind), Part: file.Part, Selected: file.Selected, Reason: file.Reason})
	}
	return &model.Task{
		Version: taskVersion(task), Revision: task.Revision,
		Files:                 files,
		ID:                    task.ID,
		Source:                model.TaskSource(source),
		Code:                  task.Code,
		Stage:                 model.TaskStage(task.Stage),
		StageStatus:           model.TaskStageStatus(task.StageStatus),
		StageLabel:            task.StageLabel,
		StageStatusLabel:      task.StageStatusLabel,
		StageErrorCode:        nilIfEmpty(task.StageErrorCode),
		StageErrorMessage:     nilIfEmpty(task.StageErrorMessage),
		Candidate:             candidateToModel(task.Candidate),
		TorrentURL:            task.TorrentURL,
		SavePath:              nilIfEmpty(task.SavePath),
		Category:              nilIfEmpty(task.Category),
		Tags:                  nilIfEmpty(task.Tags),
		TorrentHash:           nilIfEmpty(task.TorrentHash),
		TorrentName:           nilIfEmpty(task.TorrentName),
		Progress:              task.Progress,
		QbittorrentState:      nilIfEmpty(task.QBittorrentState),
		ContentPath:           nilIfEmpty(task.ContentPath),
		LastProgressAt:        formatOptionalTime(task.LastProgressAt),
		SourcingCompletedAt:   formatOptionalTime(task.SourcingCompletedAt),
		SourcingLastCheckedAt: formatOptionalTime(task.SourcingLastCheckedAt),
		SourcingNextCheckAt:   formatOptionalTime(task.SourcingNextCheckAt),
		DownloadCompletedAt:   formatOptionalTime(task.DownloadCompletedAt),
		IngestCompletedAt:     formatOptionalTime(task.IngestCompletedAt),
		MojiSourcePath:        nilIfEmpty(task.MojiSourcePath),
		TransferAction:        nilIfEmpty(task.TransferAction),
		MojiTransferPath:      nilIfEmpty(task.MojiTransferPath),
		TransferError:         nilIfEmpty(task.TransferError),
		StashIngestPhase:      nilIfEmpty(task.StashIngestPhase),
		StashGenerateJobID:    nilIfEmpty(task.StashGenerateJobID),
		StashIdentifyJobID:    nilIfEmpty(task.StashIdentifyJobID),
		StashSceneIds:         task.StashSceneIDs(),
		StashScanJobID:        nilIfEmpty(task.StashScanJobID),
		StashScanPath:         nilIfEmpty(task.StashScanPath),
		StashScanError:        nilIfEmpty(task.StashScanError),
		StashScanHint:         nilIfEmpty(task.StashScanHint),
		StashScanStartedAt:    formatOptionalTime(task.StashScanStartedAt),
		CreatedAt:             formatTime(task.CreatedAt),
		UpdatedAt:             formatTime(task.UpdatedAt),
	}
}

func taskBatchPayloadToModel(payload taskruntime.TaskBatchPayload) *model.TaskBatchPayload {
	results := make([]*model.TaskBatchResult, 0, len(payload.Results))
	for _, result := range payload.Results {
		results = append(results, &model.TaskBatchResult{
			TaskID:     result.TaskID,
			Status:     model.TaskBatchStatus(result.Status),
			ReasonCode: result.ReasonCode,
			Task:       taskToModel(result.Task),
		})
	}
	return &model.TaskBatchPayload{
		BatchID: payload.BatchID,
		Summary: &model.TaskBatchSummary{
			RequestedCount: payload.Summary.RequestedCount,
			SucceededCount: payload.Summary.SucceededCount,
			SkippedCount:   payload.Summary.SkippedCount,
			FailedCount:    payload.Summary.FailedCount,
		},
		Results: results,
	}
}

func candidateToModel(candidate taskruntime.Candidate) *model.DownloadCandidate {
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
