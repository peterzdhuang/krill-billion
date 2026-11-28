package supabank

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

type bankPayload struct {
	Categories     []categoryRow `json:"categories"`
	Prompts        []promptRow   `json:"prompts"`
	Entities       []entityRow   `json:"entities"`
	Aliases        []aliasRow    `json:"aliases"`
	PromptEntities []linkRow     `json:"prompt_entities"`
}

type categoryRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type promptRow struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	Text       string `json:"text"`
}

type entityRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type aliasRow struct {
	EntityID string `json:"entity_id"`
	Alias    string `json:"alias"`
}

type linkRow struct {
	PromptID string `json:"prompt_id"`
	EntityID string `json:"entity_id"`
	Rarity   int    `json:"rarity"`
}

func payloadFromPack(pack *data.Pack) (bankPayload, error) {
	if pack == nil {
		return bankPayload{}, fmt.Errorf("nil pack")
	}
	// Always emit [] not null — jsonb_array_elements rejects JSON null.
	out := bankPayload{
		Categories:     []categoryRow{},
		Prompts:        []promptRow{},
		Entities:       []entityRow{},
		Aliases:        []aliasRow{},
		PromptEntities: []linkRow{},
	}
	catID := make(map[string]string)
	usedID := make(map[string]string)
	labels := make(map[string]string) // entity id → canonical label
	aliases := make(map[string]map[string]struct{})
	seenPrompt := make(map[string]struct{})
	seenLink := make(map[string]struct{})

	for _, p := range pack.Prompts {
		if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Text) == "" {
			return bankPayload{}, fmt.Errorf("prompt %q is missing an id or text", p.ID)
		}
		if _, ok := seenPrompt[p.ID]; ok {
			return bankPayload{}, fmt.Errorf("duplicate prompt id %s", p.ID)
		}
		seenPrompt[p.ID] = struct{}{}

		name := strings.TrimSpace(p.Category)
		if name == "" {
			name = "Uncategorized"
		}
		cid, ok := catID[name]
		if !ok {
			cid = categorySlug(name)
			if prev, exists := usedID[cid]; exists && prev != name {
				return bankPayload{}, fmt.Errorf("category id %q would be used by both %q and %q", cid, prev, name)
			}
			catID[name] = cid
			usedID[cid] = name
			out.Categories = append(out.Categories, categoryRow{ID: cid, Name: name})
		}
		out.Prompts = append(out.Prompts, promptRow{ID: p.ID, CategoryID: cid, Text: p.Text})

		for _, e := range p.Entries {
			if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Label) == "" {
				return bankPayload{}, fmt.Errorf("prompt %s has an entry with an empty id or label", p.ID)
			}
			if e.Rarity < 0 || e.Rarity > 100 {
				return bankPayload{}, fmt.Errorf("prompt %s entity %s rarity %d out of range", p.ID, e.ID, e.Rarity)
			}

			set := aliases[e.ID]
			if set == nil {
				set = make(map[string]struct{})
				aliases[e.ID] = set
			}

			if prev, ok := labels[e.ID]; ok {
				if prev != e.Label {
					if strings.EqualFold(prev, e.Label) {
						// Case-variant of the same label: keep first, fold other as alias.
						set[e.Label] = struct{}{}
					} else {
						return bankPayload{}, fmt.Errorf("entity %s label %q conflicts with %q", e.ID, e.Label, prev)
					}
				}
			} else {
				labels[e.ID] = e.Label
			}

			linkKey := p.ID + "\x00" + e.ID
			if _, ok := seenLink[linkKey]; ok {
				return bankPayload{}, fmt.Errorf("prompt %s lists entity %s twice", p.ID, e.ID)
			}
			seenLink[linkKey] = struct{}{}
			out.PromptEntities = append(out.PromptEntities, linkRow{PromptID: p.ID, EntityID: e.ID, Rarity: e.Rarity})

			for _, a := range e.Aliases {
				a = strings.TrimSpace(a)
				if a == "" || strings.EqualFold(a, labels[e.ID]) {
					continue
				}
				set[a] = struct{}{}
			}
		}
	}

	for id, label := range labels {
		out.Entities = append(out.Entities, entityRow{ID: id, Label: label})
		for alias := range aliases[id] {
			// Drop aliases that only differ by case from the canonical label.
			if strings.EqualFold(alias, label) {
				continue
			}
			out.Aliases = append(out.Aliases, aliasRow{EntityID: id, Alias: alias})
		}
	}
	sort.Slice(out.Categories, func(i, j int) bool { return out.Categories[i].ID < out.Categories[j].ID })
	sort.Slice(out.Prompts, func(i, j int) bool { return out.Prompts[i].ID < out.Prompts[j].ID })
	sort.Slice(out.Entities, func(i, j int) bool { return out.Entities[i].ID < out.Entities[j].ID })
	sort.Slice(out.Aliases, func(i, j int) bool {
		if out.Aliases[i].EntityID != out.Aliases[j].EntityID {
			return out.Aliases[i].EntityID < out.Aliases[j].EntityID
		}
		return out.Aliases[i].Alias < out.Aliases[j].Alias
	})
	sort.Slice(out.PromptEntities, func(i, j int) bool {
		if out.PromptEntities[i].PromptID != out.PromptEntities[j].PromptID {
			return out.PromptEntities[i].PromptID < out.PromptEntities[j].PromptID
		}
		return out.PromptEntities[i].EntityID < out.PromptEntities[j].EntityID
	})
	return out, nil
}

func categorySlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "category"
	}
	return s
}
