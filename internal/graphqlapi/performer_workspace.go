package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/ordering"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/performer"
)

// Apply list options before pagination, while retaining the complete subscription snapshot.
type performerCheckTimes interface {
	PerformerCheckTimes(context.Context, []string) (map[string]time.Time, error)
}

type performerReviewCounts interface {
	PendingReviewPerformerCounts(context.Context) (map[string]int, error)
}

func filterAndSortPerformers(items []performer.Performer, subscribed []*model.SubscribedPerformer, reviews map[string]int, options *model.PerformerListOptions, checks map[string]time.Time) []performer.Performer {
	if options == nil {
		options = &model.PerformerListOptions{}
	}
	states := make(map[string]*model.SubscribedPerformer, len(subscribed))
	for _, state := range subscribed {
		states[state.Performer.ID] = state
	}
	filtered := make([]performer.Performer, 0, len(items))
	for _, item := range items {
		state := states[item.ID]
		filter := model.PerformerListFilterAll
		if options.Filter != nil {
			filter = *options.Filter
		}
		keep := performerMatchesFilter(item, state, reviews, filter)
		if keep {
			filtered = append(filtered, item)
		}
	}
	field := model.PerformerListSortName
	if options.Sort != nil {
		field = *options.Sort
	}
	descending := options.Direction != nil && *options.Direction == model.SortDirectionDesc
	checkedAt := func(id string) time.Time {
		if checks != nil {
			return checks[id]
		}
		if state := states[id]; state != nil {
			value, _ := time.Parse(time.RFC3339Nano, derefString(state.LastCheckedAt))
			return value
		}
		return time.Time{}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		comparison := 0
		switch field {
		case model.PerformerListSortSceneCount:
			if a.SceneCount < b.SceneCount {
				comparison = -1
			} else if a.SceneCount > b.SceneCount {
				comparison = 1
			}
		case model.PerformerListSortCreated:
			if a.CreatedAt.IsZero() != b.CreatedAt.IsZero() {
				return !a.CreatedAt.IsZero()
			}
			comparison = a.CreatedAt.Compare(b.CreatedAt)
		case model.PerformerListSortLastChecked:
			left, right := checkedAt(a.ID), checkedAt(b.ID)
			if left.IsZero() != right.IsZero() {
				return !left.IsZero()
			}
			comparison = left.Compare(right)
		default:
			comparison = ordering.Compare(a.Name, b.Name)
		}
		if comparison != 0 {
			if descending {
				return comparison > 0
			}
			return comparison < 0
		}
		if left, right := strings.ToLower(a.Name), strings.ToLower(b.Name); left != right {
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
		matched := performerMatchesSearch(item, needle)
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

func performerMatchesSearch(item performer.Performer, needle string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" || strings.Contains(strings.ToLower(item.Name), needle) {
		return true
	}
	for _, alias := range item.AliasList {
		if strings.Contains(strings.ToLower(alias), needle) {
			return true
		}
	}
	return false
}

func performerMatchesFilter(item performer.Performer, state *model.SubscribedPerformer, reviews map[string]int, filter model.PerformerListFilter) bool {
	switch filter {
	case model.PerformerListFilterSubscribed:
		return item.Subscribed
	case model.PerformerListFilterFavorite:
		return item.Favorite
	case model.PerformerListFilterReview:
		return reviews[item.ID] > 0
	case model.PerformerListFilterError:
		return item.Subscribed && state != nil && strings.TrimSpace(derefString(state.LastError)) != ""
	default:
		return true
	}
}

// Count actors once per matching category, before applying the selected category or page.
func performerFilterCounts(items []performer.Performer, subscribed []*model.SubscribedPerformer, reviews map[string]int, search string, reviewsKnown bool) *model.PerformerFilterCounts {
	counts := &model.PerformerFilterCounts{}
	review := 0
	if reviewsKnown {
		counts.Review = &review
	}
	states := make(map[string]*model.SubscribedPerformer, len(subscribed))
	for _, state := range subscribed {
		states[state.Performer.ID] = state
	}
	for _, item := range items {
		if !performerMatchesSearch(item, search) {
			continue
		}
		counts.Total++
		if performerMatchesFilter(item, states[item.ID], reviews, model.PerformerListFilterSubscribed) {
			counts.Subscribed++
		}
		if performerMatchesFilter(item, states[item.ID], reviews, model.PerformerListFilterFavorite) {
			counts.Favorite++
		}
		if performerMatchesFilter(item, states[item.ID], reviews, model.PerformerListFilterError) {
			counts.Error++
		}
		if counts.Review != nil && performerMatchesFilter(item, states[item.ID], reviews, model.PerformerListFilterReview) {
			*counts.Review++
		}
	}
	return counts
}
