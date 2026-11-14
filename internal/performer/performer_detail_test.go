package performer

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/stashboxcache"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type pagedStashBoxClient struct {
	pages     map[int][]*stashboxgraphql.SceneFragment
	inputs    []stashboxgraphql.SceneQueryInput
	performer *stashboxgraphql.PerformerFragment
}

type performerClientFactory struct{ client metadata.Client }

func (f performerClientFactory) NewClient(stash.StashBoxEndpoint) metadata.Client { return f.client }

func (c *pagedStashBoxClient) FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error) {
	return c.performer, nil
}

func (c *pagedStashBoxClient) FindSceneByID(_ context.Context, id string) (*stashboxgraphql.SceneFragment, error) {
	for _, page := range c.pages {
		for _, scene := range page {
			if scene.ID == id {
				return scene, nil
			}
		}
	}
	return nil, nil
}

func (*pagedStashBoxClient) SearchPerformer(context.Context, string) ([]*stashboxgraphql.PerformerFragment, error) {
	return nil, nil
}

func (*pagedStashBoxClient) SearchScene(context.Context, string) ([]*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (c *pagedStashBoxClient) QueryScenes(_ context.Context, input stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	c.inputs = append(c.inputs, input)
	return c.pages[input.Page], nil
}

func (c *pagedStashBoxClient) QueryScenesPage(ctx context.Context, input stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error) {
	scenes, err := c.QueryScenes(ctx, input)
	return stashboxpkg.ScenePage{Scenes: scenes, Count: 85}, err
}

func sceneFragments(start, count int) []*stashboxgraphql.SceneFragment {
	out := make([]*stashboxgraphql.SceneFragment, count)
	for index := range out {
		out[index] = &stashboxgraphql.SceneFragment{ID: fmt.Sprintf("scene-%03d", start+index)}
	}
	return out
}

func TestListPerformerScenesLoadsFortyItemBatchesOnDemand(t *testing.T) {
	clientPerformer := &stashboxgraphql.PerformerFragment{ID: "performer-1", Name: "Alice"}
	client := &pagedStashBoxClient{performer: clientPerformer, pages: map[int][]*stashboxgraphql.SceneFragment{
		1: sceneFragments(1, 40),
		2: sceneFragments(41, 40),
		3: sceneFragments(81, 5),
	}}
	for _, batch := range client.pages {
		for i, scene := range batch {
			scene.Performers = []*stashboxgraphql.PerformerAppearanceFragment{{Performer: clientPerformer}}
			if i%2 == 1 {
				scene.Performers = append(scene.Performers, &stashboxgraphql.PerformerAppearanceFragment{Performer: &stashboxgraphql.PerformerFragment{ID: "other", Name: "Beth"}})
			}
		}
	}
	oldDate, newDate, title := "2000-01-01", "2000-02-01", "Latest source work"
	client.pages[1][0].Date = &oldDate
	client.pages[1][30].Date = &newDate
	client.pages[1][30].Title = &title
	registry := metadata.NewRegistry(performerClientFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "box", Endpoint: "https://box/graphql"}})
	metadataService := metadata.NewService(nil, registry)
	cache, err := stashboxcache.New(filepath.Join(t.TempDir(), "cache.db"), func() stashboxcache.Config {
		return stashboxcache.Config{TTL: 24 * time.Hour, StaleRetention: 30 * 24 * time.Hour}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	metadataService.SetCache(cache)

	stashPerformer := &stashgraphql.PerformerFragment{
		ID:   "stash-performer-1",
		Name: "Alice",
		StashIds: []*stashgraphql.StashIDFragment{{
			Endpoint: "https://box/graphql",
			StashID:  clientPerformer.ID,
		}},
	}
	service, err := NewService(testStashClient{performer: stashPerformer}, metadataService, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	reader := &testUpcomingStateReader{}
	service.SetUpcomingStateReader(reader)
	first, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Page: 1, PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 24 || first.TotalCount != 85 || first.TotalPages != 4 || !first.TotalCountExact || first.StashBoxLoadedCount != 40 || len(client.inputs) != 1 {
		t.Fatalf("unexpected first page: %+v calls=%+v", first, client.inputs)
	}
	if len(reader.keys) != 24 || first.Items[0].UpcomingState != "WAITING" || first.Items[1].UpcomingState != "" {
		t.Fatalf("upcoming lookup must be bounded to visible page: keys=%v items=%+v", reader.keys, first.Items)
	}
	for _, cast := range []SceneCastFilter{SceneCastFilterSolo, SceneCastFilterGroup} {
		filtered, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Cast: cast, Page: 1, PageSize: 12})
		if err != nil || len(filtered.Items) != 12 || filtered.TotalCount != 20 || filtered.TotalCountExact || filtered.TotalPages != 0 || len(client.inputs) != 1 {
			t.Fatalf("filtered partial catalog reported the unfiltered remote total: %+v err=%v", filtered, err)
		}
		for _, item := range filtered.Items {
			if !matchesPerformerSceneCast(item, cast) {
				t.Fatal("unexpected cast in filtered page", item)
			}
		}
	}
	detail, err := service.GetPerformerDetail(context.Background(), stashPerformer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.StashBoxSceneCount != 85 || detail.SceneStatsComplete || detail.ActiveTaskCount != 0 {
		t.Fatalf("unexpected detail summary: %+v", detail)
	}
	if detail.LatestRelease == nil || detail.LatestRelease.Date != newDate || detail.LatestRelease.Title != title || len(client.inputs) != 1 {
		t.Fatalf("latest source release should use all loaded records without more requests: %+v calls=%d", detail.LatestRelease, len(client.inputs))
	}
	second, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Page: 2, PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 24 || second.StashBoxLoadedCount != 80 || len(client.inputs) != 2 || client.inputs[1].Page != 2 || client.inputs[1].PerPage != 40 {
		t.Fatalf("unexpected second page: %+v calls=%+v", second, client.inputs)
	}
	third, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Page: 3, PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Items) != 24 || len(client.inputs) != 2 {
		t.Fatalf("third page should reuse cached coverage: %+v calls=%+v", third, client.inputs)
	}
	last, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Page: 4, PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Items) != 13 || !last.CacheComplete || len(client.inputs) != 3 || client.inputs[2].Page != 3 || client.inputs[2].PerPage != 40 {
		t.Fatalf("unexpected final page: %+v calls=%+v", last, client.inputs)
	}
}

func TestCountActivePerformerSceneTasksCountsUniqueUnfinishedTasks(t *testing.T) {
	scenes := []Scene{
		{MojiTask: &SceneTask{ID: "active", Stage: taskruntime.TaskStageDownloading}},
		{MojiTask: &SceneTask{ID: "active", Stage: taskruntime.TaskStageDownloading}},
		{MojiTask: &SceneTask{ID: "waiting", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusPending}},
		{MojiTask: &SceneTask{ID: "blocked", Stage: taskruntime.TaskStageScanning, StageStatus: taskruntime.TaskStageStatusBlocked}},
		{MojiTask: &SceneTask{ID: "done", Stage: taskruntime.TaskStageCompleted}},
		{},
	}
	if got := countActivePerformerSceneTasks(scenes); got != 3 {
		t.Fatalf("got %d unfinished tasks, want 3", got)
	}
	if got := countActivePerformerSceneTasks(nil); got != 0 {
		t.Fatalf("got %d tasks for no scenes", got)
	}
}

func TestProfileDoesNotContactMetadataOrLoadScenes(t *testing.T) {
	// Deliberately omit metadata: basic Stash information must remain available.
	s := &Service{stash: testStashClient{performer: &stashgraphql.PerformerFragment{ID: "p1", Name: "Alice"}}, customFieldKey: DefaultCustomFieldKey}
	profile, err := s.GetPerformerProfile(context.Background(), "p1")
	if err != nil || profile.Performer.Name != "Alice" {
		t.Fatalf("profile=%+v error=%v", profile, err)
	}
}

type testUpcomingStateReader struct{ keys []string }

func (r *testUpcomingStateReader) UpcomingReleaseStates(_ context.Context, keys []string) (map[string]string, error) {
	r.keys = append([]string(nil), keys...)
	return map[string]string{keys[0]: "WAITING"}, nil
}

func TestPerformerCastFiltersBeforePagination(t *testing.T) {
	actor := &stashgraphql.PerformerFragment{ID: "actor", Name: "Alice"}
	scenes := make([]*stashgraphql.SceneFragment, 7)
	for i := range scenes {
		people := []*stashgraphql.SceneFragment_Performers{{ID: "actor", Name: "Alice"}}
		if i%2 == 1 {
			people = append(people, &stashgraphql.SceneFragment_Performers{ID: "other", Name: "Beth"})
		}
		if i == 6 {
			people = []*stashgraphql.SceneFragment_Performers{nil}
		}
		scenes[i] = &stashgraphql.SceneFragment{ID: fmt.Sprint(i), Performers: people}
	}
	service, err := NewService(testStashClient{performer: actor, scenes: scenes}, metadata.NewService(nil, metadata.NewRegistry(nil)), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, cast := range []SceneCastFilter{SceneCastFilterAll, SceneCastFilterSolo, SceneCastFilterGroup} {
		page, err := service.ListPerformerScenes(context.Background(), actor.ID, SceneQuery{Source: SceneSourceFilterStash, Cast: cast, Page: 2, PageSize: 2})
		expected := 3
		if cast == SceneCastFilterAll {
			expected = 7
		}
		if err != nil || page.TotalCount != expected || !page.TotalCountExact || page.Page != 2 {
			t.Fatal(cast, page, err)
		}
		for _, item := range page.Items {
			if !matchesPerformerSceneCast(item, cast) {
				t.Fatal("wrong cast", item)
			}
		}
		if cast != SceneCastFilterAll && len(page.Items) != 1 {
			t.Fatal("filter applied after pagination", page)
		}
	}
}
