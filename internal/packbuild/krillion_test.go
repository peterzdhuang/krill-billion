package packbuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildKrillionPack(t *testing.T) {
	dir := t.TempDir()
	day := KrillionDay{
		Date:   "2026-11-01",
		Source: "test",
		Prompts: []KrillionPrompt{
			{
				ID:   "p1",
				Text: "Name a fruit",
				Answers: []KrillionAnswer{
					{Answer: "Durian", Aliases: []string{"King of fruits"}, Tier: "krillion", Score: 100},
					{Answer: "Apple", Tier: "plankton", Score: 10},
					{Answer: "Kiwi", Aliases: []string{"Kiwifruit"}, Tier: "schooler", Score: 30},
				},
			},
			{
				ID:   "p2",
				Text: "Name a programming language",
				Answers: []KrillionAnswer{
					{Answer: "Rust", Score: 60},
					{Answer: "Python", Score: 10},
				},
			},
		},
	}
	b, err := json.Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2026-11-01.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	// Second day merges aliases / higher score onto the same fruit prompt.
	day2 := KrillionDay{
		Date: "2026-11-02",
		Prompts: []KrillionPrompt{{
			ID:   "p1",
			Text: "Name a fruit",
			Answers: []KrillionAnswer{
				{Answer: "Durian", Aliases: []string{"Durian"}, Score: 100},
				{Answer: "Rambutan", Score: 85},
			},
		}},
	}
	b2, err := json.Marshal(day2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2026-11-02.json"), b2, 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out.json")
	pack, err := BuildKrillionPack(dir, out, "krillion-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.Prompts) != 2 {
		t.Fatalf("prompts = %d, want 2", len(pack.Prompts))
	}
	var fruit *struct {
		n int
	}
	_ = fruit
	for _, p := range pack.Prompts {
		if p.Category == "" || p.ID == "" || p.Text == "" {
			t.Fatalf("incomplete prompt: %+v", p)
		}
		if len(p.Entries) == 0 {
			t.Fatalf("empty entries for %s", p.ID)
		}
		if p.Text == "A fruit" {
			if len(p.Entries) != 4 {
				t.Fatalf("fruit entries = %d, want 4", len(p.Entries))
			}
			foundAlias := false
			for _, e := range p.Entries {
				if e.Label == "Durian" {
					for _, a := range e.Aliases {
						if a == "King of fruits" || a == "Durian" {
							foundAlias = true
						}
					}
				}
			}
			if !foundAlias {
				t.Fatalf("durian aliases not merged: %+v", p.Entries)
			}
		}
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}
