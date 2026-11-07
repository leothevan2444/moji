package stash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

// IngestIdentity fences saved metadata plans when the Stash connection changes.
// The digest includes credentials without persisting or exposing them.
func (c *Client) IngestIdentity() string {
	if c.configProvider == nil {
		return ""
	}
	raw, _ := json.Marshal(c.configProvider())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (c *Client) nativeIdentifyOptions(ctx context.Context) (bool, error) {
	r, err := c.resolve().IngestIdentifyCapabilities(ctx)
	if err != nil {
		return false, err
	}
	if r.Type != nil {
		for _, field := range r.Type.Fields {
			if field != nil && field.Name == "performerGenders" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) ScrapeIngestScene(ctx context.Context, source graphql.ScraperSourceInput, input graphql.ScrapeSingleSceneInput, images bool) ([]*graphql.ScrapedScene, error) {
	modern, err := c.nativeIdentifyOptions(ctx)
	if err != nil {
		return nil, err
	}
	var raw []byte
	if modern {
		r, e := c.resolve().ScrapeIngestScene(ctx, source, input, images)
		if e != nil {
			return nil, e
		}
		raw, err = json.Marshal(r.ScrapeSingleScene)
	} else {
		r, e := c.resolve().ScrapeIngestSceneLegacy(ctx, source, input, images)
		if e != nil {
			return nil, e
		}
		raw, err = json.Marshal(r.ScrapeSingleScene)
	}
	if err != nil {
		return nil, err
	}
	var result []*graphql.ScrapedScene
	err = json.Unmarshal(raw, &result)
	return result, err
}

func (c *Client) ingestWriteClient(identity string) (*graphql.Client, error) {
	client := c.resolve()
	c.mu.RLock()
	raw, _ := json.Marshal(c.lastConfig)
	sum := sha256.Sum256(raw)
	valid := c.graphql == client && identity != "" && hex.EncodeToString(sum[:]) == identity
	c.mu.RUnlock()
	if !valid {
		return nil, errors.New("Stash connection changed before metadata saving")
	}
	return client, nil
}

// GraphQL distinguishes omission from an explicit empty list. The generated
// input uses omitempty, so preserve empty replacement lists at this boundary.
type ingestSceneInput struct{ graphql.SceneUpdateInput }

func (v ingestSceneInput) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(v.SceneUpdateInput)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for key, ids := range map[string][]string{"performer_ids": v.PerformerIds, "tag_ids": v.TagIds, "urls": v.Urls} {
		if ids != nil && len(ids) == 0 {
			fields[key] = json.RawMessage("[]")
		}
	}
	if v.StashIds != nil && len(v.StashIds) == 0 {
		fields["stash_ids"] = json.RawMessage("[]")
	}
	return json.Marshal(fields)
}

func (c *Client) SaveIngestScene(ctx context.Context, identity string, input graphql.SceneUpdateInput) error {
	client, err := c.ingestWriteClient(identity)
	if err != nil {
		return err
	}
	var response graphql.SaveIngestScene
	err = client.Client.Post(ctx, "SaveIngestScene", graphql.SaveIngestSceneDocument, &response, map[string]any{"input": ingestSceneInput{input}})
	r := &response
	if err != nil {
		return err
	}
	if r.SceneUpdate == nil || r.SceneUpdate.ID != input.ID {
		return errors.New("Stash did not confirm the saved scene")
	}
	return nil
}

func (c *Client) MetadataGenerate(ctx context.Context, input graphql.GenerateMetadataInput) (string, error) {
	response, err := c.resolve().MetadataGenerate(ctx, input)
	if err != nil {
		return "", err
	}
	return response.MetadataGenerate, nil
}

type ingestIdentifyInput struct{ graphql.IdentifyMetadataInput }

func nativeOptionsJSON(options *graphql.IdentifyMetadataOptionsInput) (json.RawMessage, error) {
	if options == nil {
		return nil, nil
	}
	raw, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if options.PerformerGenders != nil {
		raw, err = json.Marshal(options.PerformerGenders)
		if err != nil {
			return nil, err
		}
		fields["performerGenders"] = raw
	}
	return json.Marshal(fields)
}
func (v ingestIdentifyInput) MarshalJSON() ([]byte, error) {
	var sources []json.RawMessage
	for _, source := range v.Sources {
		if source == nil {
			continue
		}
		options, err := nativeOptionsJSON(source.Options)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(struct {
			Source  *graphql.ScraperSourceInput `json:"source"`
			Options json.RawMessage             `json:"options,omitempty"`
		}{source.Source, options})
		if err != nil {
			return nil, err
		}
		sources = append(sources, raw)
	}
	options, err := nativeOptionsJSON(v.Options)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Sources  []json.RawMessage `json:"sources"`
		Options  json.RawMessage   `json:"options,omitempty"`
		SceneIDs []string          `json:"sceneIDs,omitempty"`
		Paths    []string          `json:"paths,omitempty"`
	}{sources, options, v.SceneIDs, v.Paths})
}
func (c *Client) MetadataIdentify(ctx context.Context, input graphql.IdentifyMetadataInput) (string, error) {
	var response graphql.MetadataIdentify
	err := c.resolve().Client.Post(ctx, "MetadataIdentify", graphql.MetadataIdentifyDocument, &response, map[string]any{"input": ingestIdentifyInput{input}})
	if err != nil {
		return "", err
	}
	return response.MetadataIdentify, nil
}

func (c *Client) IngestLogs(ctx context.Context) ([]*graphql.IngestLogs_Logs, error) {
	response, err := c.resolve().IngestLogs(ctx)
	if err != nil {
		return nil, err
	}
	return response.Logs, nil
}

func (c *Client) IngestIdentifyDefaults(ctx context.Context) (*graphql.IdentifyMetadataInput, error) {
	modern, err := c.nativeIdentifyOptions(ctx)
	if err != nil {
		return nil, err
	}
	var data []byte
	var genders [][]graphql.GenderEnum
	if modern {
		response, e := c.resolve().IngestDefaultsNative(ctx)
		if e != nil {
			return nil, e
		}
		identify := response.Configuration.Defaults.Identify
		if identify == nil {
			return nil, errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")
		}
		data, err = json.Marshal(identify)
		var defaults []graphql.GenderEnum
		if identify.Options != nil {
			defaults = identify.Options.PerformerGenders
		}
		genders = append(genders, defaults)
		for _, source := range identify.Sources {
			var values []graphql.GenderEnum
			if source != nil && source.Options != nil {
				values = source.Options.PerformerGenders
			}
			genders = append(genders, values)
		}
	} else {
		response, e := c.resolve().IngestDefaults(ctx)
		if e != nil {
			return nil, e
		}
		if response.Configuration.Defaults.Identify == nil {
			return nil, errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")
		}
		data, err = json.Marshal(response.Configuration.Defaults.Identify)
	}
	if err != nil {
		return nil, err
	}
	var input graphql.IdentifyMetadataInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	if len(genders) > 0 {
		if input.Options != nil && genders[0] != nil {
			input.Options.PerformerGenders = append([]graphql.GenderEnum{}, genders[0]...)
		}
		for i, source := range input.Sources {
			if source != nil && source.Options != nil && i+1 < len(genders) && genders[i+1] != nil {
				source.Options.PerformerGenders = append([]graphql.GenderEnum{}, genders[i+1]...)
			}
		}
	}
	if len(input.Sources) == 0 {
		return nil, errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")
	}
	for _, source := range input.Sources {
		if source == nil || source.Source == nil || source.Source.StashBoxEndpoint != nil || source.Source.StashBoxIndex == nil {
			continue
		}
		configured, e := c.resolve().Configuration(ctx)
		if e != nil {
			return nil, e
		}
		index := *source.Source.StashBoxIndex
		if index < 0 || index >= len(configured.Configuration.General.StashBoxes) {
			return nil, errors.New("configured StashBox source index is unavailable")
		}
		endpoint := configured.Configuration.General.StashBoxes[index].Endpoint
		source.Source.StashBoxEndpoint = &endpoint
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

// Remove only the scene record and generated assets. Downloaded and delivered
// media are deleted separately by Moji's persisted task cleanup.
func (c *Client) RemoveIngestScene(ctx context.Context, id string) error {
	no, yes := false, true
	response, err := c.resolve().RemoveIngestScene(ctx, graphql.SceneDestroyInput{ID: id, DeleteFile: &no, DeleteGenerated: &yes})
	if err != nil {
		return err
	}
	if !response.SceneDestroy {
		return errors.New("Stash did not remove the scene")
	}
	return nil
}
