package tracker

import "github.com/leothevan2444/moji/pkg/jackett"

type Tracker interface {
	Search(query string, options ...SearchOption) ([]jackett.SearchResult, error)
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
