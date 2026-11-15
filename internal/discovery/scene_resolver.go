package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskflow"
)

type StashBoxSceneRegistry interface {
	Get(endpoint string) (metadata.Client, bool)
}

type discoveredSceneResolver struct {
	registry StashBoxSceneRegistry
	images   ImageProxy
}

func NewDiscoveredSceneResolver(registry StashBoxSceneRegistry, images ...ImageProxy) taskflow.DiscoveredSceneResolver {
	resolver := discoveredSceneResolver{registry: registry}
	if len(images) > 0 {
		resolver.images = images[0]
	}
	return resolver
}

func (r discoveredSceneResolver) ResolveDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (taskflow.ResolvedScene, error) {
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: scene id is required")
	}
	stashBoxEndpoint = strings.TrimSpace(stashBoxEndpoint)
	if stashBoxEndpoint == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: stash-box endpoint is required")
	}
	if r.registry == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: stash-box resolver is not configured")
	}

	client, ok := r.registry.Get(stashBoxEndpoint)
	if !ok || client == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: stash-box endpoint %q is not available", stashBoxEndpoint)
	}

	scene, err := client.FindSceneByID(ctx, sceneID)
	if err != nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: load scene %q from %q: %w", sceneID, stashBoxEndpoint, err)
	}
	if scene == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: scene %q not found in %q", sceneID, stashBoxEndpoint)
	}

	code := value(scene.Code)
	if code == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("discovery: scene %q is missing code", sceneID)
	}

	snapshot := taskflow.ResolvedScene{Endpoint: film.Endpoint(stashBoxEndpoint), SceneID: scene.ID, DurationSeconds: scene.Duration,
		Code:  code,
		Date:  value(scene.Date),
		Title: value(scene.Title),
	}
	if scene.Studio != nil {
		snapshot.Studio = film.Person{ID: scene.Studio.ID, Name: scene.Studio.Name}
	}
	for _, p := range scene.Performers {
		if p != nil && p.Performer != nil {
			snapshot.Performers = append(snapshot.Performers, film.Person{ID: p.Performer.ID, Name: p.Performer.Name})
		}
	}
	snapshot.ImageURL = sceneImage(scene)
	if r.images != nil {
		snapshot.ImageURL = r.images(ctx, stashBoxEndpoint, snapshot.ImageURL)
	}
	snapshot.URL = sceneURL(scene)
	return snapshot, nil
}
