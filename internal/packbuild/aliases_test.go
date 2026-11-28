package packbuild

import (
	"testing"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

func TestExpandAliases(t *testing.T) {
	got := ExpandAliases("The Beatles", []string{"Beatles (band)"})
	wantAny := []string{"Beatles", "The Beatles (band)", "Beatles"}
	found := map[string]bool{}
	for _, a := range got {
		found[a] = true
	}
	if !found["Beatles"] {
		t.Fatalf("expected Beatles short form, got %v", got)
	}
	_ = wantAny
	got2 := ExpandAliases("Simon & Garfunkel", nil)
	foundAnd := false
	for _, a := range got2 {
		if a == "Simon and Garfunkel" {
			foundAnd = true
		}
	}
	if !foundAnd {
		t.Fatalf("& → and missing: %v", got2)
	}
	got3 := ExpandAliases("São Paulo", nil)
	foundFlat := false
	for _, a := range got3 {
		if a == "Sao Paulo" {
			foundFlat = true
		}
	}
	if !foundFlat {
		t.Fatalf("accent-free missing: %v", got3)
	}
}

func TestFoldCaseVariantEntries(t *testing.T) {
	in := []data.Entry{
		{ID: "1", Label: "Paris", Aliases: []string{"City of Light"}, Rarity: 10},
		{ID: "2", Label: "paris", Aliases: []string{"PARIS"}, Rarity: 5},
	}
	out := FoldCaseVariantEntries(in)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(out), out)
	}
	if out[0].Label != "Paris" {
		t.Fatalf("label=%q want Paris", out[0].Label)
	}
	if out[0].Rarity != 5 {
		t.Fatalf("rarity=%d want 5 (min)", out[0].Rarity)
	}
}

func TestFilterPromptsMinEntries(t *testing.T) {
	prompts := []data.Prompt{
		{ID: "a", Text: "A", Entries: make([]data.Entry, 10000)},
		{ID: "b", Text: "B", Entries: make([]data.Entry, 9999)},
	}
	kept, removed := FilterPromptsMinEntries(prompts, MinEntries)
	if len(kept) != 1 || removed != 1 || kept[0].ID != "a" {
		t.Fatalf("kept=%d removed=%d", len(kept), removed)
	}
}
