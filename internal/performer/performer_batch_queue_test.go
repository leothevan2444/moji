package performer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type queueTestTaskCreator struct {
	task  *taskruntime.Task
	err   error
	calls int
}

func (f *queueTestTaskCreator) QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error) {
	f.calls++
	return f.task, f.err
}

func TestQueuePerformerScenesRejectsInvalidBatchSizeBeforeDependencies(t *testing.T) {
	service := &Service{}

	if _, err := service.QueuePerformerScenes(context.Background(), "performer-1", nil); !errors.Is(err, ErrQueueSceneBatchEmpty) {
		t.Fatalf("empty batch error = %v, want %v", err, ErrQueueSceneBatchEmpty)
	}

	tooMany := make([]QueueSceneSelection, MaxQueueSceneBatchSize+1)
	for index := range tooMany {
		tooMany[index].Key = fmt.Sprintf("scene-%03d", index)
	}
	if _, err := service.QueuePerformerScenes(context.Background(), "performer-1", tooMany); !errors.Is(err, ErrQueueSceneBatchTooLarge) {
		t.Fatalf("oversized batch error = %v, want %v", err, ErrQueueSceneBatchTooLarge)
	}
}

func TestQueuePerformerScenesRejectsDuplicateKeysBeforeSourceLookup(t *testing.T) {
	service := &Service{taskCreator: &queueTestTaskCreator{}}
	_, err := service.QueuePerformerScenes(context.Background(), "performer-1", []QueueSceneSelection{
		{Key: "scene-1"},
		{Key: " SCENE-1 "},
	})
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestQueuePerformerScenesAcceptsMaximumAndPreservesOrder(t *testing.T) {
	performer := &stashgraphql.PerformerFragment{ID: "performer-1", Name: "Alice"}
	scenes := make([]*stashgraphql.SceneFragment, MaxQueueSceneBatchSize)
	selections := make([]QueueSceneSelection, MaxQueueSceneBatchSize)
	for index := range scenes {
		id := fmt.Sprintf("scene-%03d", index)
		scenes[index] = &stashgraphql.SceneFragment{ID: id}
		selections[index] = QueueSceneSelection{Key: "stash:" + id}
	}
	creator := &queueTestTaskCreator{}
	service, err := NewService(
		testStashClient{performer: performer, scenes: scenes},
		metadata.NewService(nil, metadata.NewRegistry(nil)),
		creator,
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.QueuePerformerScenes(context.Background(), performer.ID, selections)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != MaxQueueSceneBatchSize || result.Summary.RequestedCount != MaxQueueSceneBatchSize {
		t.Fatalf("unexpected result size: results=%d summary=%+v", len(result.Results), result.Summary)
	}
	for index, item := range result.Results {
		if item.Key != selections[index].Key || item.ReasonCode != "ALREADY_IN_LIBRARY" {
			t.Fatalf("result %d = %+v, want key %q in input order", index, item, selections[index].Key)
		}
	}
	if creator.calls != 0 {
		t.Fatalf("in-library scenes should not create tasks, got %d calls", creator.calls)
	}
}

func TestQueuePerformerScenesReturnsOrderedPartialSuccess(t *testing.T) {
	const endpoint = "https://box.example/graphql"
	code := "ABC-123"
	remote := &stashboxgraphql.SceneFragment{ID: "remote-1", Code: &code}
	client := &pagedStashBoxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "box-performer-1", Name: "Alice"},
		pages:     map[int][]*stashboxgraphql.SceneFragment{1: {remote}},
	}
	registry := metadata.NewRegistry(performerClientFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "box", Endpoint: endpoint}})
	performer := &stashgraphql.PerformerFragment{
		ID:   "performer-1",
		Name: "Alice",
		StashIds: []*stashgraphql.StashIDFragment{{
			Endpoint: endpoint,
			StashID:  "box-performer-1",
		}},
	}
	creator := &queueTestTaskCreator{task: &taskruntime.Task{ID: "task-1"}}
	service, err := NewService(
		testStashClient{
			performer: performer,
			scenes:    []*stashgraphql.SceneFragment{{ID: "local-1", Code: &code}},
		},
		metadata.NewService(nil, registry),
		creator,
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	remoteKey := "stashbox:" + endpointKey(endpoint) + ":" + remote.ID
	result, err := service.QueuePerformerScenes(context.Background(), performer.ID, []QueueSceneSelection{
		{Key: remoteKey},
		{Key: "stash:local-1"},
		{Key: "missing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != (QueueScenesSummary{RequestedCount: 3, QueuedCount: 1, SkippedCount: 1, FailedCount: 1}) {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	want := []struct {
		key    string
		status QueueSceneStatus
		reason string
	}{
		{key: remoteKey, status: QueueSceneStatusQueued, reason: "QUEUED"},
		{key: "stash:local-1", status: QueueSceneStatusSkipped, reason: "ALREADY_IN_LIBRARY"},
		{key: "missing", status: QueueSceneStatusFailed, reason: "SCENE_NOT_FOUND"},
	}
	if len(result.Results) != len(want) {
		t.Fatalf("result count = %d, want %d", len(result.Results), len(want))
	}
	for index, expected := range want {
		item := result.Results[index]
		if item.Key != expected.key || item.Status != expected.status || item.ReasonCode != expected.reason {
			t.Fatalf("result %d = %+v, want key=%q status=%s reason=%s", index, item, expected.key, expected.status, expected.reason)
		}
	}
	if creator.calls != 1 || len(result.QueuedTasks) != 1 || result.QueuedTasks[0].ID != "task-1" {
		t.Fatalf("unexpected task creation: calls=%d tasks=%+v", creator.calls, result.QueuedTasks)
	}
}

func TestQueuePerformerSceneSelectionReasonCodes(t *testing.T) {
	valid := Scene{Key: "valid", Code: "ABC-123", HasStashBoxSource: true, StashBoxSceneID: "box-scene", StashBoxEndpoint: "https://box.example/graphql"}
	tests := []struct {
		name    string
		scenes  map[string]Scene
		key     string
		creator *queueTestTaskCreator
		status  QueueSceneStatus
		reason  string
	}{
		{name: "not found", scenes: map[string]Scene{}, key: "missing", creator: &queueTestTaskCreator{}, status: QueueSceneStatusFailed, reason: "SCENE_NOT_FOUND"},
		{name: "in library", scenes: map[string]Scene{"scene": {Key: "scene", Code: "ABC-123", InLibrary: true}}, key: "scene", creator: &queueTestTaskCreator{}, status: QueueSceneStatusSkipped, reason: "ALREADY_IN_LIBRARY"},
		{name: "no StashBox source", scenes: map[string]Scene{"scene": {Key: "scene", Code: "ABC-123"}}, key: "scene", creator: &queueTestTaskCreator{}, status: QueueSceneStatusSkipped, reason: "NO_STASHBOX_SOURCE"},
		{name: "missing code", scenes: map[string]Scene{"scene": {Key: "scene", HasStashBoxSource: true, StashBoxSceneID: "id", StashBoxEndpoint: "endpoint"}}, key: "scene", creator: &queueTestTaskCreator{}, status: QueueSceneStatusSkipped, reason: "MISSING_CODE"},
		{name: "queued", scenes: map[string]Scene{"valid": valid}, key: "valid", creator: &queueTestTaskCreator{task: &taskruntime.Task{ID: "task-1"}}, status: QueueSceneStatusQueued, reason: "QUEUED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &Service{taskCreator: test.creator}
			result := service.queuePerformerSceneSelection(context.Background(), test.scenes, QueueSceneSelection{Key: test.key})
			if result.Status != test.status || result.ReasonCode != test.reason {
				t.Fatalf("result = %+v, want status %s reason %s", result, test.status, test.reason)
			}
		})
	}
}

func TestMapPerformerSceneQueueErrorReasonCodes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status QueueSceneStatus
		reason string
	}{
		{name: "duplicate code", err: taskruntime.ErrDuplicateCodeTask, status: QueueSceneStatusSkipped, reason: "DUPLICATE_CODE_TASK"},
		{name: "duplicate torrent", err: taskruntime.ErrDuplicateTorrentTask, status: QueueSceneStatusSkipped, reason: "DUPLICATE_TORRENT_TASK"},
		{name: "library code", err: taskruntime.ErrDuplicateLibraryCode, status: QueueSceneStatusSkipped, reason: "ALREADY_IN_LIBRARY"},
		{name: "missing code", err: taskruntime.ErrTaskCodeRequired, status: QueueSceneStatusSkipped, reason: "MISSING_CODE"},
		{name: "queue failed", err: errors.New("upstream secret detail"), status: QueueSceneStatusFailed, reason: "QUEUE_FAILED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := mapPerformerSceneQueueError("scene", "ABC-123", nil, test.err)
			if result.Status != test.status || result.ReasonCode != test.reason {
				t.Fatalf("result = %+v, want status %s reason %s", result, test.status, test.reason)
			}
		})
	}
}
