package performer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

func (s *Service) QueuePerformerScenes(ctx context.Context, performerID string, selections []QueueSceneSelection) (QueueScenesResult, error) {
	performerID = strings.TrimSpace(performerID)
	if performerID == "" {
		return QueueScenesResult{}, errors.New("performer: performer id is required")
	}
	if len(selections) == 0 {
		return QueueScenesResult{}, errors.New("performer: at least one scene selection is required")
	}
	if s.taskCreator == nil {
		return QueueScenesResult{}, errors.New("performer: task creator is not configured")
	}

	seenKeys := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		key := normalize(selection.Key)
		if key == "" {
			return QueueScenesResult{}, errors.New("performer: scene key is required")
		}
		if _, exists := seenKeys[key]; exists {
			return QueueScenesResult{}, fmt.Errorf("performer: duplicate scene key %q in request", selection.Key)
		}
		seenKeys[key] = struct{}{}
	}

	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		return QueueScenesResult{}, err
	}
	if performer == nil {
		return QueueScenesResult{}, fmt.Errorf("performer: performer %q not found", performerID)
	}

	scenes, _, _, err := s.loadPerformerScenes(ctx, performer)
	if err != nil {
		return QueueScenesResult{}, err
	}
	byKey := make(map[string]Scene, len(scenes))
	for _, scene := range scenes {
		byKey[scene.Key] = scene
	}

	result := QueueScenesResult{
		QueuedTasks: make([]*taskruntime.Task, 0, len(selections)),
		Results:     make([]QueueSceneResult, 0, len(selections)),
		Summary: QueueScenesSummary{
			RequestedCount: len(selections),
		},
	}

	for _, selection := range selections {
		itemResult := s.queuePerformerSceneSelection(ctx, byKey, selection)
		result.Results = append(result.Results, itemResult)
		switch itemResult.Status {
		case QueueSceneStatusQueued:
			result.Summary.QueuedCount++
			if itemResult.Task != nil {
				result.QueuedTasks = append(result.QueuedTasks, itemResult.Task)
			}
		case QueueSceneStatusSkipped:
			result.Summary.SkippedCount++
		default:
			result.Summary.FailedCount++
		}
	}

	return result, nil
}

func (s *Service) queuePerformerSceneSelection(ctx context.Context, byKey map[string]Scene, selection QueueSceneSelection) QueueSceneResult {
	current, ok := byKey[selection.Key]
	if !ok {
		return QueueSceneResult{
			Key:        selection.Key,
			Status:     QueueSceneStatusFailed,
			ReasonCode: "SCENE_NOT_FOUND",
			Message:    "当前演员页中未找到该作品",
		}
	}

	if current.InLibrary {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusSkipped,
			ReasonCode:   "ALREADY_IN_LIBRARY",
			Message:      "作品已在库中，跳过创建任务",
			ResolvedCode: buildReleaseCode(current.Code, current.Title),
		}
	}

	if !current.HasStashBoxSource || current.StashBoxSceneID == "" || current.StashBoxEndpoint == "" {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusSkipped,
			ReasonCode:   "NO_STASHBOX_SOURCE",
			Message:      "缺少可用于下载的 StashBox 场景来源",
			ResolvedCode: buildReleaseCode(current.Code, current.Title),
		}
	}

	resolvedCode := buildReleaseCode(current.Code, current.Title)
	if resolvedCode == "" {
		return QueueSceneResult{
			Key:        selection.Key,
			Status:     QueueSceneStatusSkipped,
			ReasonCode: "MISSING_CODE",
			Message:    "作品缺少可稳定解析的番号",
		}
	}
	task, err := s.taskCreator.QueueDiscoveredScene(ctx, current.StashBoxSceneID, current.StashBoxEndpoint)
	if err == nil {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusQueued,
			ReasonCode:   "QUEUED",
			Message:      "已创建下载任务",
			Task:         task,
			ResolvedCode: resolvedCode,
		}
	}
	return mapPerformerSceneQueueError(selection.Key, resolvedCode, task, err)
}

func mapPerformerSceneQueueError(key, resolvedCode string, task *taskruntime.Task, err error) QueueSceneResult {
	result := QueueSceneResult{
		Key:          key,
		Task:         task,
		ResolvedCode: resolvedCode,
	}
	switch {
	case errors.Is(err, taskruntime.ErrDuplicateCodeTask):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "DUPLICATE_CODE_TASK"
		result.Message = "同一番号已存在下载任务"
	case errors.Is(err, taskruntime.ErrDuplicateTorrentTask):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "DUPLICATE_TORRENT_TASK"
		result.Message = "同一 torrent 或 magnet 已存在下载任务"
	case errors.Is(err, taskruntime.ErrDuplicateLibraryCode):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "ALREADY_IN_LIBRARY"
		result.Message = "作品已在库中，跳过创建任务"
	case errors.Is(err, taskruntime.ErrTaskCodeRequired):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "MISSING_CODE"
		result.Message = "作品缺少可稳定解析的番号"
	default:
		result.Status = QueueSceneStatusFailed
		result.ReasonCode = "QUEUE_FAILED"
		result.Message = err.Error()
	}
	return result
}
