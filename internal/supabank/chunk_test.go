package supabank

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

func TestPayloadEmitsEmptyArrays(t *testing.T) {
	p, err := payloadFromPack(&data.Pack{PackID: "x", Prompts: nil})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{"categories", "prompts", "entities", "aliases", "prompt_entities"} {
		if !json.Valid(b) {
			t.Fatal("invalid json")
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		v := string(raw[key])
		if v != "[]" {
			t.Fatalf("%s = %s, want [] (got from %s)", key, v, s)
		}
	}
}

func TestPayloadFoldsCaseVariantLabels(t *testing.T) {
	p, err := payloadFromPack(&data.Pack{
		PackID: "x",
		Prompts: []data.Prompt{
			{
				ID: "p1", Category: "C", Text: "T",
				Entries: []data.Entry{{ID: "e1", Label: "Paris", Rarity: 1}},
			},
			{
				ID: "p2", Category: "C", Text: "U",
				Entries: []data.Entry{{ID: "e1", Label: "paris", Aliases: []string{"City of Light"}, Rarity: 2}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Entities) != 1 || p.Entities[0].Label != "Paris" {
		t.Fatalf("entities = %+v", p.Entities)
	}
	found := false
	for _, a := range p.Aliases {
		if a.Alias == "City of Light" {
			found = true
		}
		if a.Alias == "paris" {
			t.Fatal("case-variant alias should be dropped")
		}
	}
	if !found {
		t.Fatalf("aliases = %+v", p.Aliases)
	}
}

func TestChunkPacks(t *testing.T) {
	entries := make([]data.Entry, 200)
	for i := range entries {
		entries[i] = data.Entry{
			ID:     fmt.Sprintf("id-%04d", i),
			Label:  fmt.Sprintf("Item Number %04d", i),
			Rarity: i % 100,
		}
	}
	pack := &data.Pack{
		PackID: "big",
		Prompts: []data.Prompt{
			{ID: "p1", Category: "C", Text: "One", Entries: entries},
			{ID: "p2", Category: "C", Text: "Two", Entries: entries},
			{ID: "p3", Category: "C", Text: "Three", Entries: entries[:50]},
		},
	}
	chunks, err := chunkPacks(pack, 8*1024) // tiny budget forces splits
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		total += len(c.Prompts)
	}
	if total != 3 {
		t.Fatalf("prompt sum %d want 3", total)
	}
}
