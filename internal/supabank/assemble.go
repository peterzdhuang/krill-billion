package supabank

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

// Row is one flattened answer-bank record (one alias, or a blank alias
// when the entity has none).
type Row struct {
	PromptID     string
	PromptText   string
	CategoryName string
	EntityID     string
	Label        string
	Rarity       int
	Alias        string
}

// Assemble turns flat bank rows into one pack. Prompt and entry order
// follow ids, so a shuffled result set still builds the same pack.
func Assemble(rows []Row) (*data.Pack, error) {
	type ent struct {
		label   string
		rarity  int
		aliases []string
		seen    map[string]struct{}
	}
	type pr struct {
		text     string
		category string
		order    []string
		ents     map[string]*ent
	}

	prompts := make(map[string]*pr)
	var order []string
	for _, r := range rows {
		if r.PromptID == "" || r.EntityID == "" {
			return nil, fmt.Errorf("answer bank row missing prompt or entity id")
		}
		if strings.TrimSpace(r.Label) == "" {
			return nil, fmt.Errorf("prompt %s entity %s has an empty label", r.PromptID, r.EntityID)
		}
		if r.Rarity < 0 || r.Rarity > 100 {
			return nil, fmt.Errorf("prompt %s entity %s rarity %d out of range", r.PromptID, r.EntityID, r.Rarity)
		}
		p := prompts[r.PromptID]
		if p == nil {
			p = &pr{
				text:     r.PromptText,
				category: r.CategoryName,
				ents:     make(map[string]*ent),
			}
			prompts[r.PromptID] = p
			order = append(order, r.PromptID)
		} else if p.text != r.PromptText || p.category != r.CategoryName {
			return nil, fmt.Errorf("prompt %s has conflicting text or category", r.PromptID)
		}
		e := p.ents[r.EntityID]
		if e == nil {
			e = &ent{label: r.Label, rarity: r.Rarity, seen: make(map[string]struct{})}
			p.ents[r.EntityID] = e
			p.order = append(p.order, r.EntityID)
		} else if e.label != r.Label || e.rarity != r.Rarity {
			return nil, fmt.Errorf("prompt %s entity %s conflicts across rows", r.PromptID, r.EntityID)
		}
		alias := strings.TrimSpace(r.Alias)
		if alias == "" {
			continue
		}
		if _, ok := e.seen[alias]; ok {
			continue
		}
		e.seen[alias] = struct{}{}
		e.aliases = append(e.aliases, alias)
	}

	sort.Strings(order)
	pack := &data.Pack{
		PackID:      "supabase",
		Version:     1,
		BuiltAt:     time.Now().UTC().Format(time.RFC3339),
		Attribution: []string{"Supabase answer bank"},
	}
	for _, id := range order {
		p := prompts[id]
		sort.Strings(p.order)
		entries := make([]data.Entry, 0, len(p.order))
		for _, eid := range p.order {
			e := p.ents[eid]
			sort.Strings(e.aliases)
			entries = append(entries, data.Entry{
				ID:      eid,
				Label:   e.label,
				Aliases: e.aliases,
				Rarity:  e.rarity,
			})
		}
		pack.Prompts = append(pack.Prompts, data.Prompt{
			ID:       id,
			Category: p.category,
			Text:     p.text,
			Entries:  entries,
		})
	}
	return pack, nil
}
