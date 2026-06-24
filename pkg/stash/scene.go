package stash

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (c *Client) FindScenes(ctx context.Context, sceneFilter *graphql.SceneFilterType, filter *graphql.FindFilterType) ([]*graphql.SceneFragment, error) {
	scenes, err := c.graphql.FindScenes(context.Background(), sceneFilter, filter)
	if err != nil {
		return nil, err
	}
	return scenes.FindScenes.Scenes, nil
}
