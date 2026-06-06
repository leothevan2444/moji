package stash

import (
	"context"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (c *Client) MetadataScan(ctx context.Context, input graphql.ScanMetadataInput) (string, error) {
	resp, err := c.graphql.MetadataScan(ctx, input)
	if err != nil {
		return "", err
	}

	return resp.MetadataScan, nil
}

func (c *Client) FindJob(ctx context.Context, id string) (*graphql.FindJob_FindJob, error) {
	resp, err := c.graphql.FindJob(ctx, graphql.FindJobInput{ID: id})
	if err != nil {
		return nil, err
	}

	return resp.FindJob, nil
}
