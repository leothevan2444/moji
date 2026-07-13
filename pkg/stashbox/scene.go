package stashbox

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (c *Client) QueryScenes(ctx context.Context, input graphql.SceneQueryInput) ([]*graphql.SceneFragment, error) {
	scenes, err := c.graphql.QueryScenes(ctx, input)
	if err != nil {
		return nil, err
	}
	return scenes.QueryScenes.Scenes, nil
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
