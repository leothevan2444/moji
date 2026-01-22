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

func (c *Client) FindPerformers(ctx context.Context, performerFilter *graphql.PerformerFilterType,
	filter *graphql.FindFilterType, performerIds []int, ids []string) ([]*graphql.PerformerFragment, error) {
	performers, err := c.graphql.FindPerformers(context.Background(), performerFilter, filter, performerIds, ids)
	if err != nil {
		return nil, err
	}
	return performers.FindPerformers.Performers, nil
}

func (c *Client) AllPerformers(ctx context.Context) ([]*graphql.PerformerFragment, error) {
	performers, err := c.graphql.AllPerformers(context.Background())
	if err != nil {
		return nil, err
	}
	return performers.AllPerformers, nil
}
