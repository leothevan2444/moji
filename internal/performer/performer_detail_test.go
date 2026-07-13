package performer

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/stashboxcache"
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

func (*pagedStashBoxClient) FindSceneByID(context.Context, string) (*stashboxgraphql.SceneFragment, error) {
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

	first, err := service.ListPerformerScenes(context.Background(), stashPerformer.ID, SceneQuery{Source: SceneSourceFilterStashBox, Page: 1, PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 24 || first.TotalCount != 85 || first.TotalPages != 4 || !first.TotalCountExact || first.StashBoxLoadedCount != 40 || len(client.inputs) != 1 {
		t.Fatalf("unexpected first page: %+v calls=%+v", first, client.inputs)
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
