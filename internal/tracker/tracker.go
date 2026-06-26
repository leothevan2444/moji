package tracker

import "github.com/leothevan2444/moji/pkg/jackett"

type Tracker interface {
	Search(query string, options ...SearchOption) ([]jackett.SearchResult, error)
}

// IndexerLister is optionally implemented by Tracker implementations that can
// enumerate the configured Jackett indexers. A nil return is a valid "not
// supported" answer — the GraphQL resolver surfaces that as an empty list.
type IndexerLister interface {
	ListIndexers() ([]jackett.Indexer, error)
}

type SearchOption func(*SearchOptions)

type SearchOptions struct {
	Categories []int
	Trackers   []string
	Limit      int
}

func WithCategories(categories []int) SearchOption {
	return func(o *SearchOptions) {
		o.Categories = categories
	}
}

func WithTrackers(trackers []string) SearchOption {
	return func(o *SearchOptions) {
		o.Trackers = trackers
	}
}

func WithLimit(limit int) SearchOption {
	return func(o *SearchOptions) {
		o.Limit = limit
	}
}
