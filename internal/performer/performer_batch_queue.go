package performer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

const MaxQueueSceneBatchSize = 100

var (
	ErrQueueSceneBatchEmpty    = errors.New("performer: scene batch requires at least one scene")
	ErrQueueSceneBatchTooLarge = errors.New("performer: scene batch exceeds maximum size")
)

func (s *Service) QueuePerformerScenes(ctx context.Context, performerID string, selections []QueueSceneSelection) (QueueScenesResult, error) {
	performerID = strings.TrimSpace(performerID)
	if performerID == "" {
		return QueueScenesResult{}, errors.New("performer: performer id is required")
	}
	if len(selections) == 0 {
		return QueueScenesResult{}, ErrQueueSceneBatchEmpty
	}
	if len(selections) > MaxQueueSceneBatchSize {
		return QueueScenesResult{}, fmt.Errorf("%w: maximum is %d", ErrQueueSceneBatchTooLarge, MaxQueueSceneBatchSize)
	}
	if s.taskCreator == nil {
		return QueueScenesResult{}, errors.New("performer: task creator is not configured")
	}

	seenKeys := make(map[string]struct{}, len(selections))
	for index := range selections {
		key := normalize(selections[index].Key)
		if key == "" {
			return QueueScenesResult{}, errors.New("performer: scene key is required")
		}
		if _, exists := seenKeys[key]; exists {
			return QueueScenesResult{}, fmt.Errorf("performer: duplicate scene key %q in request", selections[index].Key)
		}
		seenKeys[key] = struct{}{}
		selections[index].Key = strings.TrimSpace(selections[index].Key)
	}

	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		return QueueScenesResult{}, err
	}
	if performer == nil {
		return QueueScenesResult{}, fmt.Errorf("performer: performer %q not found", performerID)
	}

	scenes, _, _, _, _, _, _, err := s.loadPerformerScenes(ctx, performer, SceneQuery{Page: 1, PageSize: 100}, true, metadata.CachePreferred)
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
		}
	}

	if current.InLibrary {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusSkipped,
			ReasonCode:   "ALREADY_IN_LIBRARY",
			ResolvedCode: buildReleaseCode(current.Code, current.Title),
		}
	}

	if !current.HasStashBoxSource || current.StashBoxSceneID == "" || current.StashBoxEndpoint == "" {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusSkipped,
			ReasonCode:   "NO_STASHBOX_SOURCE",
			ResolvedCode: buildReleaseCode(current.Code, current.Title),
		}
	}

	resolvedCode := buildReleaseCode(current.Code, current.Title)
	if resolvedCode == "" {
		return QueueSceneResult{
			Key:        selection.Key,
			Status:     QueueSceneStatusSkipped,
			ReasonCode: "MISSING_CODE",
		}
	}
	task, err := s.taskCreator.QueueDiscoveredScene(ctx, current.StashBoxSceneID, current.StashBoxEndpoint)
	if err == nil {
		return QueueSceneResult{
			Key:          selection.Key,
			Status:       QueueSceneStatusQueued,
			ReasonCode:   "QUEUED",
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
	case errors.Is(err, taskruntime.ErrDuplicateTorrentTask):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "DUPLICATE_TORRENT_TASK"
	case errors.Is(err, taskruntime.ErrDuplicateLibraryCode):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "ALREADY_IN_LIBRARY"
	case errors.Is(err, taskruntime.ErrTaskCodeRequired):
		result.Status = QueueSceneStatusSkipped
		result.ReasonCode = "MISSING_CODE"
	default:
		result.Status = QueueSceneStatusFailed
		result.ReasonCode = "QUEUE_FAILED"
	}
	return result
}
