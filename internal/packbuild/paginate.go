package packbuild

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// QueryPaginated runs a SPARQL SELECT that binds ?item, walking pages with a
// cursor on the item IRI. The caller supplies a queryBuilder that injects
// afterIRI (empty on the first page) and limit into the query text.
//
// Prefer this for large classes; a single unbounded SELECT often times out.
func (c *WikidataClient) QueryPaginated(
	ctx context.Context,
	promptID string,
	pageSize int,
	maxPages int,
	build func(afterIRI string, limit int) string,
) ([]Member, error) {
	if pageSize <= 0 {
		pageSize = 3000
	}
	if maxPages <= 0 {
		maxPages = 40 // up to ~120k rows
	}

	var all []Member
	seen := map[string]struct{}{}
	after := ""
	for page := 0; page < maxPages; page++ {
		q := build(after, pageSize)
		cacheID := fmt.Sprintf("%s-p%d", promptID, page)
		members, err := c.QueryAllowEmpty(ctx, cacheID, q)
		if err != nil {
			if page == 0 {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "  pagination stopped at page %d: %v\n", page, err)
			break
		}
		if len(members) == 0 {
			break
		}
		added := 0
		var lastIRI string
		for _, m := range members {
			if _, ok := seen[m.ID]; ok {
				continue
			}
			seen[m.ID] = struct{}{}
			all = append(all, m)
			added++
			lastIRI = "http://www.wikidata.org/entity/" + m.ID
		}
		fmt.Fprintf(os.Stderr, "  page %d: +%d (total %d)\n", page+1, added, len(all))
		if len(members) < pageSize || lastIRI == "" {
			break
		}
		after = lastIRI
	}
	return all, nil
}

// QueryAllowEmpty is like Query but returns an empty slice (not an error) when
// SPARQL yields zero rows — used to terminate pagination cleanly.
func (c *WikidataClient) QueryAllowEmpty(ctx context.Context, promptID, sparql string) ([]Member, error) {
	members, err := c.Query(ctx, promptID, sparql)
	if err != nil && strings.Contains(err.Error(), "wikidata returned no rows") {
		return nil, nil
	}
	return members, err
}

// PaginatedSPARQL wraps a WHERE body so results are ordered by ?item and can
// resume after afterIRI. body should bind ?item (and optional ?article /
// ?sitelinks). Aliases are omitted from SPARQL for speed — ExpandAliases fills
// short forms at packbuild time.
func PaginatedSPARQL(whereBody string, afterIRI string, limit int) string {
	whereBody = strings.TrimSpace(whereBody)
	var b strings.Builder
	b.WriteString("SELECT ?item ?itemLabel ?article ?sitelinks WHERE {\n")
	b.WriteString(whereBody)
	b.WriteByte('\n')
	if afterIRI != "" {
		fmt.Fprintf(&b, "  FILTER(STR(?item) > %q)\n", afterIRI)
	}
	b.WriteString("  SERVICE wikibase:label { bd:serviceParam wikibase:language \"en\". }\n")
	b.WriteString("}\nORDER BY ?item\n")
	fmt.Fprintf(&b, "LIMIT %d\n", limit)
	return b.String()
}
