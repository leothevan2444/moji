package downloader

import "strings"

type TaskStage string

const (
	TaskStageSourcing      TaskStage = "SOURCING"
	TaskStageDownloading   TaskStage = "DOWNLOADING"
	TaskStagePendingIngest TaskStage = "PENDING_INGEST"
	TaskStageTransferring  TaskStage = "TRANSFERRING"
	TaskStageScanning      TaskStage = "SCANNING"
	TaskStageCompleted     TaskStage = "COMPLETED"
)

type TaskStageStatus string

const (
	TaskStageStatusPending TaskStageStatus = "PENDING"
	TaskStageStatusRunning TaskStageStatus = "RUNNING"
	TaskStageStatusBlocked TaskStageStatus = "BLOCKED"
	TaskStageStatusDone    TaskStageStatus = "DONE"
)

const (
	TaskStageErrorSearch              = "SEARCH_ERROR"
	TaskStageErrorNoCandidate         = "NO_CANDIDATE"
	TaskStageErrorNoDownloadCandidate = "NO_DOWNLOADABLE_CANDIDATE"
	TaskStageErrorTorrentSubmit       = "TORRENT_SUBMIT_FAILED"
	TaskStageErrorTransferPlan        = "TRANSFER_PLAN_FAILED"
	TaskStageErrorTransfer            = "TRANSFER_FAILED"
	TaskStageErrorScanTrigger         = "SCAN_TRIGGER_FAILED"
	TaskStageErrorDuplicateTorrent    = "DUPLICATE_TORRENT"
	TaskStageErrorDuplicateCode       = "DUPLICATE_CODE"
	TaskStageErrorDuplicateLibrary    = "DUPLICATE_LIBRARY_CODE"
	TaskStageErrorCodeRequired        = "TASK_CODE_REQUIRED"
)

func normalizeTaskStage(value TaskStage) TaskStage {
	switch value {
	case TaskStageSourcing,
		TaskStageDownloading,
		TaskStagePendingIngest,
		TaskStageTransferring,
		TaskStageScanning,
		TaskStageCompleted:
		return value
	default:
		return TaskStageSourcing
	}
}

type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusAdded       TaskStatus = "added"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
)

func legacyStatusToStage(value TaskStatus) (TaskStage, TaskStageStatus) {
	switch value {
	case TaskStatusPending:
		return TaskStageSourcing, TaskStageStatusRunning
	case TaskStatusAdded, TaskStatusDownloading:
		return TaskStageDownloading, TaskStageStatusRunning
	case TaskStatusCompleted:
		return TaskStagePendingIngest, TaskStageStatusPending
	case TaskStatusFailed:
		return TaskStageDownloading, TaskStageStatusBlocked
	default:
		return TaskStageSourcing, TaskStageStatusPending
	}
}

func stageToLegacyStatus(stage TaskStage, status TaskStageStatus) TaskStatus {
	switch normalizeTaskStage(stage) {
	case TaskStageSourcing:
		if normalizeTaskStageStatus(status) == TaskStageStatusBlocked {
			return TaskStatusFailed
		}
		return TaskStatusPending
	case TaskStageDownloading:
		if normalizeTaskStageStatus(status) == TaskStageStatusBlocked {
			return TaskStatusFailed
		}
		if normalizeTaskStageStatus(status) == TaskStageStatusPending {
			return TaskStatusAdded
		}
		return TaskStatusDownloading
	case TaskStagePendingIngest, TaskStageTransferring, TaskStageScanning:
		if normalizeTaskStageStatus(status) == TaskStageStatusBlocked {
			return TaskStatusFailed
		}
		return TaskStatusCompleted
	case TaskStageCompleted:
		return TaskStatusCompleted
	default:
		return TaskStatusPending
	}
}

func normalizeTaskStageStatus(value TaskStageStatus) TaskStageStatus {
	switch value {
	case TaskStageStatusPending,
		TaskStageStatusRunning,
		TaskStageStatusBlocked,
		TaskStageStatusDone:
		return value
	default:
		return TaskStageStatusPending
	}
}

func taskStageLabel(stage TaskStage) string {
	switch normalizeTaskStage(stage) {
	case TaskStageSourcing:
		return "待选种"
	case TaskStageDownloading:
		return "下载中"
	case TaskStagePendingIngest:
		return "待入库"
	case TaskStageTransferring:
		return "搬运中"
	case TaskStageScanning:
		return "扫描中"
	case TaskStageCompleted:
		return "已完成"
	default:
		return "待选种"
	}
}

func taskStageStatusLabel(status TaskStageStatus) string {
	switch normalizeTaskStageStatus(status) {
	case TaskStageStatusPending:
		return "待处理"
	case TaskStageStatusRunning:
		return "进行中"
	case TaskStageStatusBlocked:
		return "受阻"
	case TaskStageStatusDone:
		return "已完成"
	default:
		return "待处理"
	}
}

func refreshTaskStageFields(task *Task) {
	if task == nil {
		return
	}
	if task.Stage == "" && task.Status != "" {
		task.Stage, task.StageStatus = legacyStatusToStage(task.Status)
	}
	if task.DownloadCompletedAt == nil && task.CompletedAt != nil {
		task.DownloadCompletedAt = cloneTime(task.CompletedAt)
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
	task.Stage = normalizeTaskStage(task.Stage)
	task.StageStatus = normalizeTaskStageStatus(task.StageStatus)
	task.StageLabel = taskStageLabel(task.Stage)
	task.StageStatusLabel = taskStageStatusLabel(task.StageStatus)
	task.StageErrorCode = strings.TrimSpace(task.StageErrorCode)
	task.StageErrorMessage = strings.TrimSpace(task.StageErrorMessage)
	task.Status = stageToLegacyStatus(task.Stage, task.StageStatus)
	task.CompletedAt = cloneTime(task.DownloadCompletedAt)
	task.StashMode = task.DeliveryMode
	task.StashSourcePath = task.MojiSourcePath
	task.StashTransferAction = task.TransferAction
	task.StashTransferPath = task.MojiTransferPath
	task.StashTransferError = task.TransferError
	task.StashJobID = task.StashScanJobID
	switch {
	case task.Stage == TaskStageTransferring && task.StageStatus == TaskStageStatusRunning:
		task.StashTransferStatus = "started"
	case task.Stage == TaskStageScanning || task.Stage == TaskStageCompleted:
		if task.MojiTransferPath != "" {
			task.StashTransferStatus = "completed"
		} else {
			task.StashTransferStatus = ""
		}
	case task.Stage == TaskStageTransferring && task.StageStatus == TaskStageStatusBlocked:
		task.StashTransferStatus = "failed"
	default:
		task.StashTransferStatus = ""
	}
	switch {
	case task.Stage == TaskStageScanning && task.StageStatus == TaskStageStatusRunning:
		task.StashScanStatus = "started"
	case task.Stage == TaskStageCompleted && task.StageStatus == TaskStageStatusDone:
		task.StashScanStatus = "completed"
	case task.StageStatus == TaskStageStatusBlocked && task.StashScanError != "":
		task.StashScanStatus = "failed"
	default:
		task.StashScanStatus = ""
	}
	if task.StageStatus == TaskStageStatusBlocked {
		task.Error = task.StageErrorMessage
	} else {
		task.Error = ""
	}
}

func setTaskStage(task *Task, stage TaskStage, status TaskStageStatus) {
	if task == nil {
		return
	}
	task.Stage = normalizeTaskStage(stage)
	task.StageStatus = normalizeTaskStageStatus(status)
	refreshTaskStageFields(task)
}

func clearTaskStageError(task *Task) {
	if task == nil {
		return
	}
	task.StageErrorCode = ""
	task.StageErrorMessage = ""
	refreshTaskStageFields(task)
}

func setTaskStageError(task *Task, code string, message string) {
	if task == nil {
		return
	}
	task.StageErrorCode = strings.TrimSpace(code)
	task.StageErrorMessage = strings.TrimSpace(message)
	refreshTaskStageFields(task)
}
