package packbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const sitelinkChunk = 80

// SitelinkCounts returns the Wikidata sitelink count for each id.
// Counts already present on a cached member (SitelinksKnown) are not requested.
// Results are cached under .packbuild-cache/sitelinks so reruns stay local.
func (c *WikidataClient) SitelinkCounts(ctx context.Context, ids []string) (map[string]int, error) {
	if err := c.loadSitelinks(); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(ids))
	need := make([]string, 0)
	for _, id := range ids {
		if !qidLabel.MatchString(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := c.sitelinks[id]; ok {
			continue
		}
		need = append(need, id)
	}
	sort.Strings(need)
	if len(need) > 0 && c.Offline {
		return nil, fmt.Errorf("offline: sitelink cache missing %d ids", len(need))
	}
	if len(need) > 0 {
		fmt.Fprintf(os.Stderr, "  sitelinks: %d cached, fetching %d\n", len(seen)-len(need), len(need))
	}
	for i := 0; i < len(need); i += sitelinkChunk {
		end := i + sitelinkChunk
		if end > len(need) {
			end = len(need)
		}
		part := need[i:end]
		fmt.Fprintf(os.Stderr, "  sitelinks %d-%d of %d\n", i+1, end, len(need))
		got, err := c.fetchSitelinkChunk(ctx, part)
		if err != nil {
			return nil, err
		}
		for _, id := range part {
			n := got[id]
			c.sitelinks[id] = n
		}
		if err := c.saveSitelinks(); err != nil {
			return nil, err
		}
	}
	out := make(map[string]int, len(seen))
	for id := range seen {
		out[id] = c.sitelinks[id]
	}
	return out, nil
}

func (c *WikidataClient) fetchSitelinkChunk(ctx context.Context, ids []string) (map[string]int, error) {
	parsed, err := c.fetchBindingsRetry(ctx, sitelinkQuery(ids))
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(ids))
	for _, b := range parsed.Results.Bindings {
		id := extractQID(b["item"].Value)
		if id == "" {
			continue
		}
		n, ok := parseIntLiteral(b["sitelinks"].Value)
		if !ok {
			continue
		}
		out[id] = n
	}
	return out, nil
}

func sitelinkQuery(ids []string) string {
	var b strings.Builder
	b.WriteString("SELECT ?item ?sitelinks WHERE {\n  VALUES ?item {")
	for _, id := range ids {
		if !qidLabel.MatchString(id) {
			continue
		}
		b.WriteString(" wd:")
		b.WriteString(id)
	}
	b.WriteString(" }\n  ?item wikibase:sitelinks ?sitelinks .\n}\n")
	return b.String()
}

func (c *WikidataClient) sitelinkCachePath() string {
	dir := c.CacheDir
	if dir == "" {
		dir = ".packbuild-cache"
	}
	return filepath.Join(dir, "sitelinks", "counts.json")
}

func (c *WikidataClient) loadSitelinks() error {
	if c.sitelinks != nil {
		return nil
	}
	c.sitelinks = map[string]int{}
	b, err := os.ReadFile(c.sitelinkCachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := json.Unmarshal(b, &c.sitelinks); err != nil {
		c.sitelinks = map[string]int{}
	}
	return nil
}

func (c *WikidataClient) saveSitelinks() error {
	path := c.sitelinkCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c.sitelinks, "", "  ")
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
