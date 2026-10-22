package stashsync

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// RemoveIngestScenes scopes deletion to the task-owned delivery path and rejects
// scenes which also link files outside it. Query by path to make retries safe
// after some scene records have already been removed.
func (s *Service) RemoveIngestScenes(ctx context.Context, path string, ids []string) error {
	client, err := s.ingestClient()
	if err != nil {
		return err
	}
	remover, ok := s.client.(interface {
		RemoveIngestScene(context.Context, string) error
	})
	if !ok {
		return errors.New("Stash scene removal is unavailable")
	}
	root := filepath.Clean(path)
	if !filepath.IsAbs(root) || root == string(filepath.Separator) {
		return errors.New("invalid Stash delivery path")
	}
	scenes, err := client.FindIngestScenes(ctx, path, nil)
	if err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	remove := []string{}
	for _, scene := range scenes {
		if scene == nil {
			continue
		}
		inside, outside := false, false
		for _, file := range scene.Files {
			if file == nil {
				continue
			}
			if strings.HasPrefix(filepath.Clean(file.Path), root+string(filepath.Separator)) {
				inside = true
			} else {
				outside = true
			}
		}
		if !inside {
			continue
		}
		if outside || (len(wanted) > 0 && !wanted[scene.ID]) {
			return fmt.Errorf("Stash scene %s is not exclusively owned by this task", scene.ID)
		}
		remove = append(remove, scene.ID)
	}
	for _, id := range remove {
		if err := remover.RemoveIngestScene(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

var ErrIdentifySourcesMissing = errors.New("configure Identify sources in Stash Settings > Tasks before ingesting")

type IngestClient interface {
	MetadataGenerate(context.Context, stashgraphql.GenerateMetadataInput) (string, error)
	MetadataIdentify(context.Context, stashgraphql.IdentifyMetadataInput) (string, error)
	IngestIdentifyDefaults(context.Context) (*stashgraphql.IdentifyMetadataInput, error)
	FindIngestScenes(context.Context, string, []string) ([]*stashgraphql.IngestScenes_FindScenes_Scenes, error)
}

type IngestScene struct {
	ID             string
	Title          string
	Code           string
	Date           string
	StudioID       string
	StudioName     string
	PerformerNames []string
	ImageURL       string
	PerformerCount int
	Paths          []string
}

func (s *Service) ingestClient() (IngestClient, error) {
	client, ok := s.client.(IngestClient)
	if !ok {
		return nil, errors.New("Stash client does not support generation and identification")
	}
	return client, nil
}

func (s *Service) MetadataGenerate(ctx context.Context, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", errors.New("Stash generation requires scene IDs")
	}
	client, err := s.ingestClient()
	if err != nil {
		return "", err
	}
	enabled := true
	transcodes := s.CurrentConfig().GenerateTranscodes
	// pHash must exist before stash-box identification. Generate playback assets
	// without overwriting existing files; opt-in transcodes are generated only if needed.
	return client.MetadataGenerate(ctx, stashgraphql.GenerateMetadataInput{
		SceneIDs: ids, Covers: &enabled, Previews: &enabled, Sprites: &enabled,
		Phashes: &enabled, Transcodes: &transcodes,
	})
}

func (s *Service) MetadataIdentify(ctx context.Context, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", errors.New("Stash identification requires scene IDs")
	}
	client, err := s.ingestClient()
	if err != nil {
		return "", err
	}
	input, err := client.IngestIdentifyDefaults(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "configure Identify sources in Stash Settings > Tasks") {
			return "", ErrIdentifySourcesMissing
		}
		return "", err
	}
	if input == nil || len(input.Sources) == 0 {
		return "", ErrIdentifySourcesMissing
	}
	input.SceneIDs = ids
	input.Paths = nil
	return client.MetadataIdentify(ctx, *input)
}

func (s *Service) FindIngestScenes(ctx context.Context, path string, ids []string) ([]IngestScene, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("Stash ingest path is required")
	}
	client, err := s.ingestClient()
	if err != nil {
		return nil, err
	}
	scenes, err := client.FindIngestScenes(ctx, path, ids)
	if err != nil {
		return nil, err
	}
	result := []IngestScene{}
	root := filepath.Clean(path)
	for _, scene := range scenes {
		if scene == nil {
			continue
		}
		item := IngestScene{ID: scene.ID, PerformerCount: len(scene.Performers)}
		if scene.Title != nil {
			item.Title = *scene.Title
		}
		if scene.Code != nil {
			item.Code = *scene.Code
		}
		if scene.Date != nil {
			item.Date = *scene.Date
		}
		if scene.Studio != nil {
			item.StudioID = scene.Studio.ID
			item.StudioName = scene.Studio.Name
		}
		if scene.Paths.Screenshot != nil {
			item.ImageURL = *scene.Paths.Screenshot
		}
		for _, performer := range scene.Performers {
			if performer != nil && strings.TrimSpace(performer.Name) != "" {
				item.PerformerNames = append(item.PerformerNames, performer.Name)
			}
		}
		for _, file := range scene.Files {
			if file == nil {
				continue
			}
			clean := filepath.Clean(file.Path)
			// The GraphQL substring filter is only a first pass: exclude neighbouring
			// torrents with similar names and files outside the delivered directory.
			if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
				item.Paths = append(item.Paths, file.Path)
			}
		}
		if len(item.Paths) > 0 {
			result = append(result, item)
		}
	}
	return result, nil
}
