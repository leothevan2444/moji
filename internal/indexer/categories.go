package indexer

import (
	"sort"

	"github.com/leothevan2444/moji/internal/indexercategory"
)

func categoriesFromNames(names []string) []int {
	ids := []int{}
	seen := map[int]bool{}
	for _, name := range names {
		if id, ok := indexercategory.ID(name); ok && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
