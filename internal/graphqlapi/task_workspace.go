package graphqlapi

import (
	"context"
	"fmt"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func taskVersion(t *taskruntime.Task) string {
	return fmt.Sprintf("%020d:%020d", t.Revision, t.UpdatedAt.UnixNano())
}
func taskSummaryToModel(t *taskruntime.Task) *model.TaskSummary {
	if t == nil {
		return nil
	}
	source := t.Source
	if source == "" {
		source = taskruntime.TaskSourceManual
	}
	sceneID := ""
	if ids := t.StashSceneIDs(); len(ids) > 0 {
		sceneID = ids[0]
	}
	message := []rune(t.StageErrorMessage)
	if len(message) > 512 {
		message = append(message[:512], '…')
	}
	return &model.TaskSummary{
		StashSceneID:          nilIfEmpty(sceneID),
		ID:                    t.ID,
		Version:               taskVersion(t),
		Revision:              t.Revision,
		Source:                model.TaskSource(source),
		Code:                  t.Code,
		Stage:                 model.TaskStage(t.Stage),
		StageStatus:           model.TaskStageStatus(t.StageStatus),
		StageLabel:            t.StageLabel,
		StageStatusLabel:      t.StageStatusLabel,
		StageErrorCode:        nilIfEmpty(t.StageErrorCode),
		StageErrorMessage:     nilIfEmpty(string(message)),
		TorrentName:           nilIfEmpty(t.TorrentName),
		Progress:              t.Progress,
		QbittorrentState:      nilIfEmpty(t.QBittorrentState),
		StashIngestPhase:      nilIfEmpty(t.StashIngestPhase),
		SourcingNextCheckAt:   formatOptionalTime(t.SourcingNextCheckAt),
		SourcingLastCheckedAt: formatOptionalTime(t.SourcingLastCheckedAt),
		SourcingCompletedAt:   formatOptionalTime(t.SourcingCompletedAt),
		DownloadCompletedAt:   formatOptionalTime(t.DownloadCompletedAt),
		IngestCompletedAt:     formatOptionalTime(t.IngestCompletedAt),
		LastProgressAt:        formatOptionalTime(t.LastProgressAt),
		CreatedAt:             formatTime(t.CreatedAt),
		UpdatedAt:             formatTime(t.UpdatedAt),
	}
}
func taskSummaryModels(tasks []*taskruntime.Task) []*model.TaskSummary {
	out := make([]*model.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskSummaryToModel(t))
	}
	return out
}
func workspaceStatsToModel(s taskruntime.TaskStats) *model.DashboardStats {
	return &model.DashboardStats{
		Total:        s.Total,
		Active:       s.Active,
		Completed:    s.Completed,
		Downloading:  s.Downloading,
		PendingScans: s.PendingScans,
		Failed:       s.Failed,
		Attention:    s.Attention,
		Waiting:      s.Waiting,
	}
}

type taskWorkspaceRuntime interface {
	TaskPage(context.Context, taskruntime.TaskPageInput) (taskruntime.TaskPage, error)
	AttentionQueue(context.Context) (taskruntime.TaskPage, error)
	HomeTaskSnapshot(context.Context) (taskruntime.HomeTaskSnapshot, error)
}

func (r *queryResolver) workspaceRuntime() (taskWorkspaceRuntime, error) {
	if r.TaskRuntime == nil {
		return nil, nil
	}
	p, ok := r.TaskRuntime.(taskWorkspaceRuntime)
	if !ok {
		return nil, fmt.Errorf("task workspace unavailable")
	}
	return p, nil
}
