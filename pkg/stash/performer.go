package stash

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (c *Client) FindPerformerByID(ctx context.Context, id string) (*graphql.PerformerFragment, error) {
	performer, err := c.graphql.FindPerformerByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return performer.FindPerformer, nil
}

func (c *Client) AllPerformers(ctx context.Context) ([]*graphql.PerformerFragment, error) {
	performers, err := c.graphql.AllPerformers(context.Background())
	if err != nil {
		return nil, err
	}
	return performers.AllPerformers, nil
}
