package stashbox

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type ScenePage struct {
	Scenes []*graphql.SceneFragment
	Count  int
}

func (c *Client) QueryScenesPage(ctx context.Context, input graphql.SceneQueryInput) (ScenePage, error) {
	result, err := c.graphql.QueryScenes(ctx, input)
	if err != nil {
		return ScenePage{}, err
	}
	return ScenePage{Scenes: result.QueryScenes.Scenes, Count: result.QueryScenes.Count}, nil
}

func (c *Client) QueryScenes(ctx context.Context, input graphql.SceneQueryInput) ([]*graphql.SceneFragment, error) {
	page, err := c.QueryScenesPage(ctx, input)
	if err != nil {
		return nil, err
	}
	return page.Scenes, nil
}

func (c *Client) SearchScene(ctx context.Context, term string) ([]*graphql.SceneFragment, error) {
	resp, err := c.graphql.SearchScene(ctx, term)
	if err != nil {
		return nil, err
	}
	return resp.SearchScene, nil
}

func (c *Client) FindSceneByID(ctx context.Context, id string) (*graphql.SceneFragment, error) {
	resp, err := c.graphql.FindSceneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return resp.FindScene, nil
}
