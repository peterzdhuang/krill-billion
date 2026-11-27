package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
