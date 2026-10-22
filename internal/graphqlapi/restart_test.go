package graphqlapi

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

type restartRuntime struct {
	fakeTaskRuntime
	restartedID string
}

func (r *restartRuntime) RestartTaskSourcing(_ context.Context, id string, _ taskruntime.StashScanner) (*taskruntime.Task, error) {
	r.restartedID = id
	return &taskruntime.Task{ID: id, Code: "SNOS-184", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusBlocked, StageErrorCode: taskruntime.TaskStageErrorManualSource}, nil
}

func TestRestartSourcingGraphQLReturnsOriginalTaskAtManualSelection(t *testing.T) {
	runtime := &restartRuntime{}
	r := NewResolver(nil, nil, runtime, nil, "test")
	var response struct {
		Data struct {
			RestartTaskSourcing struct {
				ID, Stage, StageStatus, StageErrorCode string
				Progress                               float64
				Files                                  []any
				StashSceneIDs                          []string
				TorrentHash                            *string
			}
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, r, "mutation { restartTaskSourcing(id:\"task-184\") { id stage stageStatus stageErrorCode progress files { index } stashSceneIds torrentHash } }", &response)
	got := response.Data.RestartTaskSourcing
	if len(response.Errors) > 0 || runtime.restartedID != "task-184" || got.ID != "task-184" || got.Stage != "SOURCING" || got.StageStatus != "BLOCKED" || got.StageErrorCode != "MANUAL_SOURCE_REQUIRED" || got.Progress != 0 || len(got.Files) != 0 || len(got.StashSceneIDs) != 0 || got.TorrentHash != nil {
		t.Fatalf("unexpected restart response: %+v", response)
	}
}
