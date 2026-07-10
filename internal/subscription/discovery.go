package subscription

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (s *Service) SearchPreferredStashBoxScenes(ctx context.Context, query string, limit int, sortBy DiscoverSort) (DiscoverScenePage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return DiscoverScenePage{}, errors.New("subscription: query is required")
	}
	if s.stashbox == nil || len(s.orderedEndpoints()) == 0 {
		return DiscoverScenePage{}, errors.New("subscription: no stash-box endpoints configured in Stash")
	}
	if limit <= 0 {
		limit = 50
	}
	if sortBy == "" {
		sortBy = DiscoverSortRelevance
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
		for _, scene := range items {
			if scene == nil {
				continue
			}
			page.Items = append(page.Items, discoveredSceneFromStashBox(scene, box))
			if len(page.Items) >= limit {
				break
			}
		}
		sortDiscoveredScenes(page.Items, sortBy)
		return page, nil
	}

	if lastErr != nil {
		return DiscoverScenePage{Items: []DiscoveredScene{}, SearchedQuery: query}, lastErr
	}
	return DiscoverScenePage{Items: []DiscoveredScene{}, SearchedQuery: query}, nil
}

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*taskruntime.Task, error) {
	if s.taskCreator == nil {
		return nil, errors.New("subscription: task runtime is not configured")
	}
	return s.taskCreator.QueueDiscoveredScene(ctx, sceneID, stashBoxEndpoint)
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
		DerivedQuery:     buildReleaseCode(stringValue(scene.Code), stringValue(scene.Title)),
	}
}

func stashBoxStudioName(scene *stashboxgraphql.SceneFragment) string {
	if scene == nil || scene.Studio == nil {
		return ""
	}
	return strings.TrimSpace(scene.Studio.Name)
}

// sortDiscoveredScenes sorts in place. RELEVANCE leaves the slice untouched —
// StashBox's native ordering is the relevance order. Empty values always sort
// to the end regardless of direction so the most informative results stay on top.
func sortDiscoveredScenes(items []DiscoveredScene, sortBy DiscoverSort) {
	switch sortBy {
	case DiscoverSortRelevance:
		return
	case DiscoverSortDateDesc:
		sort.SliceStable(items, func(i, j int) bool {
			return compareDateDesc(items[i].Date, items[j].Date)
		})
	case DiscoverSortDateAsc:
		sort.SliceStable(items, func(i, j int) bool {
			return compareDateDesc(items[j].Date, items[i].Date)
		})
	case DiscoverSortDurationDesc:
		sort.SliceStable(items, func(i, j int) bool {
			return compareDurationDesc(items[i].DurationSeconds, items[j].DurationSeconds)
		})
	case DiscoverSortTitleAsc:
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		})
	}
}

// compareDateDesc returns true when a should come before b. Empty dates sort last.
func compareDateDesc(a, b string) bool {
	if a == b {
		return false
	}
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	return a > b
}

// compareDurationDesc returns true when a should come before b. Nil durations sort last.
func compareDurationDesc(a, b *int) bool {
	if a == b {
		return false
	}
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return *a > *b
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
