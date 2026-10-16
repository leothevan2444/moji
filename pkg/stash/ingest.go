package stash

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (c *Client) MetadataGenerate(ctx context.Context, input graphql.GenerateMetadataInput) (string, error) {
	response, err := c.resolve().MetadataGenerate(ctx, input)
	if err != nil {
		return "", err
	}
	return response.MetadataGenerate, nil
}

func (c *Client) MetadataIdentify(ctx context.Context, input graphql.IdentifyMetadataInput) (string, error) {
	response, err := c.resolve().MetadataIdentify(ctx, input)
	if err != nil {
		return "", err
	}
	return response.MetadataIdentify, nil
}

func (c *Client) IngestIdentifyDefaults(ctx context.Context) (*graphql.IdentifyMetadataInput, error) {
	response, err := c.resolve().IngestDefaults(ctx)
	if err != nil {
		return nil, err
	}
	if response.Configuration.Defaults.Identify == nil {
		return nil, errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")
	}
	// Stash uses the same field names for its saved settings and mutation inputs.
	data, err := json.Marshal(response.Configuration.Defaults.Identify)
	if err != nil {
		return nil, err
	}
	var input graphql.IdentifyMetadataInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	if len(input.Sources) == 0 {
		return nil, errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")
	}
	return &input, nil
}

func (c *Client) FindIngestScenes(ctx context.Context, path string, ids []string) ([]*graphql.IngestScenes_FindScenes_Scenes, error) {
	var sceneFilter *graphql.SceneFilterType
	if len(ids) == 0 {
		sceneFilter = &graphql.SceneFilterType{Path: &graphql.StringCriterionInput{Value: path, Modifier: graphql.CriterionModifierIncludes}}
	}
	all := -1
	response, err := c.resolve().IngestScenes(ctx, sceneFilter, &graphql.FindFilterType{PerPage: &all}, ids)
	if err != nil {
		return nil, err
	}
	return response.FindScenes.Scenes, nil
}
