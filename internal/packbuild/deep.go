package packbuild

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

// DeepOptions configures the multi-source deep pack build.
type DeepOptions struct {
	OutPath       string
	PackID        string
	CacheDir      string
	Offline       bool
	MinEntries    int
	MaxEntries    int // per-prompt cap; 0 → DefaultMaxEntries; <0 → unlimited
	SkipDatasets  bool
	SkipWikidata  bool
}

// deepClass is one paginated Wikidata class used for open-ended prompts.
type deepClass struct {
	ID             string
	Category       string
	Text           string
	Where          string
	RequireArticle bool
}

// Lean WHERE bodies (no alias GROUP_CONCAT) — ExpandAliases fills short forms.
var deepClasses = []deepClass{
	{
		ID: "actor", Category: "Entertainment", Text: "An actor",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q5 ; wdt:P106 wd:Q33999 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "musician", Category: "Music", Text: "A musician",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q5 ; wdt:P106 wd:Q639669 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "video-game", Category: "Games", Text: "A video game",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q7889 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "book", Category: "Literature", Text: "A book",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q7725634 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "company", Category: "Business", Text: "A company",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q4830453 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "chemical-compound", Category: "Science", Text: "A chemical compound",
		RequireArticle: false,
		Where: `
  ?item wdt:P31 wd:Q11173 .
  OPTIONAL {
    ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  }
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "animal-species", Category: "Animals", Text: "A species of animal",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q16521 ; wdt:P105 wd:Q7432 .
  ?item wdt:P171* wd:Q729 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "song", Category: "Music", Text: "A song",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q7366 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "album", Category: "Music", Text: "An album",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q482994 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "association-football-player", Category: "Sports", Text: "A soccer player",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q5 ; wdt:P106 wd:Q937857 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "mountain", Category: "Geography", Text: "A mountain",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q8502 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "river", Category: "Geography", Text: "A river",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q4022 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
	{
		ID: "brand", Category: "Business", Text: "A brand",
		RequireArticle: true,
		Where: `
  ?item wdt:P31 wd:Q431289 .
  ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> .
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }`,
	},
}

// BuildDeep builds a multi-source deep prompt pack and writes JSON (and optional .gz).
func BuildDeep(ctx context.Context, opt DeepOptions) (*data.Pack, error) {
	if opt.OutPath == "" {
		opt.OutPath = "packs/deep-v1.json"
	}
	if opt.PackID == "" {
		opt.PackID = strings.TrimSuffix(filepath.Base(opt.OutPath), filepath.Ext(opt.OutPath))
	}
	if opt.CacheDir == "" {
		opt.CacheDir = ".packbuild-cache"
	}
	if opt.MinEntries <= 0 {
		opt.MinEntries = MinEntries
	}
	maxEntries := opt.MaxEntries
	if maxEntries == 0 {
		maxEntries = DefaultMaxEntries
	}

	wd := &WikidataClient{
		CacheDir: opt.CacheDir,
		Offline:  opt.Offline,
		HTTP:     &http.Client{Timeout: 180 * time.Second},
	}
	ds := DatasetOptions{CacheDir: opt.CacheDir, Offline: opt.Offline}

	var prompts []data.Prompt
	attr := []string{
		"Wikidata (CC0)",
		"GeoNames (CC-BY 4.0)",
		"IMDb non-commercial datasets",
		"Open Food Facts (ODbL / CC-BY-SA)",
		"MusicBrainz / Wikipedia entities via Wikidata sitelinks",
	}

	if !opt.SkipDatasets {
		fmt.Fprintf(os.Stderr, "[datasets] GeoNames cities5000 …\n")
		if p, err := BuildGeoNamesCitiesPrompt(ctx, ds); err != nil {
			fmt.Fprintf(os.Stderr, "  geonames: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  → %d entries\n", len(p.Entries))
			prompts = append(prompts, p)
		}

		fmt.Fprintf(os.Stderr, "[datasets] IMDb titles …\n")
		if list, err := BuildIMDbPrompts(ctx, ds); err != nil {
			fmt.Fprintf(os.Stderr, "  imdb: %v\n", err)
		} else {
			for _, p := range list {
				fmt.Fprintf(os.Stderr, "  → %s: %d entries\n", p.ID, len(p.Entries))
				prompts = append(prompts, p)
			}
		}

		fmt.Fprintf(os.Stderr, "[datasets] Open Food Facts taxonomies …\n")
		if p, err := BuildOpenFoodFactsPrompt(ctx, ds); err != nil {
			fmt.Fprintf(os.Stderr, "  off: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  → %d entries\n", len(p.Entries))
			prompts = append(prompts, p)
		}
	}

	if !opt.SkipWikidata {
		for i, dc := range deepClasses {
			fmt.Fprintf(os.Stderr, "[%d/%d] wikidata %s …\n", i+1, len(deepClasses), dc.ID)
			p, err := buildDeepClass(ctx, wd, dc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  skip %s: %v\n", dc.ID, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "  → %d entries\n", len(p.Entries))
			prompts = append(prompts, p)
		}
	} else {
		_ = wd
		fmt.Fprintf(os.Stderr, "[wikidata] skipped\n")
	}

	if maxEntries > 0 {
		prompts = CapAllPromptEntries(prompts, maxEntries)
		fmt.Fprintf(os.Stderr, "max-entries %d applied\n", maxEntries)
	}
	kept, removed := FilterPromptsMinEntries(prompts, opt.MinEntries)
	fmt.Fprintf(os.Stderr, "min-entries %d: kept %d, removed %d\n", opt.MinEntries, len(kept), removed)

	pack := &data.Pack{
		PackID:      opt.PackID,
		Version:     1,
		BuiltAt:     time.Now().UTC().Format(time.RFC3339),
		Attribution: attr,
		Prompts:     kept,
	}
	if err := writePack(opt.OutPath, pack); err != nil {
		return nil, err
	}
	if err := writePackGZ(opt.OutPath+".gz", pack); err != nil {
		fmt.Fprintf(os.Stderr, "warn: gzip: %v\n", err)
	}
	return pack, nil
}

func buildDeepClass(ctx context.Context, wd *WikidataClient, dc deepClass) (data.Prompt, error) {
	members, err := wd.QueryPaginated(ctx, dc.ID, 2500, 60, func(after string, limit int) string {
		return PaginatedSPARQL(dc.Where, after, limit)
	})
	if err != nil {
		return data.Prompt{}, err
	}
	byLabel := map[string]Member{}
	for _, m := range members {
		if m.Label == "" || qidLabel.MatchString(m.Label) {
			continue
		}
		if dc.RequireArticle && m.Article == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(m.Label))
		prev, ok := byLabel[key]
		if !ok || m.Sitelinks > prev.Sitelinks || (m.Sitelinks == prev.Sitelinks && m.ID < prev.ID) {
			byLabel[key] = m
		}
	}
	if len(byLabel) == 0 {
		return data.Prompt{}, fmt.Errorf("no usable rows")
	}

	scored := make([]Scored, 0, len(byLabel))
	entries := make([]data.Entry, 0, len(byLabel))
	for _, m := range byLabel {
		scored = append(scored, Scored{ID: m.ID, Sitelinks: int64(m.Sitelinks)})
	}
	rarities := AssignBlendedRarity(scored)
	for _, m := range byLabel {
		entries = append(entries, data.Entry{
			ID:      m.ID,
			Label:   m.Label,
			Aliases: ExpandAliases(m.Label, m.Aliases),
			Rarity:  rarities[m.ID],
		})
	}
	entries = FoldCaseVariantEntries(entries)
	sortEntries(entries)
	return data.Prompt{
		ID:       dc.ID,
		Category: dc.Category,
		Text:     dc.Text,
		Entries:  entries,
	}, nil
}
