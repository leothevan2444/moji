package stashbox

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (c *Client) FindPerformerByID(ctx context.Context, id string) (*graphql.PerformerFragment, error) {
	performer, err := c.graphql.FindPerformerByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return performer.FindPerformer, nil
}

func (c *Client) SearchPerformer(ctx context.Context, term string) ([]*graphql.PerformerFragment, error) {
	performers, err := c.graphql.SearchPerformer(context.Background(), term)
	if err != nil {
		return nil, err
	}
	return performers.SearchPerformer, nil
}

func (c *Client) QueryPerformers(ctx context.Context, input graphql.PerformerQueryInput) ([]*graphql.PerformerFragment, error) {
	performers, err := c.graphql.QueryPerformers(context.Background(), input)
	if err != nil {
		return nil, err
	}
	return performers.QueryPerformers.Performers, nil
}
