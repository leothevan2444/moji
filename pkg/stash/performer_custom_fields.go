package stash

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (c *Client) UpdatePerformerCustomFields(ctx context.Context, id string, partial map[string]any, remove []string) (*graphql.PerformerFragment, error) {
	input := graphql.PerformerUpdateInput{
		ID: id,
		CustomFields: &graphql.CustomFieldsInput{
			Partial: partial,
			Remove:  remove,
		},
	}

	resp, err := c.graphql.UpdatePerformerCustomFields(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp.PerformerUpdate, nil
}
