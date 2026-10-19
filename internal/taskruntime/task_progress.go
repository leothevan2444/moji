package taskruntime

import "strings"

// trackTaskProgress separates observable execution changes from routine sync writes.
// Store callers receive the recorded value too, so event snapshots and mutation
// responses carry the same timestamp as the persisted task.
func trackTaskProgress(previous, next *Task) {
	if previous == nil {
		if !next.CreatedAt.IsZero() {
			next.LastProgressAt = cloneTime(&next.CreatedAt)
		}
		return
	}
	next.LastProgressAt = cloneTime(previous.LastProgressAt)
	if sameTaskProgress(previous, next) || next.UpdatedAt.IsZero() {
		return
	}
	if next.LastProgressAt == nil || next.UpdatedAt.After(*next.LastProgressAt) {
		next.LastProgressAt = cloneTime(&next.UpdatedAt)
	}
}

func sameTaskProgress(a, b *Task) bool {
	return a.Stage == b.Stage && a.StageStatus == b.StageStatus && a.Progress == b.Progress &&
		strings.EqualFold(strings.TrimSpace(a.QBittorrentState), strings.TrimSpace(b.QBittorrentState)) &&
		taskProgressPhase(a) == taskProgressPhase(b) &&
		a.StageErrorCode == b.StageErrorCode && a.StageErrorMessage == b.StageErrorMessage &&
		a.TransferError == b.TransferError && a.StashScanError == b.StashScanError
}

func taskProgressPhase(task *Task) string {
	phase := strings.ToUpper(strings.TrimSpace(task.StashIngestPhase))
	if task.Stage == TaskStageScanning && phase == "" {
		return StashIngestScan
	}
	return phase
}
