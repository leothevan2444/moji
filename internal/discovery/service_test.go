package discovery

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type testClientFactory struct {
	client metadata.Client
}

func (f testClientFactory) NewClient(stash.StashBoxEndpoint) metadata.Client { return f.client }

type testStashBoxClient struct {
	scenes []*stashboxgraphql.SceneFragment
}

func (testStashBoxClient) FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error) {
	return nil, nil
}

func (testStashBoxClient) FindSceneByID(context.Context, string) (*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (testStashBoxClient) SearchPerformer(context.Context, string) ([]*stashboxgraphql.PerformerFragment, error) {
	return nil, nil
}

func (c testStashBoxClient) SearchScene(context.Context, string) ([]*stashboxgraphql.SceneFragment, error) {
	return c.scenes, nil
}

func (testStashBoxClient) QueryScenes(context.Context, stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (testStashBoxClient) QueryScenesPage(context.Context, stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error) {
	return stashboxpkg.ScenePage{}, nil
}

type testTaskCreator struct {
	task *taskruntime.Task
}

func (c testTaskCreator) QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error) {
	return c.task, nil
}

func TestServiceSearchesAndQueuesWithOnlyDiscoveryDependencies(t *testing.T) {
	title := "Scene A"
	client := testStashBoxClient{scenes: []*stashboxgraphql.SceneFragment{{ID: "scene-1", Title: &title}}}
	registry := metadata.NewRegistry(testClientFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Primary", Endpoint: "https://box.example/graphql"}})
	queued := &taskruntime.Task{ID: "task-1"}
	service := NewService(metadata.NewService(nil, registry), testTaskCreator{task: queued}, nil)

	page, err := service.Search(context.Background(), "Scene", 10, SortRelevance)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].SceneID != "scene-1" || page.UsedStashBox == nil || page.UsedStashBox.Name != "Primary" {
		t.Fatalf("unexpected page: %+v", page)
	}
	task, err := service.Queue(context.Background(), "scene-1", "https://box.example/graphql")
	if err != nil || task != queued {
		t.Fatalf("Queue: task=%+v err=%v", task, err)
	}
}
