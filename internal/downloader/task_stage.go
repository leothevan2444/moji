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
	task.Stage = normalizeTaskStage(task.Stage)
	task.StageStatus = normalizeTaskStageStatus(task.StageStatus)
	task.StageLabel = taskStageLabel(task.Stage)
	task.StageStatusLabel = taskStageStatusLabel(task.StageStatus)
	task.StageErrorCode = strings.TrimSpace(task.StageErrorCode)
	task.StageErrorMessage = strings.TrimSpace(task.StageErrorMessage)
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
