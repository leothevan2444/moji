package performer

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/metadata"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type pagedStashBoxClient struct {
	pages  map[int][]*stashboxgraphql.SceneFragment
	inputs []stashboxgraphql.SceneQueryInput
}

func (*pagedStashBoxClient) FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error) {
	return nil, nil
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

func TestFetchStashBoxScenesLoadsEveryPageWithFortyItems(t *testing.T) {
	client := &pagedStashBoxClient{pages: map[int][]*stashboxgraphql.SceneFragment{
		1: sceneFragments(1, 40),
		2: sceneFragments(41, 40),
		3: sceneFragments(81, 5),
	}}
	target := &metadata.MatchedPerformer{
		Client:    client,
		Performer: &stashboxgraphql.PerformerFragment{ID: "performer-1"},
	}

	scenes, err := (&Service{}).fetchStashBoxScenes(context.Background(), target)
	if err != nil {
		t.Fatalf("fetchStashBoxScenes: %v", err)
	}
	if len(scenes) != 85 {
		t.Fatalf("scene count = %d, want 85", len(scenes))
	}
	if len(client.inputs) != 3 {
		t.Fatalf("query count = %d, want 3", len(client.inputs))
	}
	for index, input := range client.inputs {
		wantPage := index + 1
		if input.Page != wantPage {
			t.Errorf("query %d page = %d, want %d", index, input.Page, wantPage)
		}
		if input.PerPage != 40 {
			t.Errorf("query %d perPage = %d, want 40", index, input.PerPage)
		}
		if input.Performers == nil || len(input.Performers.Value) != 1 || input.Performers.Value[0] != "performer-1" {
			t.Errorf("query %d performer filter = %+v", index, input.Performers)
		}
	}
}

func sceneFragments(start, count int) []*stashboxgraphql.SceneFragment {
	out := make([]*stashboxgraphql.SceneFragment, count)
	for index := range out {
		out[index] = &stashboxgraphql.SceneFragment{ID: string(rune(start + index))}
	}
	return out
}
