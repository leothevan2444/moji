package stashbox

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (c *Client) QueryScenes(ctx context.Context, input graphql.SceneQueryInput) ([]*graphql.SceneFragment, error) {
	scenes, err := c.graphql.QueryScenes(context.Background(), input)
	if err != nil {
		return nil, err
	}
	return scenes.QueryScenes.Scenes, nil
}
