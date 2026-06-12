package graphqlapi

import "strings"

func isTaskTerminalStatus(status string) bool {
	return status == "completed" || status == "failed" || status == "cancelled" || status == "canceled" || status == "paused"
}

func isDownloadingStatus(status string) bool {
	return strings.Contains(status, "download") || strings.Contains(status, "sync") || strings.Contains(status, "queued") || strings.Contains(status, "stalled")
}

func isPendingScanStatus(status string) bool {
	if status == "" {
		return false
	}
	return status != "completed" && status != "done" && status != "failed" && status != "skipped" && status != "idle"
}
