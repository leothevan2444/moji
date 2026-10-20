package performer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
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

	result := QueueScenesResult{
		QueuedTasks: make([]*taskruntime.Task, 0, len(selections)),
		Results:     make([]QueueSceneResult, 0, len(selections)),
		Summary: QueueScenesSummary{
			RequestedCount: len(selections),
		},
	}

	for _, selection := range selections {
		scene, found, lookupErr := s.resolveQueueScene(ctx, performer, selection.Key)
		itemResult := QueueSceneResult{Key: selection.Key, Status: QueueSceneStatusFailed, ReasonCode: "QUEUE_FAILED"}
		if lookupErr != nil {
			logging.Warnf("performer: selected scene lookup failed performer=%s key=%s: %v", performerID, selection.Key, lookupErr)
		}
		if lookupErr == nil {
			byKey := map[string]Scene{}
			if found {
				byKey[selection.Key] = scene
			}
			itemResult = s.queuePerformerSceneSelection(ctx, byKey, selection)
		}
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
	task, err := s.taskCreator.QueueResolvedScene(ctx, resolvedCode, current.Title)
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

// Resolve only the selected scene. Never enumerate a performer's entire catalogue
// or register images while creating a task.
func (s *Service) resolveQueueScene(ctx context.Context, performer *stashgraphql.PerformerFragment, key string) (Scene, bool, error) {
	if strings.HasPrefix(key, "stash:") && strings.TrimPrefix(key, "stash:") != "" {
		// Local scene selections cannot create a download, irrespective of ownership.
		return Scene{Key: key, InLibrary: true}, true, nil
	}
	for _, mapping := range performer.StashIds {
		if mapping == nil || mapping.StashID == "" {
			continue
		}
		prefix := "stashbox:" + endpointKey(mapping.Endpoint) + ":"
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		id := strings.TrimPrefix(key, prefix)
		if id == "" {
			return Scene{}, false, nil
		}
		client, ok := s.metadata.Get(mapping.Endpoint)
		if !ok {
			return Scene{}, false, nil
		}
		remote, err := client.FindSceneByID(ctx, id)
		if err != nil {
			return Scene{}, false, err
		}
		if remote == nil || remote.ID != id {
			return Scene{}, false, nil
		}
		belongs := false
		for _, appearance := range remote.Performers {
			if appearance != nil && appearance.Performer != nil && appearance.Performer.ID == mapping.StashID {
				belongs = true
				break
			}
		}
		if !belongs {
			return Scene{}, false, nil
		}
		return stashBoxSceneToPerformerScene(remote, mapping.Endpoint), true, nil
	}
	return Scene{}, false, nil
}
