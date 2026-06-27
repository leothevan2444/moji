package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

// mapDiscoverSortBy translates the GraphQL enum into the subscription-domain
// enum so resolver code doesn't depend on gqlgen-generated types directly.
func mapDiscoverSortBy(value *model.DiscoverSortBy) subscription.DiscoverSort {
	if value == nil {
		return subscription.DiscoverSortRelevance
	}
	switch *value {
	case model.DiscoverSortByRelevance:
		return subscription.DiscoverSortRelevance
	case model.DiscoverSortByDateDesc:
		return subscription.DiscoverSortDateDesc
	case model.DiscoverSortByDateAsc:
		return subscription.DiscoverSortDateAsc
	case model.DiscoverSortByDurationDesc:
		return subscription.DiscoverSortDurationDesc
	case model.DiscoverSortByTitleAsc:
		return subscription.DiscoverSortTitleAsc
	default:
		return subscription.DiscoverSortRelevance
	}
}
