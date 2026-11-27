package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSamplePackParses(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	packPath := filepath.Join(filepath.Dir(file), "..", "..", "packs", "sample.json")
	b, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("read sample pack: %v", err)
	}
	var pack Pack
	if err := json.Unmarshal(b, &pack); err != nil {
		t.Fatalf("parse sample pack: %v", err)
	}
	if pack.PackID != "sample" {
		t.Fatalf("pack_id = %q, want sample", pack.PackID)
	}
	if pack.Version != 1 {
		t.Fatalf("version = %d, want 1", pack.Version)
	}
	if len(pack.Prompts) != 2 {
		t.Fatalf("prompts = %d, want 2", len(pack.Prompts))
	}
	for _, p := range pack.Prompts {
		if p.ID == "" || p.Text == "" {
			t.Fatalf("prompt missing id/text: %+v", p)
		}
		if len(p.Entries) < 8 {
			t.Fatalf("prompt %s has %d entries, want >= 8", p.ID, len(p.Entries))
		}
		for _, e := range p.Entries {
			if e.ID == "" || e.Label == "" {
				t.Fatalf("entry missing id/label in %s: %+v", p.ID, e)
			}
			if e.Rarity < 0 || e.Rarity > 100 {
				t.Fatalf("rarity out of range for %s: %d", e.ID, e.Rarity)
			}
		}
	}
}

func TestCorePackLoads(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "packs")
	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// sample (2) + core (84) + broad (4) + krillion (~100+)
	if store.PromptCount() < 200 {
		t.Fatalf("prompts = %d, want sample+core+broad+krillion", store.PromptCount())
	}

	b, err := os.ReadFile(filepath.Join(dir, "core-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pack Pack
	if err := json.Unmarshal(b, &pack); err != nil {
		t.Fatal(err)
	}
	if pack.PackID != "core-v1" {
		t.Fatalf("pack_id = %q", pack.PackID)
	}
	if len(pack.Prompts) != 84 {
		t.Fatalf("prompts = %d, want 84", len(pack.Prompts))
	}

	kb, err := os.ReadFile(filepath.Join(dir, "krillion-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var kpack Pack
	if err := json.Unmarshal(kb, &kpack); err != nil {
		t.Fatal(err)
	}
	if kpack.PackID != "krillion-v1" {
		t.Fatalf("krillion pack_id = %q", kpack.PackID)
	}
	if len(kpack.Prompts) < 100 {
		t.Fatalf("krillion prompts = %d, want >= 100", len(kpack.Prompts))
	}
	attrK := strings.Join(kpack.Attribution, "\n")
	if !strings.Contains(attrK, "Krillion") {
		t.Fatalf("krillion attribution missing Krillion: %v", kpack.Attribution)
	}
	byCat := map[string]int{}
	for _, p := range pack.Prompts {
		byCat[p.Category]++
	}
	for _, cat := range []string{
		"Cat breeds", "Currencies", "Constellations", "US states",
		"Musical instruments", "Video game consoles", "Greek deities",
		"Gemstones", "Cheeses", "Car manufacturers",
	} {
		if byCat[cat] < 3 {
			t.Fatalf("category %q has %d prompts", cat, byCat[cat])
		}
	}
	attr := strings.Join(pack.Attribution, "\n")
	for _, want := range []string{"Wikidata (CC0)", "Wikimedia Pageviews API", "REST Countries", "PokeAPI"} {
		if !strings.Contains(attr, want) {
			t.Fatalf("attribution missing %q: %v", want, pack.Attribution)
		}
	}
	for _, p := range pack.Prompts {
		if len(p.Entries) == 0 {
			t.Fatalf("prompt %s has no entries", p.ID)
		}
		for i := 1; i < len(p.Entries); i++ {
			a, b := p.Entries[i-1], p.Entries[i]
			if a.Rarity > b.Rarity || (a.Rarity == b.Rarity && a.ID > b.ID) {
				t.Fatalf("prompt %s entries not ordered at %s", p.ID, b.ID)
			}
		}
	}
}
