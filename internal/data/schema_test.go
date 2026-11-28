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

func TestSampleExcludedFromLoad(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "packs")
	// Ensure sample.json alone is ignored when no other packs exist by loading
	// via NewFromPacks after reading sample — Load skips sample.json by name.
	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := store.Prompt("sample-country"); ok {
		t.Fatal("sample-country should not load from packs/ via Load")
	}
	_ = store
}

func TestNewFromPacksIndexes(t *testing.T) {
	store, err := NewFromPacks(Pack{
		PackID: "t",
		Prompts: []Prompt{{
			ID: "p", Text: "A thing", Category: "T",
			Entries: []Entry{
				{ID: "1", Label: "Alpha", Aliases: []string{"A"}, Rarity: 1},
				{ID: "2", Label: "Beta", Rarity: 50},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.PromptCount() != 1 || store.EntryCount() != 2 {
		t.Fatalf("counts prompts=%d entries=%d", store.PromptCount(), store.EntryCount())
	}
	id, ok := store.Match("p", "a")
	if !ok || id != "1" {
		t.Fatalf("Match alias = (%q,%v)", id, ok)
	}
}

func TestCoreDefsStillPresent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// Legacy narrow packs may be gitignored; defs must remain rebuildable.
	defsDir := filepath.Join(filepath.Dir(file), "..", "..", "packs", "defs")
	entries, err := os.ReadDir(defsDir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(e.Name()), ".yml") {
			n++
		}
	}
	if n < 10 {
		t.Fatalf("defs yaml files = %d, want many", n)
	}
}

func TestShippedPackMinEntries(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "packs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if e.IsDir() || lower == "sample.json" {
			continue
		}
		if !strings.HasSuffix(lower, ".json") && !strings.HasSuffix(lower, ".json.gz") {
			continue
		}
		pack, err := loadPackFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if pack.PackID == "sample" || pack.PackID == "broad-v1" || pack.PackID == "core-v1" {
			// Legacy/narrow packs: skip min-entries gate (may be absent or pruned).
			continue
		}
		for _, p := range pack.Prompts {
			if len(p.Entries) < 10000 {
				t.Fatalf("%s prompt %s has %d entries, want >= 10000", name, p.ID, len(p.Entries))
			}
			checked++
		}
		attr := strings.Join(pack.Attribution, "\n")
		if pack.PackID == "krillion-v1" && !strings.Contains(attr, "Krillion") {
			t.Fatalf("krillion attribution missing: %v", pack.Attribution)
		}
		if pack.PackID == "deep-v1" {
			for _, want := range []string{"Wikidata", "GeoNames", "IMDb", "Open Food Facts"} {
				if !strings.Contains(attr, want) {
					t.Fatalf("deep attribution missing %q: %v", want, pack.Attribution)
				}
			}
		}
	}
	if checked == 0 {
		t.Skip("no deep/krillion shipped packs present yet (rebuild with packbuild)")
	}
}
