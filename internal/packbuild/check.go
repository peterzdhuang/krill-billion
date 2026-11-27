package packbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	restCountriesURL = "https://restcountries.com/v3.1/all"
	// Upstream open dataset the REST Countries API was built from.
	// v3.1/all currently 301s to a deprecation error, so a build that cannot
	// parse a country list from that URL uses this file instead.
	restCountriesFallbackURL = "https://raw.githubusercontent.com/mledoze/countries/master/countries.json"
	pokeAPIURL               = "https://pokeapi.co/api/v2/pokemon?limit=2000"

	restCountriesMin = 100
	pokeAPIMin       = 800
)

// CheckClient loads optional second membership lists, cached on disk.
type CheckClient struct {
	HTTP     *http.Client
	CacheDir string
	Offline  bool

	rest *ExternalList
	poke *ExternalList
}

// List returns the external name list for a def check value.
// An empty check returns a nil list.
func (c *CheckClient) List(ctx context.Context, check string) (*ExternalList, error) {
	switch check {
	case "":
		return nil, nil
	case "restcountries":
		return c.restCountries(ctx)
	case "pokeapi":
		return c.pokeAPI(ctx)
	default:
		return nil, fmt.Errorf("unknown check %q", check)
	}
}

func (c *CheckClient) restCountries(ctx context.Context) (*ExternalList, error) {
	if c.rest != nil {
		return c.rest, nil
	}
	cachePath := c.checkCachePath("restcountries.json")
	if b, err := os.ReadFile(cachePath); err == nil {
		if list, err := parseRESTCountries(b); err == nil && list.n >= restCountriesMin {
			c.rest = list
			return list, nil
		}
	}
	if c.Offline {
		return nil, fmt.Errorf("offline: no REST Countries cache")
	}

	body, err := c.get(ctx, restCountriesURL)
	if err == nil {
		if list, perr := parseRESTCountries(body); perr == nil && list.n >= restCountriesMin {
			if werr := c.writeCache("restcountries.json", body); werr != nil {
				return nil, werr
			}
			fmt.Fprintf(os.Stderr, "  restcountries: %d names from %s\n", list.n, restCountriesURL)
			c.rest = list
			return list, nil
		} else if perr != nil {
			err = perr
		} else {
			err = fmt.Errorf("only %d countries", list.n)
		}
	}
	fmt.Fprintf(os.Stderr, "  restcountries: %s unusable (%v); using upstream dataset\n", restCountriesURL, err)

	body, err = c.get(ctx, restCountriesFallbackURL)
	if err != nil {
		return nil, fmt.Errorf("restcountries: %w", err)
	}
	list, err := parseRESTCountries(body)
	if err != nil {
		return nil, fmt.Errorf("restcountries fallback: %w", err)
	}
	if list.n < restCountriesMin {
		return nil, fmt.Errorf("restcountries fallback: only %d countries", list.n)
	}
	if err := c.writeCache("restcountries.json", body); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "  restcountries: %d names from upstream dataset\n", list.n)
	c.rest = list
	return list, nil
}

func (c *CheckClient) pokeAPI(ctx context.Context) (*ExternalList, error) {
	if c.poke != nil {
		return c.poke, nil
	}
	cachePath := c.checkCachePath("pokeapi.json")
	if b, err := os.ReadFile(cachePath); err == nil {
		if list, err := parsePokeAPI(b); err == nil && list.n >= pokeAPIMin {
			c.poke = list
			return list, nil
		}
	}
	if c.Offline {
		return nil, fmt.Errorf("offline: no PokeAPI cache")
	}
	body, err := c.fetchPokeAPI(ctx)
	if err != nil {
		return nil, err
	}
	list, err := parsePokeAPI(body)
	if err != nil {
		return nil, err
	}
	if list.n < pokeAPIMin {
		return nil, fmt.Errorf("pokeapi: only %d names", list.n)
	}
	if err := c.writeCache("pokeapi.json", body); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "  pokeapi: %d names\n", list.n)
	c.poke = list
	return list, nil
}

func (c *CheckClient) fetchPokeAPI(ctx context.Context) ([]byte, error) {
	type pokePage struct {
		Count   int    `json:"count"`
		Next    string `json:"next"`
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	var all []struct {
		Name string `json:"name"`
	}
	next := pokeAPIURL
	count := 0
	for pages := 0; pages < 5 && next != ""; pages++ {
		body, err := c.get(ctx, next)
		if err != nil {
			return nil, fmt.Errorf("pokeapi: %w", err)
		}
		var page pokePage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("pokeapi decode: %w", err)
		}
		count = page.Count
		all = append(all, page.Results...)
		next = page.Next
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("pokeapi: empty list")
	}
	if count > len(all) {
		return nil, fmt.Errorf("pokeapi: got %d of %d", len(all), count)
	}
	payload := struct {
		Count   int `json:"count"`
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}{Count: len(all), Results: all}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *CheckClient) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		client := c.HTTP
		if client == nil {
			client = &http.Client{Timeout: 90 * time.Second}
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 160))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 160))
		}
		return body, nil
	}
	return nil, lastErr
}

func (c *CheckClient) checkCachePath(name string) string {
	dir := c.CacheDir
	if dir == "" {
		dir = ".packbuild-cache"
	}
	return filepath.Join(dir, "checks", name)
}

func (c *CheckClient) writeCache(name string, body []byte) error {
	path := c.checkCachePath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type restCountry struct {
	Name struct {
		Common   string `json:"common"`
		Official string `json:"official"`
	} `json:"name"`
	AltSpellings []string `json:"altSpellings"`
}

func parseRESTCountries(body []byte) (*ExternalList, error) {
	var countries []restCountry
	if err := json.Unmarshal(body, &countries); err != nil {
		return nil, err
	}
	list := &ExternalList{}
	for _, country := range countries {
		names := make([]string, 0, 2+len(country.AltSpellings))
		names = append(names, country.Name.Common, country.Name.Official)
		names = append(names, country.AltSpellings...)
		list.add(names)
	}
	if list.n == 0 {
		return nil, fmt.Errorf("restcountries: no names")
	}
	return list, nil
}

func parsePokeAPI(body []byte) (*ExternalList, error) {
	var page struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}
	list := &ExternalList{}
	for _, row := range page.Results {
		list.add([]string{row.Name})
	}
	if list.n == 0 {
		return nil, fmt.Errorf("pokeapi: no names")
	}
	return list, nil
}

// ExternalList is a second membership source, indexed by normalized name.
type ExternalList struct {
	byKey map[string][]*extRecord
	n     int
}

type extRecord struct {
	names []string
}

func (l *ExternalList) add(names []string) {
	if l.byKey == nil {
		l.byKey = map[string][]*extRecord{}
	}
	cleaned := uniqueFold(names)
	if len(cleaned) == 0 {
		return
	}
	rec := &extRecord{names: cleaned}
	seen := map[string]struct{}{}
	added := false
	for _, name := range cleaned {
		key := normalizeKey(name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		l.byKey[key] = append(l.byKey[key], rec)
		added = true
	}
	if added {
		l.n++
	}
}

// Match reports whether label or any alias hits this list.
// The returned names are the matched record's spellings, in a stable order.
func (l *ExternalList) Match(label string, aliases []string) (bool, []string) {
	if l == nil || len(l.byKey) == 0 {
		return false, nil
	}
	var recs []*extRecord
	seen := map[*extRecord]struct{}{}
	consider := func(s string) {
		key := normalizeKey(s)
		if key == "" {
			return
		}
		for _, rec := range l.byKey[key] {
			if _, ok := seen[rec]; ok {
				continue
			}
			seen[rec] = struct{}{}
			recs = append(recs, rec)
		}
	}
	consider(label)
	for _, alias := range aliases {
		consider(alias)
	}
	if len(recs) == 0 {
		return false, nil
	}
	extras := make([]string, 0)
	for _, rec := range recs {
		extras = append(extras, rec.names...)
	}
	sort.Strings(extras)
	return true, uniqueFold(extras)
}

// normalizeKey lowercases, strips accents and punctuation, and folds hyphens
// to spaces so "Mr. Mime" matches "mr-mime" and "Côte d'Ivoire" matches
// "Cote d'Ivoire". A leading "the " is removed so "The Gambia" matches "Gambia".
func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripMarks(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '-', '‐', '‑', '–', '—', '―', '_', '/':
			b.WriteRune(' ')
		default:
			if unicode.IsPunct(r) || unicode.IsSymbol(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	s = strings.Join(strings.Fields(b.String()), " ")
	if strings.HasPrefix(s, "the ") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "the "))
	}
	return s
}

func stripMarks(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
