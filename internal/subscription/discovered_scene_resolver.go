package subscription

import (
	"context"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/taskflow"
)

type StashBoxSceneRegistry interface {
	Get(endpoint string) (StashboxClient, bool)
}

type discoveredSceneResolver struct {
	registry StashBoxSceneRegistry
}

func NewDiscoveredSceneResolver(registry StashBoxSceneRegistry) taskflow.DiscoveredSceneResolver {
	return discoveredSceneResolver{registry: registry}
}

func (r discoveredSceneResolver) ResolveDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (taskflow.ResolvedScene, error) {
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: scene id is required")
	}
	stashBoxEndpoint = strings.TrimSpace(stashBoxEndpoint)
	if stashBoxEndpoint == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: stash-box endpoint is required")
	}
	if r.registry == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: stash-box resolver is not configured")
	}

	client, ok := r.registry.Get(stashBoxEndpoint)
	if !ok || client == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: stash-box endpoint %q is not available", stashBoxEndpoint)
	}

	scene, err := client.FindSceneByID(ctx, sceneID)
	if err != nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: load scene %q from %q: %w", sceneID, stashBoxEndpoint, err)
	}
	if scene == nil {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: scene %q not found in %q", sceneID, stashBoxEndpoint)
	}

	code := stringValue(scene.Code)
	if buildReleaseQuery(code, stringValue(scene.Title)) == "" {
		return taskflow.ResolvedScene{}, fmt.Errorf("subscription: scene %q is missing code", sceneID)
	}

	return taskflow.ResolvedScene{
		Code:  code,
		Title: stringValue(scene.Title),
	}, nil
}
