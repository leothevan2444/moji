package stash

import (
	"context"
	"errors"
	"strconv"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// Query by a filter rather than findScenes(ids), which fails the entire query
// for an already removed ID. An empty successful result confirms absence.
func (c *Client) FindSceneForDeletion(ctx context.Context, identity, id string) (*gql.IngestScenes_FindScenes_Scenes, error) {
	client, err := c.ingestWriteClient(identity)
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return nil, errors.New("invalid Stash scene ID")
	}
	limit := 2
	r, err := client.IngestScenes(ctx, &gql.SceneFilterType{ID: &gql.IntCriterionInput{Value: n, Modifier: gql.CriterionModifierEquals}}, &gql.FindFilterType{PerPage: &limit}, nil)
	if err != nil {
		return nil, err
	}
	if len(r.FindScenes.Scenes) == 0 {
		return nil, nil
	}
	if len(r.FindScenes.Scenes) != 1 || r.FindScenes.Scenes[0] == nil || r.FindScenes.Scenes[0].ID != id {
		return nil, errors.New("Stash scene identity did not match")
	}
	return r.FindScenes.Scenes[0], nil
}

func (c *Client) DeleteSceneWithFiles(ctx context.Context, identity, id string) error {
	client, err := c.ingestWriteClient(identity)
	if err != nil {
		return err
	}
	yes := true
	r, err := client.RemoveIngestScene(ctx, gql.SceneDestroyInput{ID: id, DeleteFile: &yes, DeleteGenerated: &yes})
	if err != nil {
		return err
	}
	if !r.SceneDestroy {
		return errors.New("Stash did not confirm scene deletion")
	}
	return nil
}
