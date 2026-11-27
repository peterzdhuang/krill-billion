package packbuild

import (
	"math"
	"sort"
)

// Viewed is an entry with a pageview total used for rarity ranking.
type Viewed struct {
	ID    string
	Views int64
}

// AssignRarity ranks entries by views descending and assigns rarity 0–100.
// Most viewed → 0, least viewed → 100. For n==1, rarity is 0.
// Ties (equal views) are broken by ID ascending so ordering is deterministic.
func AssignRarity(entries []Viewed) map[string]int {
	out := make(map[string]int, len(entries))
	n := len(entries)
	if n == 0 {
		return out
	}
	if n == 1 {
		out[entries[0].ID] = 0
		return out
	}

	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		a, b := entries[idx[i]], entries[idx[j]]
		if a.Views != b.Views {
			return a.Views > b.Views
		}
		return a.ID < b.ID
	})

	denom := float64(n - 1)
	for rank, i := range idx {
		rarity := int(math.Round(100 * float64(rank) / denom))
		out[entries[i].ID] = rarity
	}
	return out
}
