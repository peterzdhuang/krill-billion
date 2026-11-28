package packbuild

import (
	"compress/gzip"
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
	DefsDir    string
	OutPath    string
	PackID     string
	Only       string
	Offline    bool
	CacheDir   string
	MinEntries int // drop prompts below this; 0 = keep all (legacy packs)
}

type builtPrompt struct {
	prompt        data.Prompt
	check         string
	disagreed     bool
	pageviewsUsed bool
}

type raw struct {
	Member Member
	Views  int64
}

// Build loads defs, fetches Wikidata, sitelinks, pageviews, and any second
// membership list, then writes pack JSON.
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
	checks := &CheckClient{
		CacheDir: opt.CacheDir,
		Offline:  opt.Offline,
		HTTP:     &http.Client{Timeout: 90 * time.Second},
	}

	var (
		pageviewsUsed bool
		restUsed      bool
		pokeUsed      bool
		restDisagree  []string
		pokeDisagree  []string
		prompts       []data.Prompt
	)

	for i, def := range defs {
		fmt.Fprintf(os.Stderr, "[%d/%d] %s …\n", i+1, len(defs), def.ID)
		built, err := buildPrompt(ctx, def, wd, pv, checks)
		if err != nil {
			return nil, fmt.Errorf("prompt %s: %w", def.ID, err)
		}
		note := ""
		if built.disagreed {
			note = " (kept Wikidata membership)"
		}
		fmt.Fprintf(os.Stderr, "  → %d entries%s\n", len(built.prompt.Entries), note)
		switch built.check {
		case "restcountries":
			restUsed = true
			if built.disagreed {
				restDisagree = append(restDisagree, def.ID)
			}
		case "pokeapi":
			pokeUsed = true
			if built.disagreed {
				pokeDisagree = append(pokeDisagree, def.ID)
			}
		}
		if built.pageviewsUsed {
			pageviewsUsed = true
		}
		prompts = append(prompts, built.prompt)
	}

	sort.SliceStable(prompts, func(i, j int) bool {
		if prompts[i].Category != prompts[j].Category {
			return prompts[i].Category < prompts[j].Category
		}
		return prompts[i].ID < prompts[j].ID
	})
	sort.Strings(restDisagree)
	sort.Strings(pokeDisagree)

	removed := 0
	if opt.MinEntries > 0 {
		prompts, removed = FilterPromptsMinEntries(prompts, opt.MinEntries)
		fmt.Fprintf(os.Stderr, "min-entries %d: kept %d, removed %d\n", opt.MinEntries, len(prompts), removed)
	}

	pack := &data.Pack{
		PackID:      opt.PackID,
		Version:     1,
		BuiltAt:     time.Now().UTC().Format(time.RFC3339),
		Attribution: packAttribution(pageviewsUsed, restUsed, pokeUsed, restDisagree, pokeDisagree),
		Prompts:     prompts,
	}
	if len(restDisagree) > 0 {
		fmt.Fprintf(os.Stderr, "REST Countries disagreement: %s\n", strings.Join(restDisagree, ", "))
	}
	if len(pokeDisagree) > 0 {
		fmt.Fprintf(os.Stderr, "PokeAPI disagreement: %s\n", strings.Join(pokeDisagree, ", "))
	}
	fmt.Fprintf(os.Stderr, "attribution: %s\n", strings.Join(pack.Attribution, " | "))

	if err := writePack(opt.OutPath, pack); err != nil {
		return nil, err
	}
	return pack, nil
}

func packAttribution(pageviewsUsed, restUsed, pokeUsed bool, restDisagree, pokeDisagree []string) []string {
	attr := []string{"Wikidata (CC0)"}
	if pageviewsUsed {
		attr = append(attr, "Wikimedia Pageviews API")
	}
	if restUsed {
		attr = append(attr, "REST Countries")
	}
	if pokeUsed {
		attr = append(attr, "PokeAPI")
	}
	if len(restDisagree) > 0 {
		attr = append(attr, "REST Countries disagreement: kept Wikidata membership")
	}
	if len(pokeDisagree) > 0 {
		attr = append(attr, "PokeAPI disagreement: kept Wikidata membership")
	}
	return attr
}

func buildPrompt(ctx context.Context, def PromptDef, wd *WikidataClient, pv *PageviewsClient, checks *CheckClient) (*builtPrompt, error) {
	if len(def.Entries) > 0 {
		return buildStaticPrompt(def)
	}
	members, err := wd.Query(ctx, def.ID, def.SPARQL)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("wikidata returned no rows")
	}
	candidates := make([]raw, 0, len(members))
	for _, m := range members {
		if m.Article == "" {
			continue
		}
		if m.Label == "" || qidLabel.MatchString(m.Label) {
			continue
		}
		candidates = append(candidates, raw{Member: m})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no wikidata rows with an enwiki article")
	}

	needLinks := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if !c.Member.SitelinksKnown {
			needLinks = append(needLinks, c.Member.ID)
		}
	}
	counts, err := wd.SitelinkCounts(ctx, needLinks)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		m := &candidates[i].Member
		if m.SitelinksKnown {
			continue
		}
		if n, ok := counts[m.ID]; ok {
			m.Sitelinks = n
			m.SitelinksKnown = true
		}
	}

	disagreed := false
	if def.Check != "" {
		list, err := checks.List(ctx, def.Check)
		if err != nil {
			return nil, err
		}
		var kept []raw
		kept, disagreed = applySecondSource(candidates, list)
		fmt.Fprintf(os.Stderr, "  %s: %d/%d wikidata rows in the second list\n", def.Check, countMatches(candidates, list), len(candidates))
		candidates = kept
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no entries after membership check")
	}

	articles := make([]string, 0, len(candidates))
	for _, c := range candidates {
		articles = append(articles, c.Member.Article)
	}
	fmt.Fprintf(os.Stderr, "  pageviews: %d articles\n", len(articles))
	views, err := pv.FetchMany(ctx, articles)
	pageviewsUsed := err == nil && !allZeroViews(views)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  pageviews unavailable (%v); sitelinks only\n", err)
	} else if !pageviewsUsed {
		fmt.Fprintf(os.Stderr, "  pageviews all zero; sitelinks only\n")
	}
	for i := range candidates {
		if !pageviewsUsed {
			candidates[i].Views = 0
			continue
		}
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

	scored := make([]Scored, 0, len(kept))
	for _, c := range kept {
		scored = append(scored, Scored{
			ID:        c.Member.ID,
			Views:     c.Views,
			Sitelinks: int64(c.Member.Sitelinks),
		})
	}
	rarities := AssignBlendedRarity(scored)

		entries := make([]data.Entry, 0, len(kept))
	for _, c := range kept {
		aliases := ExpandAliases(c.Member.Label, filterAliases(c.Member.Aliases, c.Member.Label, labelSet, c.Member.ID))
		entries = append(entries, data.Entry{
			ID:      c.Member.ID,
			Label:   c.Member.Label,
			Aliases: aliases,
			Rarity:  rarities[c.Member.ID],
		})
	}
	entries = FoldCaseVariantEntries(entries)
	sortEntries(entries)

	return &builtPrompt{
		prompt: data.Prompt{
			ID:       def.ID,
			Category: def.Category,
			Text:     def.Text,
			Entries:  entries,
		},
		check:         def.Check,
		disagreed:     disagreed,
		pageviewsUsed: pageviewsUsed,
	}, nil
}

func buildStaticPrompt(def PromptDef) (*builtPrompt, error) {
	entries := make([]data.Entry, 0, len(def.Entries))
	seen := map[string]struct{}{}
	for i, e := range def.Entries {
		label := strings.TrimSpace(e.Label)
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		id := fmt.Sprintf("static-%s-%d", def.ID, i+1)
		aliases := ExpandAliases(label, e.Aliases)
		entries = append(entries, data.Entry{
			ID:      id,
			Label:   label,
			Aliases: aliases,
			Rarity:  e.Rarity,
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no static entries")
	}
	entries = FoldCaseVariantEntries(entries)
	sortEntries(entries)
	return &builtPrompt{
		prompt: data.Prompt{
			ID:       def.ID,
			Category: def.Category,
			Text:     def.Text,
			Entries:  entries,
		},
	}, nil
}

// applySecondSource keeps the intersection of Wikidata rows and list.
// If that would drop more than 40% of the Wikidata rows, the Wikidata set is
// kept and disagreed is true. Rows that do match still pick up the second
// source's spellings. Names with no Wikidata id are never added.
func applySecondSource(rows []raw, list *ExternalList) ([]raw, bool) {
	if list == nil || len(rows) == 0 {
		return rows, false
	}
	matched := make([]raw, 0, len(rows))
	matchedAt := make(map[string][]string, len(rows))
	for _, row := range rows {
		ok, extras := list.Match(row.Member.Label, row.Member.Aliases)
		if !ok {
			continue
		}
		aliases := make([]string, 0, len(row.Member.Aliases)+len(extras))
		aliases = append(aliases, row.Member.Aliases...)
		aliases = append(aliases, extras...)
		row.Member.Aliases = aliases
		matched = append(matched, row)
		matchedAt[row.Member.ID] = aliases
	}
	if !keepIntersection(len(rows), len(matched)) {
		out := make([]raw, len(rows))
		for i, row := range rows {
			if aliases, ok := matchedAt[row.Member.ID]; ok {
				row.Member.Aliases = aliases
			}
			out[i] = row
		}
		return out, true
	}
	return matched, false
}

// keepIntersection is false when the overlap would drop more than 40% of rows.
func keepIntersection(wikidataRows, intersectionRows int) bool {
	if wikidataRows <= 0 || intersectionRows >= wikidataRows {
		return true
	}
	dropped := wikidataRows - intersectionRows
	return dropped*5 <= wikidataRows*2
}

func countMatches(rows []raw, list *ExternalList) int {
	n := 0
	for _, row := range rows {
		if ok, _ := list.Match(row.Member.Label, row.Member.Aliases); ok {
			n++
		}
	}
	return n
}

func allZeroViews(views map[string]int64) bool {
	if len(views) == 0 {
		return true
	}
	for _, v := range views {
		if v != 0 {
			return false
		}
	}
	return true
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writePackGZ(path string, pack *data.Pack) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	enc.SetIndent("", "  ")
	if err := enc.Encode(pack); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := gz.Close(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// LoadPackFile reads a pack JSON or .json.gz file.
func LoadPackFile(path string) (*data.Pack, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r interface{ Read([]byte) (int, error) } = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	}
	var pack data.Pack
	if err := json.NewDecoder(r).Decode(&pack); err != nil {
		return nil, err
	}
	return &pack, nil
}
