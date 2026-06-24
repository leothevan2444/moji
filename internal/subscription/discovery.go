package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/logging"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (s *Service) SearchPreferredStashBoxScenes(ctx context.Context, query string, limit int) (DiscoverScenePage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return DiscoverScenePage{}, errors.New("subscription: query is required")
	}
	if s.stashbox == nil || len(s.orderedEndpoints()) == 0 {
		return DiscoverScenePage{}, errors.New("subscription: no stash-box endpoints configured in Stash")
	}
	if limit <= 0 {
		limit = 24
	}

	var (
		lastErr error
	)
	for idx, box := range s.orderedEndpoints() {
		client, ok := s.stashbox.Get(box.Endpoint)
		if !ok || client == nil {
			continue
		}

		items, err := client.SearchScene(ctx, query)
		if err != nil {
			lastErr = err
			logging.Warnf("subscription: discovery search failed for endpoint=%s query=%q: %v", box.Endpoint, query, err)
			continue
		}
		if len(items) == 0 {
			continue
		}

		page := DiscoverScenePage{
			Items:         make([]DiscoveredScene, 0, min(limit, len(items))),
			UsedStashBox:  &MatchedStashBox{Name: box.Name, Endpoint: box.Endpoint},
			FallbackCount: idx,
			SearchedQuery: query,
		}
		for _, scene := range items[:min(limit, len(items))] {
			if scene == nil {
				continue
			}
			page.Items = append(page.Items, discoveredSceneFromStashBox(scene, box))
		}
		return page, nil
	}

	if lastErr != nil {
		return DiscoverScenePage{Items: []DiscoveredScene{}, SearchedQuery: query}, lastErr
	}
	return DiscoverScenePage{Items: []DiscoveredScene{}, SearchedQuery: query}, nil
}

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*downloader.Task, error) {
	if s.downloader == nil {
		return nil, errors.New("subscription: downloader is not configured")
	}
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		return nil, errors.New("subscription: scene id is required")
	}
	stashBoxEndpoint = strings.TrimSpace(stashBoxEndpoint)
	if stashBoxEndpoint == "" {
		return nil, errors.New("subscription: stash-box endpoint is required")
	}
	client, ok := s.stashbox.Get(stashBoxEndpoint)
	if !ok || client == nil {
		return nil, fmt.Errorf("subscription: stash-box endpoint %q is not available", stashBoxEndpoint)
	}

	scene, err := client.FindSceneByID(ctx, sceneID)
	if err != nil {
		return nil, fmt.Errorf("subscription: load scene %q from %q: %w", sceneID, stashBoxEndpoint, err)
	}
	if scene == nil {
		return nil, fmt.Errorf("subscription: scene %q not found in %q", sceneID, stashBoxEndpoint)
	}

	query := buildReleaseQuery(stringValue(scene.Code), stringValue(scene.Title))
	if query == "" {
		return nil, fmt.Errorf("subscription: scene %q has no usable code or title", sceneID)
	}
	return s.downloader.DownloadMediaContext(ctx, downloader.DownloadRequest{
		Source: downloader.TaskSourceSearch,
		Query:  query,
	})
}

func discoveredSceneFromStashBox(scene *stashboxgraphql.SceneFragment, box StashBoxEndpoint) DiscoveredScene {
	performerNames := make([]string, 0, len(scene.Performers))
	for _, appearance := range scene.Performers {
		if appearance == nil || appearance.Performer == nil {
			continue
		}
		name := strings.TrimSpace(appearance.Performer.Name)
		if name != "" {
			performerNames = append(performerNames, name)
		}
	}

	return DiscoveredScene{
		Key:              "stashbox-search:" + endpointKey(box.Endpoint) + ":" + scene.ID,
		SceneID:          scene.ID,
		StashBoxEndpoint: box.Endpoint,
		StashBoxName:     box.Name,
		Title:            strings.TrimSpace(stringValue(scene.Title)),
		DurationSeconds:  scene.Duration,
		Code:             strings.TrimSpace(stringValue(scene.Code)),
		Date:             strings.TrimSpace(stringValue(scene.Date)),
		StudioName:       stashBoxStudioName(scene),
		ImageURL:         stashBoxSceneImageURL(scene),
		URL:              stashBoxSceneURL(scene),
		PerformerNames:   performerNames,
		DerivedQuery:     buildReleaseQuery(stringValue(scene.Code), stringValue(scene.Title)),
	}
}

func stashBoxStudioName(scene *stashboxgraphql.SceneFragment) string {
	if scene == nil || scene.Studio == nil {
		return ""
	}
	return strings.TrimSpace(scene.Studio.Name)
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
