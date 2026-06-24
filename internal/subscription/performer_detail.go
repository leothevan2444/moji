package subscription

import (
	"context"
	"fmt"
	"sort"
	"strings"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func (s *Service) GetPerformerDetail(ctx context.Context, performerID string) (PerformerDetail, error) {
	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		return PerformerDetail{}, err
	}
	if performer == nil {
		return PerformerDetail{}, fmt.Errorf("subscription: performer %q not found", performerID)
	}

	item := performerFromStash(performer, s.customFieldKey)
	detail := PerformerDetail{
		Performer:      item,
		Disambiguation: stringValue(performer.Disambiguation),
		Birthdate:      stringValue(performer.Birthdate),
		Ethnicity:      stringValue(performer.Ethnicity),
		Country:        stringValue(performer.Country),
		EyeColor:       stringValue(performer.EyeColor),
		HeightCm:       performer.HeightCm,
		Rating100:      performer.Rating100,
		URLs:           append([]string(nil), performerURLs(performer)...),
	}

	target, err := s.resolveStashboxPerformer(ctx, performer)
	if err != nil && !strings.Contains(err.Error(), errNoMatchingStashBoxMapping.Error()) {
		return PerformerDetail{}, err
	}
	if target != nil {
		detail.MatchedStashBox = &MatchedStashBox{
			Name:          target.Name,
			Endpoint:      target.Endpoint,
			PerformerID:   target.Performer.ID,
			PerformerName: target.Performer.Name,
		}
	}
	detail.TotalSceneCount = performer.SceneCount
	detail.StashSceneCount = performer.SceneCount
	detail.StashBoxSceneCount = 0
	detail.DedupedSceneCount = performer.SceneCount
	return detail, nil
}

func (s *Service) ListPerformerScenes(ctx context.Context, performerID string, query PerformerSceneQuery) (PerformerScenePage, error) {
	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		return PerformerScenePage{}, err
	}
	if performer == nil {
		return PerformerScenePage{}, fmt.Errorf("subscription: performer %q not found", performerID)
	}
	return s.buildPerformerScenePage(ctx, performer, query)
}

func (s *Service) buildPerformerScenePage(ctx context.Context, performer *stashgraphql.PerformerFragment, query PerformerSceneQuery) (PerformerScenePage, error) {
	query = normalizePerformerSceneQuery(query)

	stashScenes, err := s.fetchStashScenes(ctx, performer.ID)
	if err != nil {
		return PerformerScenePage{}, err
	}

	itemsByID := make(map[string]*PerformerScene, len(stashScenes))
	ordered := make([]*PerformerScene, 0, len(stashScenes))
	stashByEndpointAndID := make(map[string]*stashgraphql.SceneFragment, len(stashScenes))
	for _, scene := range stashScenes {
		item := stashSceneToPerformerScene(scene)
		itemsByID[item.SourceSceneID] = &item
		ordered = append(ordered, &item)
		for _, stashID := range scene.StashIds {
			if stashID == nil || strings.TrimSpace(stashID.StashID) == "" {
				continue
			}
			key := sceneLookupKey(stashID.Endpoint, stashID.StashID)
			if key == "" {
				continue
			}
			if _, exists := stashByEndpointAndID[key]; !exists {
				stashByEndpointAndID[key] = scene
			}
		}
	}

	stashBoxCount := 0
	target, targetErr := s.resolveStashboxPerformer(ctx, performer)
	if targetErr == nil && target != nil {
		stashboxScenes, err := s.fetchStashBoxScenes(ctx, target)
		if err != nil {
			return PerformerScenePage{}, err
		}
		stashBoxCount = len(stashboxScenes)
		for _, scene := range stashboxScenes {
			if scene == nil {
				continue
			}
			matched := stashByEndpointAndID[sceneLookupKey(target.Endpoint, scene.ID)]
			if matched != nil {
				if current, ok := itemsByID[matched.ID]; ok {
					mergeStashBoxIntoStashScene(current, scene, target.Endpoint)
					continue
				}
				item := stashSceneToPerformerScene(matched)
				mergeStashBoxIntoStashScene(&item, scene, target.Endpoint)
				itemsByID[item.SourceSceneID] = &item
				ordered = append(ordered, &item)
				continue
			}

			item := stashBoxSceneToPerformerScene(scene, target.Endpoint)
			ordered = append(ordered, &item)
		}
	} else if targetErr != nil && !strings.Contains(targetErr.Error(), errNoMatchingStashBoxMapping.Error()) {
		return PerformerScenePage{}, targetErr
	}

	flat := make([]PerformerScene, 0, len(ordered))
	for _, item := range ordered {
		if item == nil {
			continue
		}
		flat = append(flat, *item)
	}

	filtered := make([]PerformerScene, 0, len(flat))
	for _, item := range flat {
		if !matchesPerformerSceneSearch(item, query.Search) {
			continue
		}
		if !matchesPerformerSceneSource(item, query.Source) {
			continue
		}
		if !matchesPerformerSceneLibrary(item, query.InLibrary) {
			continue
		}
		filtered = append(filtered, item)
	}

	sort.Slice(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]
		if left.Date != right.Date {
			if left.Date == "" {
				return false
			}
			if right.Date == "" {
				return true
			}
			return left.Date > right.Date
		}
		if left.InLibrary != right.InLibrary {
			return left.InLibrary
		}
		leftKey := strings.ToLower(strings.TrimSpace(left.Code + " " + left.Title))
		rightKey := strings.ToLower(strings.TrimSpace(right.Code + " " + right.Title))
		return leftKey < rightKey
	})

	totalCount := len(filtered)
	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + query.PageSize - 1) / query.PageSize
	}
	if totalPages > 0 && query.Page > totalPages {
		query.Page = totalPages
	}

	start := 0
	end := 0
	if totalCount > 0 {
		start = (query.Page - 1) * query.PageSize
		if start > totalCount {
			start = totalCount
		}
		end = start + query.PageSize
		if end > totalCount {
			end = totalCount
		}
	}

	return PerformerScenePage{
		Items:           filtered[start:end],
		Page:            query.Page,
		PageSize:        query.PageSize,
		TotalCount:      totalCount,
		TotalPages:      totalPages,
		HasPrevPage:     query.Page > 1 && totalPages > 0,
		HasNextPage:     query.Page < totalPages,
		StashSceneCount: len(stashScenes),
		StashBoxCount:   stashBoxCount,
		DedupedCount:    len(flat),
	}, nil
}

func (s *Service) fetchStashScenes(ctx context.Context, performerID string) ([]*stashgraphql.SceneFragment, error) {
	return s.stash.FindScenes(ctx, &stashgraphql.SceneFilterType{
		Performers: &stashgraphql.MultiCriterionInput{
			Value:    []string{performerID},
			Modifier: stashgraphql.CriterionModifierIncludes,
		},
	}, &stashgraphql.FindFilterType{
		Page:      intPointer(1),
		PerPage:   intPointer(-1),
		Sort:      stringPointer("date"),
		Direction: sortDirectionPointer(stashgraphql.SortDirectionEnumDesc),
	})
}

func (s *Service) fetchStashBoxScenes(ctx context.Context, target *resolvedStashbox) ([]*stashboxgraphql.SceneFragment, error) {
	if target == nil {
		return nil, nil
	}
	return target.Client.QueryScenes(ctx, stashboxgraphql.SceneQueryInput{
		Performers: &stashboxgraphql.MultiIDCriterionInput{
			Value:    []string{target.Performer.ID},
			Modifier: stashboxgraphql.CriterionModifierIncludes,
		},
		Page:       1,
		PerPage:    200,
		Direction:  stashboxgraphql.SortDirectionEnumDesc,
		Sort:       stashboxgraphql.SceneSortEnumDate,
	})
}

func normalizePerformerSceneQuery(query PerformerSceneQuery) PerformerSceneQuery {
	query.Search = strings.TrimSpace(query.Search)
	if query.Source == "" {
		query.Source = SceneSourceFilterAll
	}
	if query.InLibrary == "" {
		query.InLibrary = LibraryFilterAll
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 24
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	return query
}

func matchesPerformerSceneSearch(item PerformerScene, search string) bool {
	if search == "" {
		return true
	}
	needle := normalize(search)
	return strings.Contains(normalize(item.Title), needle) ||
		strings.Contains(normalize(item.Code), needle) ||
		strings.Contains(normalize(item.StudioName), needle)
}

func matchesPerformerSceneSource(item PerformerScene, filter SceneSourceFilter) bool {
	switch filter {
	case SceneSourceFilterStash:
		return item.HasStashSource
	case SceneSourceFilterStashBox:
		return item.HasStashBoxSource
	default:
		return true
	}
}

func matchesPerformerSceneLibrary(item PerformerScene, filter LibraryFilter) bool {
	switch filter {
	case LibraryFilterInLibrary:
		return item.InLibrary
	case LibraryFilterNotInLibrary:
		return !item.InLibrary
	default:
		return true
	}
}

func stashSceneToPerformerScene(scene *stashgraphql.SceneFragment) PerformerScene {
	if scene == nil {
		return PerformerScene{}
	}
	item := PerformerScene{
		Key:               "stash:" + scene.ID,
		PrimarySource:     SceneSourceStash,
		SourceSceneID:     scene.ID,
		Title:             stringValue(scene.Title),
		Code:              stringValue(scene.Code),
		Date:              stringValue(scene.Date),
		ImageURL:          stashSceneImageURL(scene),
		URL:               stashSceneURL(scene),
		InLibrary:         true,
		HasStashSource:    true,
		HasStashBoxSource: false,
		SourceLabels:      []string{"Stash"},
		StashIDs:          stashSceneIDs(scene.StashIds),
	}
	if scene.Studio != nil {
		item.StudioName = scene.Studio.Name
	}
	return item
}

func stashBoxSceneToPerformerScene(scene *stashboxgraphql.SceneFragment, endpoint string) PerformerScene {
	item := PerformerScene{
		Key:               "stashbox:" + endpointKey(endpoint) + ":" + scene.ID,
		PrimarySource:     SceneSourceStashBox,
		SourceSceneID:     scene.ID,
		Title:             stringValue(scene.Title),
		Code:              stringValue(scene.Code),
		Date:              stringValue(scene.Date),
		ImageURL:          stashBoxSceneImageURL(scene),
		URL:               stashBoxSceneURL(scene),
		InLibrary:         false,
		HasStashSource:    false,
		HasStashBoxSource: true,
		StashBoxSceneID:   scene.ID,
		StashBoxEndpoint:  endpoint,
		SourceLabels:      []string{"StashBox"},
	}
	if scene.Studio != nil {
		item.StudioName = scene.Studio.Name
	}
	return item
}

func mergeStashBoxIntoStashScene(item *PerformerScene, scene *stashboxgraphql.SceneFragment, endpoint string) {
	if item == nil || scene == nil {
		return
	}
	item.InLibrary = true
	item.MatchedStashSceneID = item.SourceSceneID
	item.HasStashSource = true
	item.HasStashBoxSource = true
	item.StashBoxSceneID = scene.ID
	item.StashBoxEndpoint = endpoint
	item.SourceLabels = []string{"Stash", "StashBox"}
	if item.ImageURL == "" {
		item.ImageURL = stashBoxSceneImageURL(scene)
	}
	if item.URL == "" {
		item.URL = stashBoxSceneURL(scene)
	}
	if item.Title == "" {
		item.Title = stringValue(scene.Title)
	}
	if item.Code == "" {
		item.Code = stringValue(scene.Code)
	}
	if item.Date == "" {
		item.Date = stringValue(scene.Date)
	}
	if item.StudioName == "" && scene.Studio != nil {
		item.StudioName = scene.Studio.Name
	}
}

func stashSceneIDs(items []*stashgraphql.StashIDFragment) []StashSceneID {
	out := make([]StashSceneID, 0, len(items))
	for _, item := range items {
		if item == nil || strings.TrimSpace(item.StashID) == "" {
			continue
		}
		out = append(out, StashSceneID{
			Endpoint: item.Endpoint,
			StashID:  item.StashID,
		})
	}
	return out
}

func performerURLs(performer *stashgraphql.PerformerFragment) []string {
	if performer == nil {
		return nil
	}
	return append([]string(nil), performer.Urls...)
}

func stashSceneImageURL(scene *stashgraphql.SceneFragment) string {
	if scene == nil || scene.Paths.Screenshot == nil {
		return ""
	}
	return *scene.Paths.Screenshot
}

func stashSceneURL(scene *stashgraphql.SceneFragment) string {
	if scene == nil || len(scene.Urls) == 0 {
		return ""
	}
	return scene.Urls[0]
}

func stashBoxSceneImageURL(scene *stashboxgraphql.SceneFragment) string {
	if scene == nil || len(scene.Images) == 0 || scene.Images[0] == nil {
		return ""
	}
	return scene.Images[0].URL
}

func stashBoxSceneURL(scene *stashboxgraphql.SceneFragment) string {
	if scene == nil || len(scene.Urls) == 0 || scene.Urls[0] == nil {
		return ""
	}
	return scene.Urls[0].URL
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func sortDirectionPointer(value stashgraphql.SortDirectionEnum) *stashgraphql.SortDirectionEnum {
	return &value
}

func sceneLookupKey(endpoint, stashID string) string {
	endpoint = normalizeStashBoxEndpoint(endpoint)
	stashID = strings.TrimSpace(stashID)
	if endpoint == "" || stashID == "" {
		return ""
	}
	return endpoint + "::" + stashID
}
