package packbuild

import "testing"

func TestAssignRarity_SingleEntry(t *testing.T) {
	got := AssignRarity([]Viewed{{ID: "Q1", Views: 999}})
	if got["Q1"] != 0 {
		t.Fatalf("n=1 rarity = %d, want 0", got["Q1"])
	}
}

func TestAssignRarity_EndsAndMiddle(t *testing.T) {
	entries := []Viewed{
		{ID: "high", Views: 1000},
		{ID: "mid", Views: 500},
		{ID: "low", Views: 10},
	}
	got := AssignRarity(entries)
	if got["high"] != 0 {
		t.Fatalf("most viewed rarity = %d, want 0", got["high"])
	}
	if got["low"] != 100 {
		t.Fatalf("least viewed rarity = %d, want 100", got["low"])
	}
	if got["mid"] != 50 {
		t.Fatalf("middle rarity = %d, want 50", got["mid"])
	}
}

func TestAssignRarity_TiesDeterministic(t *testing.T) {
	// Equal views: lower ID sorts first among ties → lower rarity rank index.
	a := AssignRarity([]Viewed{
		{ID: "b", Views: 100},
		{ID: "a", Views: 100},
		{ID: "c", Views: 50},
	})
	b := AssignRarity([]Viewed{
		{ID: "c", Views: 50},
		{ID: "a", Views: 100},
		{ID: "b", Views: 100},
	})
	if a["a"] != b["a"] || a["b"] != b["b"] || a["c"] != b["c"] {
		t.Fatalf("tie break not deterministic: first=%v second=%v", a, b)
	}
	if a["a"] >= a["b"] {
		t.Fatalf("tie break by id: want a before b, got a=%d b=%d", a["a"], a["b"])
	}
	if a["c"] != 100 {
		t.Fatalf("least viewed rarity = %d, want 100", a["c"])
	}
}

func TestAssignBlendedRarity(t *testing.T) {
	// Four entries. Pageview ranks 0,33,67,100 and sitelink ranks put "a" at
	// 33, so the average is 16.5 and math.Round yields 17 (half away from zero).
	entries := []Scored{
		{ID: "a", Views: 40, Sitelinks: 30},
		{ID: "b", Views: 30, Sitelinks: 40},
		{ID: "c", Views: 20, Sitelinks: 20},
		{ID: "d", Views: 10, Sitelinks: 10},
	}
	got := AssignBlendedRarity(entries)
	if got["a"] != 17 || got["b"] != 17 {
		t.Fatalf("half-up blend: a=%d b=%d, want 17 and 17", got["a"], got["b"])
	}
	if got["c"] != 67 || got["d"] != 100 {
		t.Fatalf("blend ends: c=%d d=%d, want 67 and 100", got["c"], got["d"])
	}

	reversed := AssignBlendedRarity([]Scored{entries[3], entries[1], entries[0], entries[2]})
	for id, rarity := range got {
		if reversed[id] != rarity {
			t.Fatalf("blend not deterministic for %s: %d vs %d", id, rarity, reversed[id])
		}
	}

	// Missing pageviews (every view is 0) uses the sitelink percentile alone.
	missing := []Scored{
		{ID: "a", Views: 0, Sitelinks: 30},
		{ID: "b", Views: 0, Sitelinks: 40},
		{ID: "c", Views: 0, Sitelinks: 20},
		{ID: "d", Views: 0, Sitelinks: 10},
	}
	gotMissing := AssignBlendedRarity(missing)
	wantLinks := AssignRarity([]Viewed{
		{ID: "a", Views: 30},
		{ID: "b", Views: 40},
		{ID: "c", Views: 20},
		{ID: "d", Views: 10},
	})
	for id, rarity := range wantLinks {
		if gotMissing[id] != rarity {
			t.Fatalf("missing pageviews %s = %d, want sitelink rarity %d", id, gotMissing[id], rarity)
		}
	}
	if gotMissing["b"] != 0 || gotMissing["d"] != 100 {
		t.Fatalf("missing pageviews ends: b=%d d=%d, want 0 and 100", gotMissing["b"], gotMissing["d"])
	}

	one := AssignBlendedRarity([]Scored{{ID: "only", Views: 0, Sitelinks: 3}})
	if one["only"] != 0 {
		t.Fatalf("n=1 blend = %d, want 0", one["only"])
	}
}
