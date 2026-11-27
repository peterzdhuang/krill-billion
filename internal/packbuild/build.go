package packbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

var qidLabel = regexp.MustCompile(`^Q\d+$`)

// Options configures a pack build.
type Options struct {
	DefsDir  string
	OutPath  string
	PackID   string
	Only     string
	Offline  bool
	CacheDir string
}

// Build loads defs, fetches Wikidata + pageviews, and writes pack JSON.
func Build(ctx context.Context, opt Options) (*data.Pack, error) {
	if opt.DefsDir == "" {
		opt.DefsDir = "packs/defs"
	}
	if opt.OutPath == "" {
		opt.OutPath = "packs/core-v1.json"
	}
	if opt.PackID == "" {
		base := filepath.Base(opt.OutPath)
		opt.PackID = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if opt.CacheDir == "" {
		opt.CacheDir = ".packbuild-cache"
	}

	defs, err := LoadDefs(opt.DefsDir)
	if err != nil {
		return nil, err
	}
	if opt.Only != "" {
		filtered := defs[:0]
		for _, d := range defs {
			if d.ID == opt.Only {
				filtered = append(filtered, d)
			}
		}
		defs = filtered
		if len(defs) == 0 {
			return nil, fmt.Errorf("--only %q matched no defs", opt.Only)
		}
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("no prompt defs in %s", opt.DefsDir)
	}

	wd := &WikidataClient{
		CacheDir: opt.CacheDir,
		Offline:  opt.Offline,
		HTTP:     &http.Client{Timeout: 120 * time.Second},
	}
	pv := &PageviewsClient{
		CacheDir:    opt.CacheDir,
		Offline:     opt.Offline,
		Concurrency: 6,
		HTTP:        &http.Client{Timeout: 30 * time.Second},
	}

	pack := &data.Pack{
		PackID:  opt.PackID,
		Version: 1,
		BuiltAt: time.Now().UTC().Format(time.RFC3339),
		Attribution: []string{
			"Wikidata (CC0)",
			"Wikimedia Pageviews API",
		},
	}

	for i, def := range defs {
		fmt.Fprintf(os.Stderr, "[%d/%d] %s …\n", i+1, len(defs), def.ID)
		prompt, err := buildPrompt(ctx, def, wd, pv)
		if err != nil {
			return nil, fmt.Errorf("prompt %s: %w", def.ID, err)
		}
		fmt.Fprintf(os.Stderr, "  → %d entries\n", len(prompt.Entries))
		pack.Prompts = append(pack.Prompts, *prompt)
	}

	sort.SliceStable(pack.Prompts, func(i, j int) bool {
		if pack.Prompts[i].Category != pack.Prompts[j].Category {
			return pack.Prompts[i].Category < pack.Prompts[j].Category
		}
		return pack.Prompts[i].ID < pack.Prompts[j].ID
	})

	if err := writePack(opt.OutPath, pack); err != nil {
		return nil, err
	}
	return pack, nil
}

func buildPrompt(ctx context.Context, def PromptDef, wd *WikidataClient, pv *PageviewsClient) (*data.Prompt, error) {
	members, err := wd.Query(ctx, def.ID, def.SPARQL)
	if err != nil {
		return nil, err
	}

	type raw struct {
		Member Member
		Views  int64
	}
	candidates := make([]raw, 0, len(members))
	articles := make([]string, 0, len(members))
	for _, m := range members {
		if m.Article == "" {
			continue
		}
		if m.Label == "" || qidLabel.MatchString(m.Label) {
			continue
		}
		articles = append(articles, m.Article)
		candidates = append(candidates, raw{Member: m})
	}

	views, err := pv.FetchMany(ctx, articles)
	if err != nil && len(views) == 0 {
		return nil, err
	}
	for i := range candidates {
		candidates[i].Views = views[candidates[i].Member.Article]
	}

	// Duplicate labels: keep higher views (tie → lower id).
	byLabel := make(map[string]raw)
	for _, c := range candidates {
		key := strings.ToLower(strings.TrimSpace(c.Member.Label))
		prev, ok := byLabel[key]
		if !ok {
			byLabel[key] = c
			continue
		}
		if c.Views > prev.Views || (c.Views == prev.Views && c.Member.ID < prev.Member.ID) {
			byLabel[key] = c
		}
	}

	kept := make([]raw, 0, len(byLabel))
	labelSet := make(map[string]string, len(byLabel)) // lower label → id
	for _, c := range byLabel {
		kept = append(kept, c)
		labelSet[strings.ToLower(c.Member.Label)] = c.Member.ID
	}

	viewed := make([]Viewed, 0, len(kept))
	for _, c := range kept {
		viewed = append(viewed, Viewed{ID: c.Member.ID, Views: c.Views})
	}
	rarities := AssignRarity(viewed)

	entries := make([]data.Entry, 0, len(kept))
	for _, c := range kept {
		aliases := filterAliases(c.Member.Aliases, c.Member.Label, labelSet, c.Member.ID)
		entries = append(entries, data.Entry{
			ID:      c.Member.ID,
			Label:   c.Member.Label,
			Aliases: aliases,
			Rarity:  rarities[c.Member.ID],
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Rarity != entries[j].Rarity {
			return entries[i].Rarity < entries[j].Rarity
		}
		return entries[i].ID < entries[j].ID
	})

	return &data.Prompt{
		ID:       def.ID,
		Category: def.Category,
		Text:     def.Text,
		Entries:  entries,
	}, nil
}

func filterAliases(aliases []string, label string, labelSet map[string]string, selfID string) []string {
	out := make([]string, 0, len(aliases))
	seen := map[string]struct{}{strings.ToLower(label): {}}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if len([]rune(a)) < 2 {
			continue
		}
		k := strings.ToLower(a)
		if _, ok := seen[k]; ok {
			continue
		}
		if owner, ok := labelSet[k]; ok && owner != selfID {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

func writePack(path string, pack *data.Pack) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
