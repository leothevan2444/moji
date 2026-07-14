package graphqlapi

import (
	"math"
	"strings"

	"github.com/leothevan2444/moji/internal/performer"
)

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
