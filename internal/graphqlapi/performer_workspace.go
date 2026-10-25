package graphqlapi

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/performer"
)

// Apply list options before pagination, while retaining the complete subscription snapshot.
func filterAndSortPerformers(items []performer.Performer, subscribed []*model.SubscribedPerformer, reviews map[string]int, options *model.PerformerListOptions) []performer.Performer {
	if options == nil {
		return items
	}
	states := make(map[string]*model.SubscribedPerformer, len(subscribed))
	for _, state := range subscribed {
		states[state.Performer.ID] = state
	}
	filtered := make([]performer.Performer, 0, len(items))
	for _, item := range items {
		state := states[item.ID]
		keep := true
		if options.Filter != nil {
			switch *options.Filter {
			case model.PerformerListFilterSubscribed:
				keep = item.Subscribed
			case model.PerformerListFilterFavorite:
				keep = item.Favorite
			case model.PerformerListFilterReview:
				keep = reviews[item.ID] > 0
			case model.PerformerListFilterError:
				keep = state != nil && strings.TrimSpace(derefString(state.LastError)) != ""
			}
		}
		if keep {
			filtered = append(filtered, item)
		}
	}
	if options.Sort == nil || *options.Sort == model.PerformerListSortDefault {
		return filtered
	}
	checkedAt := func(id string) time.Time {
		if state := states[id]; state != nil {
			value, _ := time.Parse(time.RFC3339Nano, derefString(state.LastCheckedAt))
			return value
		}
		return time.Time{}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		switch *options.Sort {
		case model.PerformerListSortSceneCount:
			if a.SceneCount != b.SceneCount {
				return a.SceneCount > b.SceneCount
			}
		case model.PerformerListSortLastChecked:
			left, right := checkedAt(a.ID), checkedAt(b.ID)
			if !left.Equal(right) {
				return left.After(right)
			}
		}
		left, right := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if left != right {
			return left < right
		}
		return a.ID < b.ID
	})
	return filtered
}

func buildStashPerformerPage(items []performer.Performer, search string, page *int, pageSize *int) StashPerformerPage {
	needle := strings.ToLower(strings.TrimSpace(search))
	filtered := make([]performer.Performer, 0, len(items))
	for _, item := range items {
		matched := needle == "" || strings.Contains(strings.ToLower(item.Name), needle)
		if !matched {
			for _, alias := range item.AliasList {
				if strings.Contains(strings.ToLower(alias), needle) {
					matched = true
					break
				}
			}
		}
		if matched {
			filtered = append(filtered, item)
		}
	}
	currentPage := normalizePage(page)
	currentPageSize := normalizePageSize(pageSize)
	totalCount := len(filtered)
	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(currentPageSize)))
	}
	if totalPages > 0 && currentPage > totalPages {
		currentPage = totalPages
	}
	start, end := 0, 0
	if totalCount > 0 {
		start = min((currentPage-1)*currentPageSize, totalCount)
		end = min(start+currentPageSize, totalCount)
	}
	return StashPerformerPage{Items: filtered[start:end], Page: currentPage, PageSize: currentPageSize, TotalCount: totalCount, TotalPages: totalPages, HasPrevPage: currentPage > 1 && totalPages > 0, HasNextPage: currentPage < totalPages}
}
