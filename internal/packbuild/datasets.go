package packbuild

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

const (
	geonamesCitiesURL = "https://download.geonames.org/export/dump/cities5000.zip"
	imdbBasicsURL     = "https://datasets.imdbws.com/title.basics.tsv.gz"
	imdbRatingsURL    = "https://datasets.imdbws.com/title.ratings.tsv.gz"
	// Open Food Facts taxonomy of ingredients (CC-BY-SA) — compact vs full product dump.
	offIngredientsURL = "https://static.openfoodfacts.org/data/taxonomies/ingredients.json"
	offCategoriesURL  = "https://static.openfoodfacts.org/data/taxonomies/categories.json"
	offLabelsURL      = "https://static.openfoodfacts.org/data/taxonomies/labels.json"
	// MusicBrainz / Wikipedia entities are covered via Wikidata sitelinks.
	// GeoNames alternate names are inside the cities dump as the asciiname + name fields.
)

// DatasetOptions controls download cache for open datasets.
type DatasetOptions struct {
	CacheDir string
	Offline  bool
	HTTP     *http.Client
}

func (o DatasetOptions) client() *http.Client {
	if o.HTTP != nil {
		return o.HTTP
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

func (o DatasetOptions) cachePath(name string) string {
	dir := o.CacheDir
	if dir == "" {
		dir = ".packbuild-cache"
	}
	return filepath.Join(dir, "datasets", name)
}

func (o DatasetOptions) fetch(ctx context.Context, url, cacheName string) (string, error) {
	path := o.cachePath(cacheName)
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return path, nil
	}
	if o.Offline {
		return "", fmt.Errorf("offline: missing dataset cache %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "  downloading %s …\n", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := o.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 512<<20)); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// BuildGeoNamesCitiesPrompt builds "A city or town" from GeoNames cities5000
// (CC-BY). Rarity uses population (higher pop → lower rarity).
func BuildGeoNamesCitiesPrompt(ctx context.Context, opt DatasetOptions) (data.Prompt, error) {
	zipPath, err := opt.fetch(ctx, geonamesCitiesURL, "cities5000.zip")
	if err != nil {
		return data.Prompt{}, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return data.Prompt{}, err
	}
	defer zr.Close()

	var txt io.ReadCloser
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".txt") {
			rc, err := f.Open()
			if err != nil {
				return data.Prompt{}, err
			}
			txt = rc
			break
		}
	}
	if txt == nil {
		return data.Prompt{}, fmt.Errorf("cities5000.zip: no .txt member")
	}
	defer txt.Close()

	type row struct {
		id   string
		name string
		asc  string
		pop  int64
	}
	byKey := map[string]*row{}
	sc := bufio.NewScanner(txt)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		cols := strings.Split(sc.Text(), "\t")
		if len(cols) < 15 {
			continue
		}
		name := strings.TrimSpace(cols[1])
		ascii := strings.TrimSpace(cols[2])
		if name == "" {
			continue
		}
		pop, _ := strconv.ParseInt(cols[14], 10, 64)
		id := "gn:" + cols[0]
		key := strings.ToLower(name)
		prev := byKey[key]
		if prev == nil || pop > prev.pop {
			byKey[key] = &row{id: id, name: name, asc: ascii, pop: pop}
		}
	}
	if err := sc.Err(); err != nil {
		return data.Prompt{}, err
	}

	scored := make([]Viewed, 0, len(byKey))
	entries := make([]data.Entry, 0, len(byKey))
	for _, r := range byKey {
		aliases := ExpandAliases(r.name, nil)
		if r.asc != "" && !strings.EqualFold(r.asc, r.name) {
			aliases = ExpandAliases(r.name, append(aliases, r.asc))
		}
		entries = append(entries, data.Entry{ID: r.id, Label: r.name, Aliases: aliases})
		scored = append(scored, Viewed{ID: r.id, Views: r.pop})
	}
	rarities := AssignRarity(scored)
	for i := range entries {
		entries[i].Rarity = rarities[entries[i].ID]
	}
	entries = FoldCaseVariantEntries(entries)
	sortEntries(entries)
	return data.Prompt{
		ID:       "city-or-town",
		Category: "Geography",
		Text:     "A city or town",
		Entries:  entries,
	}, nil
}

// BuildIMDbPrompts builds film and TV-show prompts from IMDb non-commercial datasets.
func BuildIMDbPrompts(ctx context.Context, opt DatasetOptions) ([]data.Prompt, error) {
	basicsPath, err := opt.fetch(ctx, imdbBasicsURL, "title.basics.tsv.gz")
	if err != nil {
		return nil, err
	}
	ratingsPath, err := opt.fetch(ctx, imdbRatingsURL, "title.ratings.tsv.gz")
	if err != nil {
		return nil, err
	}
	votes, err := loadIMDbRatings(ratingsPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(basicsPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	r := csv.NewReader(gz)
	r.Comma = '\t'
	r.LazyQuotes = true
	r.ReuseRecord = true
	r.FieldsPerRecord = -1 // IMDb rows occasionally have ragged fields
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	need := []string{"tconst", "titleType", "primaryTitle", "originalTitle"}
	for _, n := range need {
		if _, ok := col[n]; !ok {
			return nil, fmt.Errorf("imdb basics missing column %s", n)
		}
	}

	type bucket struct {
		id, label, orig string
		votes           int64
	}
	films := map[string]*bucket{}
	shows := map[string]*bucket{}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) <= col["originalTitle"] {
			continue
		}
		tt := rec[col["titleType"]]
		title := strings.TrimSpace(rec[col["primaryTitle"]])
		if title == "" {
			continue
		}
		id := rec[col["tconst"]]
		orig := strings.TrimSpace(rec[col["originalTitle"]])
		v := votes[id]
		key := strings.ToLower(title)
		switch tt {
		case "movie":
			prev := films[key]
			if prev == nil || v > prev.votes {
				films[key] = &bucket{id: "imdb:" + id, label: title, orig: orig, votes: v}
			}
		case "tvSeries", "tvMiniSeries":
			prev := shows[key]
			if prev == nil || v > prev.votes {
				shows[key] = &bucket{id: "imdb:" + id, label: title, orig: orig, votes: v}
			}
		}
	}

	mk := func(id, category, text string, bags map[string]*bucket) data.Prompt {
		scored := make([]Viewed, 0, len(bags))
		entries := make([]data.Entry, 0, len(bags))
		for _, b := range bags {
			src := []string{}
			if b.orig != "" && !strings.EqualFold(b.orig, b.label) {
				src = append(src, b.orig)
			}
			entries = append(entries, data.Entry{
				ID:      b.id,
				Label:   b.label,
				Aliases: ExpandAliases(b.label, src),
			})
			scored = append(scored, Viewed{ID: b.id, Views: b.votes})
		}
		rarities := AssignRarity(scored)
		for i := range entries {
			entries[i].Rarity = rarities[entries[i].ID]
		}
		entries = FoldCaseVariantEntries(entries)
		sortEntries(entries)
		return data.Prompt{ID: id, Category: category, Text: text, Entries: entries}
	}

	return []data.Prompt{
		mk("film", "Entertainment", "A film", films),
		mk("tv-show", "Entertainment", "A TV show", shows),
	}, nil
}

func loadIMDbRatings(path string) (map[string]int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	r := csv.NewReader(gz)
	r.Comma = '\t'
	r.ReuseRecord = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	out := make(map[string]int64, 1<<20)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) <= col["numVotes"] {
			continue
		}
		v, _ := strconv.ParseInt(rec[col["numVotes"]], 10, 64)
		out[rec[col["tconst"]]] = v
	}
	return out, nil
}

// BuildOpenFoodFactsPrompt builds a food prompt from OFF taxonomies (ingredients,
// categories, labels) merged into one open-ended bank.
func BuildOpenFoodFactsPrompt(ctx context.Context, opt DatasetOptions) (data.Prompt, error) {
	type node struct {
		Name map[string]string `json:"name"`
	}
	byKey := map[string]data.Entry{}
	loadTaxonomy := func(url, cacheName string) error {
		path, err := opt.fetch(ctx, url, cacheName)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var raw map[string]node
		if err := json.Unmarshal(b, &raw); err != nil {
			return fmt.Errorf("parse %s: %w", cacheName, err)
		}
		for id, n := range raw {
			label := strings.TrimSpace(n.Name["en"])
			if label == "" {
				continue
			}
			lower := strings.ToLower(label)
			if strings.HasPrefix(lower, "category:") || strings.HasPrefix(lower, "en:") {
				continue
			}
			key := lower
			if prev, ok := byKey[key]; ok {
				prev.Aliases = ExpandAliases(prev.Label, append(prev.Aliases, label))
				byKey[key] = prev
				continue
			}
			// Hash the English label so the same OFF concept under different
			// taxonomy ids cannot collide as one entity with two labels.
			eid := "off:" + shortHash(key)
			byKey[key] = data.Entry{
				ID:      eid,
				Label:   label,
				Aliases: ExpandAliases(label, nil),
				Rarity:  50,
			}
			_ = id
		}
		return nil
	}
	for _, src := range []struct{ url, cache string }{
		{offIngredientsURL, "off-ingredients.json"},
		{offCategoriesURL, "off-categories.json"},
		{offLabelsURL, "off-labels.json"},
	} {
		if err := loadTaxonomy(src.url, src.cache); err != nil {
			fmt.Fprintf(os.Stderr, "  off %s: %v\n", src.cache, err)
		}
	}
	if len(byKey) == 0 {
		return data.Prompt{}, fmt.Errorf("open food facts: no taxonomy rows")
	}
	entries := make([]data.Entry, 0, len(byKey))
	scored := make([]Viewed, 0, len(byKey))
	for _, e := range byKey {
		entries = append(entries, e)
		score := int64(1000 - len(e.Label)*10)
		if score < 1 {
			score = 1
		}
		scored = append(scored, Viewed{ID: e.ID, Views: score})
	}
	rarities := AssignRarity(scored)
	for i := range entries {
		entries[i].Rarity = rarities[entries[i].ID]
	}
	entries = FoldCaseVariantEntries(entries)
	sortEntries(entries)
	return data.Prompt{
		ID:       "food-or-ingredient",
		Category: "Food",
		Text:     "A food or ingredient",
		Entries:  entries,
	}, nil
}

func shortHash(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:10])
}

func sortEntries(entries []data.Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Rarity != entries[j].Rarity {
			return entries[i].Rarity < entries[j].Rarity
		}
		return entries[i].ID < entries[j].ID
	})
}
