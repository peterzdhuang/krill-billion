package packbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	wikidataEndpoint = "https://query.wikidata.org/sparql"
	userAgent        = "KrillBillionPackbuild/0.1 (https://github.com/peterzdhuang/krill-billion; ph1425015107@gmail.com)"
)

var qidFromURI = regexp.MustCompile(`/(Q\d+)$`)

// Member is one Wikidata entity with enwiki sitelink and English aliases.
type Member struct {
	ID      string
	Label   string
	Article string // enwiki title
	Aliases []string
}

type sparqlResponse struct {
	Results struct {
		Bindings []map[string]struct {
			Type     string `json:"type"`
			Value    string `json:"value"`
			XMLLang  string `json:"xml:lang"`
			DataType string `json:"datatype"`
		} `json:"bindings"`
	} `json:"results"`
}

// WikidataClient fetches SPARQL results with disk cache, retries, and rate limiting.
type WikidataClient struct {
	HTTP     *http.Client
	CacheDir string
	Offline  bool
	MinGap   time.Duration
	lastAt   time.Time
}

func (c *WikidataClient) Query(ctx context.Context, promptID, sparql string) ([]Member, error) {
	cachePath := filepath.Join(c.CacheDir, "sparql", promptID+".json")
	if b, err := os.ReadFile(cachePath); err == nil {
		var members []Member
		if err := json.Unmarshal(b, &members); err == nil {
			return members, nil
		}
	}
	if c.Offline {
		return nil, fmt.Errorf("offline: no sparql cache for %s", promptID)
	}

	if err := c.wait(ctx); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		members, err := c.fetchSPARQL(ctx, sparql)
		if err == nil {
			if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
				return nil, err
			}
			b, err := json.MarshalIndent(members, "", "  ")
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(cachePath, b, 0o644); err != nil {
				return nil, err
			}
			return members, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *WikidataClient) wait(ctx context.Context) error {
	gap := c.MinGap
	if gap <= 0 {
		gap = 1100 * time.Millisecond
	}
	if c.lastAt.IsZero() {
		c.lastAt = time.Now()
		return nil
	}
	wait := gap - time.Since(c.lastAt)
	if wait <= 0 {
		c.lastAt = time.Now()
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		c.lastAt = time.Now()
		return nil
	}
}

func (c *WikidataClient) fetchSPARQL(ctx context.Context, sparql string) ([]Member, error) {
	u, err := url.Parse(wikidataEndpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("format", "json")
	q.Set("query", sparql)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/sparql-results+json")

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikidata HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var parsed sparqlResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode sparql: %w", err)
	}

	byID := make(map[string]*Member)
	for _, b := range parsed.Results.Bindings {
		itemURI := b["item"].Value
		id := extractQID(itemURI)
		if id == "" {
			continue
		}
		m, ok := byID[id]
		if !ok {
			m = &Member{ID: id}
			byID[id] = m
		}
		if lab := b["itemLabel"].Value; lab != "" {
			m.Label = lab
		}
		if art := b["article"].Value; art != "" {
			m.Article = articleTitleFromURL(art)
		}
		if al := b["alias"].Value; al != "" {
			m.Aliases = append(m.Aliases, al)
		}
		if als := b["aliases"].Value; als != "" {
			for _, part := range strings.Split(als, "|") {
				part = strings.TrimSpace(part)
				if part != "" {
					m.Aliases = append(m.Aliases, part)
				}
			}
		}
	}

	out := make([]Member, 0, len(byID))
	for _, m := range byID {
		m.Aliases = uniqueFold(m.Aliases)
		out = append(out, *m)
	}
	return out, nil
}

func extractQID(uri string) string {
	m := qidFromURI.FindStringSubmatch(uri)
	if len(m) == 2 {
		return m[1]
	}
	if strings.HasPrefix(uri, "Q") {
		return uri
	}
	return ""
}

func articleTitleFromURL(u string) string {
	// https://en.wikipedia.org/wiki/France → France
	const prefix = "https://en.wikipedia.org/wiki/"
	if strings.HasPrefix(u, prefix) {
		raw, err := url.PathUnescape(strings.TrimPrefix(u, prefix))
		if err != nil {
			return strings.TrimPrefix(u, prefix)
		}
		return strings.ReplaceAll(raw, "_", " ")
	}
	if i := strings.LastIndex(u, "/wiki/"); i >= 0 {
		raw, err := url.PathUnescape(u[i+6:])
		if err != nil {
			return u[i+6:]
		}
		return strings.ReplaceAll(raw, "_", " ")
	}
	return u
}

func uniqueFold(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		k := strings.ToLower(s)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
