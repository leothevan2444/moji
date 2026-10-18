package graphqlapi

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/stashsync"
)

type taskSceneReader interface {
	FindIngestScenes(context.Context, string, []string) ([]stashsync.IngestScene, error)
}

type taskImageRegistrar interface {
	Register(context.Context, imagecache.Descriptor) (string, error)
}

func (r *Resolver) loadTaskStashScenes(ctx context.Context, id string) ([]*model.TaskStashScene, error) {
	result := []*model.TaskStashScene{}
	if r.TaskRuntime == nil {
		return result, errors.New("task runtime is not configured")
	}
	task, err := r.TaskRuntime.FindTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return result, nil
	}
	ids := task.StashSceneIDs()
	if len(ids) == 0 {
		return result, nil
	}
	reader, ok := r.Stash.(taskSceneReader)
	if !ok {
		return nil, errors.New("Stash scene information is unavailable")
	}
	scenes, err := reader.FindIngestScenes(ctx, task.StashScanPath, ids)
	if err != nil {
		return nil, err
	}
	settings := r.RuntimeSettings
	if r.SettingsEditor != nil {
		settings = r.SettingsEditor.Snapshot()
	}
	byID := make(map[string]stashsync.IngestScene, len(scenes))
	for _, scene := range scenes {
		byID[scene.ID] = scene
	}
	seen := map[string]bool{}
	for _, id := range ids {
		scene, found := byID[id]
		if !found || seen[id] {
			continue
		}
		seen[id] = true
		item := &model.TaskStashScene{ID: scene.ID, Title: scene.Title, Code: scene.Code, Date: scene.Date, Studio: scene.StudioName, Performers: append([]string{}, scene.PerformerNames...)}
		if settings != nil {
			item.URL = stashTaskSceneURL(settings.Stash.URL, scene.ID)
			if registrar, ok := r.ImageCache.(taskImageRegistrar); ok && strings.TrimSpace(scene.ImageURL) != "" {
				image, err := registrar.Register(ctx, imagecache.Descriptor{Kind: imagecache.SourceStash, InstanceURL: settings.Stash.URL, ImageURL: scene.ImageURL, APIKey: settings.Stash.APIKey})
				if err == nil {
					item.ImageURL = nilIfEmpty(image)
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func stashTaskSceneURL(base, id string) *string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}
	parsed.User, parsed.RawQuery, parsed.Fragment = nil, "", ""
	return nilIfEmpty(parsed.JoinPath("scenes", url.PathEscape(id)).String())
}
