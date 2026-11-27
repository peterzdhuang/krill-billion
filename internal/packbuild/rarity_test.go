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
