package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/downloader"
)

func (s *Service) QueuePerformerScenes(ctx context.Context, performerID string, selections []QueuePerformerSceneSelection) (QueuePerformerScenesResult, error) {
	performerID = strings.TrimSpace(performerID)
	if performerID == "" {
		return QueuePerformerScenesResult{}, errors.New("subscription: performer id is required")
	}
	if len(selections) == 0 {
		return QueuePerformerScenesResult{}, errors.New("subscription: at least one scene selection is required")
	}
	if s.taskCreator == nil {
		return QueuePerformerScenesResult{}, errors.New("subscription: downloader is not configured")
	}

	seenKeys := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		key := normalize(selection.Key)
		if key == "" {
			return QueuePerformerScenesResult{}, errors.New("subscription: scene key is required")
		}
		if _, exists := seenKeys[key]; exists {
			return QueuePerformerScenesResult{}, fmt.Errorf("subscription: duplicate scene key %q in request", selection.Key)
		}
		seenKeys[key] = struct{}{}
	}

	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		return QueuePerformerScenesResult{}, err
	}
	if performer == nil {
		return QueuePerformerScenesResult{}, fmt.Errorf("subscription: performer %q not found", performerID)
	}

	scenes, _, _, err := s.loadPerformerScenes(ctx, performer)
	if err != nil {
		return QueuePerformerScenesResult{}, err
	}
	byKey := make(map[string]PerformerScene, len(scenes))
	for _, scene := range scenes {
		byKey[scene.Key] = scene
	}

	result := QueuePerformerScenesResult{
		QueuedTasks: make([]*downloader.Task, 0, len(selections)),
		Results:     make([]QueuePerformerSceneResult, 0, len(selections)),
		Summary: QueuePerformerScenesSummary{
			RequestedCount: len(selections),
		},
	}

	for _, selection := range selections {
		itemResult := s.queuePerformerSceneSelection(ctx, byKey, selection)
		result.Results = append(result.Results, itemResult)
		switch itemResult.Status {
		case QueuePerformerSceneStatusQueued:
			result.Summary.QueuedCount++
			if itemResult.Task != nil {
				result.QueuedTasks = append(result.QueuedTasks, itemResult.Task)
			}
		case QueuePerformerSceneStatusSkipped:
			result.Summary.SkippedCount++
		default:
			result.Summary.FailedCount++
		}
	}

	return result, nil
}

func (s *Service) queuePerformerSceneSelection(ctx context.Context, byKey map[string]PerformerScene, selection QueuePerformerSceneSelection) QueuePerformerSceneResult {
	current, ok := byKey[selection.Key]
	if !ok {
		return QueuePerformerSceneResult{
			Key:        selection.Key,
			Status:     QueuePerformerSceneStatusFailed,
			ReasonCode: "SCENE_NOT_FOUND",
			Message:    "当前演员页中未找到该作品",
		}
	}

	if current.InLibrary {
		return QueuePerformerSceneResult{
			Key:           selection.Key,
			Status:        QueuePerformerSceneStatusSkipped,
			ReasonCode:    "ALREADY_IN_LIBRARY",
			Message:       "作品已在库中，跳过创建任务",
			ResolvedQuery: buildReleaseQuery(current.Code, current.Title),
		}
	}

	if !current.HasStashBoxSource || current.StashBoxSceneID == "" || current.StashBoxEndpoint == "" {
		return QueuePerformerSceneResult{
			Key:           selection.Key,
			Status:        QueuePerformerSceneStatusSkipped,
			ReasonCode:    "NO_STASHBOX_SOURCE",
			Message:       "缺少可用于下载的 StashBox 场景来源",
			ResolvedQuery: buildReleaseQuery(current.Code, current.Title),
		}
	}

	resolvedQuery := buildReleaseQuery(current.Code, current.Title)
	if resolvedQuery == "" {
		return QueuePerformerSceneResult{
			Key:        selection.Key,
			Status:     QueuePerformerSceneStatusSkipped,
			ReasonCode: "MISSING_CODE",
			Message:    "作品缺少可稳定解析的番号",
		}
	}
	task, err := s.taskCreator.QueueDiscoveredScene(ctx, current.StashBoxSceneID, current.StashBoxEndpoint)
	if err == nil {
		return QueuePerformerSceneResult{
			Key:           selection.Key,
			Status:        QueuePerformerSceneStatusQueued,
			ReasonCode:    "QUEUED",
			Message:       "已创建下载任务",
			Task:          task,
			ResolvedQuery: resolvedQuery,
		}
	}
	return mapPerformerSceneQueueError(selection.Key, resolvedQuery, task, err)
}

func mapPerformerSceneQueueError(key, resolvedQuery string, task *downloader.Task, err error) QueuePerformerSceneResult {
	result := QueuePerformerSceneResult{
		Key:           key,
		Task:          task,
		ResolvedQuery: resolvedQuery,
	}
	switch {
	case errors.Is(err, downloader.ErrDuplicateCodeTask):
		result.Status = QueuePerformerSceneStatusSkipped
		result.ReasonCode = "DUPLICATE_CODE_TASK"
		result.Message = "同一番号已存在下载任务"
	case errors.Is(err, downloader.ErrDuplicateTorrentTask):
		result.Status = QueuePerformerSceneStatusSkipped
		result.ReasonCode = "DUPLICATE_TORRENT_TASK"
		result.Message = "同一 torrent 或 magnet 已存在下载任务"
	case errors.Is(err, downloader.ErrDuplicateLibraryCode):
		result.Status = QueuePerformerSceneStatusSkipped
		result.ReasonCode = "ALREADY_IN_LIBRARY"
		result.Message = "作品已在库中，跳过创建任务"
	case errors.Is(err, downloader.ErrTaskCodeRequired):
		result.Status = QueuePerformerSceneStatusSkipped
		result.ReasonCode = "MISSING_CODE"
		result.Message = "作品缺少可稳定解析的番号"
	default:
		result.Status = QueuePerformerSceneStatusFailed
		result.ReasonCode = "QUEUE_FAILED"
		result.Message = err.Error()
	}
	return result
}
