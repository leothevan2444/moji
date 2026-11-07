package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// DeleteTaskOptions applies only to this request, never to saved settings.
type DeleteTaskOptions struct {
	DeleteStashScene bool
}

var ErrStashTaskDeletion = errors.New("Stash deletion is incomplete; Moji task was kept")

type StashSceneDeleter interface {
	DeletionIdentity() string
	DeleteTaskScenes(context.Context, string, string, []string) error
}

type stashDeletionIntent struct {
	Identity string   `json:"identity"`
	Path     string   `json:"path"`
	SceneIDs []string `json:"scene_ids"`
}

func (s *Service) deleteTaskStashScenes(ctx context.Context, task *Task, stash StashSceneDeleter) error {
	var ids []string
	if task.StashSceneIDsJSON != "" && json.Unmarshal([]byte(task.StashSceneIDsJSON), &ids) != nil {
		return errors.New("invalid linked Stash scene IDs; task was kept")
	}
	if len(ids) == 0 {
		return nil
	}
	if stash == nil || stash.DeletionIdentity() == "" {
		return errors.New("Stash scene deletion is unavailable; task was kept")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	const kind = "TASK_DELETE_STASH"
	intent := stashDeletionIntent{stash.DeletionIdentity(), task.StashScanPath, ids}
	if reader, ok := s.store.(taskCommandReader); ok {
		raw, err := reader.TaskCommandPayload(ctx, task, kind)
		if err != nil {
			return err
		}
		if len(raw) > 0 {
			var previous struct {
				Intent json.RawMessage `json:"intent"`
			}
			if json.Unmarshal(raw, &previous) != nil {
				return errors.New("invalid Stash deletion intent")
			}
			var old stashDeletionIntent
			if json.Unmarshal(previous.Intent, &old) != nil || !reflect.DeepEqual(old, intent) {
				return errors.New("Stash connection or linked scenes changed since the deletion attempt; task was kept")
			}
		}
	}
	err := s.runTaskCommand(ctx, task, kind, taskCommandTarget(task, kind), map[string]any{"intent": intent}, func() error {
		return stash.DeleteTaskScenes(ctx, intent.Identity, intent.Path, intent.SceneIDs)
	})
	if err != nil {
		return fmt.Errorf("Stash scene deletion was not confirmed; task was kept: %w", err)
	}
	return nil
}
