package tracker

import (
	"context"
	"github.com/leothevan2444/moji/pkg/jackett"
)

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
	Context    context.Context
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

// SequentialSearcher stops only after the caller accepts a verified candidate.
// Search options restrict eligible sites; configured priority orders that set.
type SequentialSearcher interface {
	SearchSequential(context.Context, string, func([]jackett.SearchResult) (bool, error), ...SearchOption) error
}

func WithContext(ctx context.Context) SearchOption { return func(o *SearchOptions) { o.Context = ctx } }
