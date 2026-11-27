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

// Scored is one entity with the two notability signals blended into rarity.
type Scored struct {
	ID        string
	Views     int64
	Sitelinks int64
}

// AssignBlendedRarity averages the pageview percentile and the sitelink
// percentile. Each percentile uses AssignRarity (0 = highest signal, 100 =
// lowest; ties break by entity id). The average is rounded with math.Round
// (half away from zero). When every Views value is 0, pageviews are missing
// and the sitelink percentile is used on its own.
func AssignBlendedRarity(entries []Scored) map[string]int {
	out := make(map[string]int, len(entries))
	n := len(entries)
	if n == 0 {
		return out
	}
	if n == 1 {
		out[entries[0].ID] = 0
		return out
	}

	views := make([]Viewed, n)
	links := make([]Viewed, n)
	viewsMissing := true
	for i, e := range entries {
		views[i] = Viewed{ID: e.ID, Views: e.Views}
		links[i] = Viewed{ID: e.ID, Views: e.Sitelinks}
		if e.Views != 0 {
			viewsMissing = false
		}
	}
	linkR := AssignRarity(links)
	if viewsMissing {
		return linkR
	}
	viewR := AssignRarity(views)
	for _, e := range entries {
		avg := (float64(viewR[e.ID]) + float64(linkR[e.ID])) / 2
		out[e.ID] = int(math.Round(avg))
	}
	return out
}
