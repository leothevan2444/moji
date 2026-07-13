package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/discovery"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

// mapDiscoverSortBy translates the GraphQL enum into the discovery-domain
// enum so resolver code doesn't depend on gqlgen-generated types directly.
func mapDiscoverSortBy(value *model.DiscoverSortBy) discovery.Sort {
	if value == nil {
		return discovery.SortRelevance
	}
	switch *value {
	case model.DiscoverSortByRelevance:
		return discovery.SortRelevance
	case model.DiscoverSortByDateDesc:
		return discovery.SortDateDesc
	case model.DiscoverSortByDateAsc:
		return discovery.SortDateAsc
	case model.DiscoverSortByDurationDesc:
		return discovery.SortDurationDesc
	case model.DiscoverSortByTitleAsc:
		return discovery.SortTitleAsc
	default:
		return discovery.SortRelevance
	}
}
