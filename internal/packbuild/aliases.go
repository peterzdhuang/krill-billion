package packbuild

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"golang.org/x/text/unicode/norm"
)

var honorificPrefixes = []string{
	"sir ", "dame ", "dr ", "doctor ", "mr ", "mrs ", "ms ", "miss ",
	"prof ", "professor ", "rev ", "reverend ", "hon ", "honorable ",
	"st ", "saint ",
}

var numberWords = map[string]string{
	"0": "zero", "1": "one", "2": "two", "3": "three", "4": "four",
	"5": "five", "6": "six", "7": "seven", "8": "eight", "9": "nine",
	"10": "ten", "11": "eleven", "12": "twelve", "13": "thirteen",
	"14": "fourteen", "15": "fifteen", "16": "sixteen", "17": "seventeen",
	"18": "eighteen", "19": "nineteen", "20": "twenty",
}

// ExpandAliases generates common short forms and spelling variants for a label
// plus any source aliases. The original label is never returned.
func ExpandAliases(label string, source []string) []string {
	label = strings.TrimSpace(label)
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || len([]rune(s)) < 2 {
			return
		}
		if strings.EqualFold(s, label) {
			return
		}
		k := strings.ToLower(s)
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}

	seen[strings.ToLower(label)] = struct{}{}
	for _, a := range source {
		add(a)
		for _, v := range variantsOf(a) {
			add(v)
		}
	}
	for _, v := range variantsOf(label) {
		add(v)
	}
	return out
}

func variantsOf(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	push := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" && !strings.EqualFold(v, s) {
			out = append(out, v)
		}
	}

	// Drop parenthetical disambiguators: "Foo (band)" → "Foo"
	if i := strings.Index(s, " ("); i > 0 && strings.HasSuffix(s, ")") {
		push(s[:i])
	}
	if i := strings.Index(s, " ["); i > 0 && strings.HasSuffix(s, "]") {
		push(s[:i])
	}

	// "The X" → "X" (do not synthesize "The "+label — that doubles every bank).
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "the ") && len(s) > 4 {
		push(strings.TrimSpace(s[4:]))
	}

	for _, p := range honorificPrefixes {
		if strings.HasPrefix(lower, p) {
			push(strings.TrimSpace(s[len(p):]))
		}
	}

	if mid := dropMiddleNames(s); mid != "" {
		push(mid)
	}

	if strings.Contains(s, " & ") {
		push(strings.ReplaceAll(s, " & ", " and "))
	}
	if strings.Contains(lower, " and ") {
		push(replaceFold(s, " and ", " & "))
	}

	if flat := stripDiacritics(s); flat != s {
		push(flat)
	}

	for digit, word := range numberWords {
		if strings.Contains(s, digit) {
			push(strings.ReplaceAll(s, digit, word))
		}
		if strings.Contains(lower, word) {
			push(replaceFold(s, word, digit))
		}
	}

	return out
}

func dropMiddleNames(s string) string {
	parts := strings.Fields(s)
	if len(parts) < 3 {
		return ""
	}
	for _, p := range parts[1 : len(parts)-1] {
		clean := strings.Trim(p, ".")
		if utf8.RuneCountInString(clean) > 12 {
			return ""
		}
	}
	return parts[0] + " " + parts[len(parts)-1]
}

func stripDiacritics(s string) string {
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

func replaceFold(s, old, new string) string {
	lower := strings.ToLower(s)
	old = strings.ToLower(old)
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(lower[i:], old)
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		j += i
		b.WriteString(s[i:j])
		b.WriteString(new)
		i = j + len(old)
	}
	return b.String()
}

// FoldCaseVariantEntries merges entries whose labels differ only by case.
// The first-seen label wins; later spellings become aliases.
func FoldCaseVariantEntries(entries []data.Entry) []data.Entry {
	byKey := map[string]*data.Entry{}
	order := make([]string, 0, len(entries))
	for _, e := range entries {
		key := strings.ToLower(strings.TrimSpace(e.Label))
		if key == "" {
			continue
		}
		cur := byKey[key]
		if cur == nil {
			cp := e
			cp.Aliases = append([]string{}, e.Aliases...)
			byKey[key] = &cp
			order = append(order, key)
			continue
		}
		if cur.Label != e.Label {
			cur.Aliases = append(cur.Aliases, e.Label)
		}
		cur.Aliases = append(cur.Aliases, e.Aliases...)
		if e.Rarity < cur.Rarity {
			cur.Rarity = e.Rarity
		}
		if e.ID != "" && (cur.ID == "" || e.ID < cur.ID) {
			cur.ID = e.ID
		}
	}
	out := make([]data.Entry, 0, len(order))
	for _, k := range order {
		e := byKey[k]
		e.Aliases = dedupeAliases(e.Label, e.Aliases)
		out = append(out, *e)
	}
	return out
}

func dedupeAliases(label string, aliases []string) []string {
	seen := map[string]struct{}{strings.ToLower(strings.TrimSpace(label)): {}}
	out := make([]string, 0, len(aliases))
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" || len([]rune(a)) < 2 {
			continue
		}
		k := strings.ToLower(a)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, a)
	}
	return out
}

// MinEntries is the baseline accepted-answer count for shipped prompts.
const MinEntries = 10000

// DefaultMaxEntries caps huge open banks so a 1 GiB Fly VM can load the store.
// Prompts still satisfy MinEntries; head (common) + tail (rare) are kept.
const DefaultMaxEntries = 40000

// FilterPromptsMinEntries drops prompts with fewer than min entries.
// If min <= 0, MinEntries is used.
func FilterPromptsMinEntries(prompts []data.Prompt, min int) (kept []data.Prompt, removed int) {
	if min <= 0 {
		min = MinEntries
	}
	kept = make([]data.Prompt, 0, len(prompts))
	for _, p := range prompts {
		if len(p.Entries) >= min {
			kept = append(kept, p)
		} else {
			removed++
		}
	}
	return kept, removed
}

// CapPromptEntries keeps at most max entries per prompt (common head + rare tail).
// Entries must already be sorted by rarity ascending. max <= 0 means no cap.
func CapPromptEntries(p data.Prompt, max int) data.Prompt {
	if max <= 0 || len(p.Entries) <= max {
		return p
	}
	head := max / 5
	if head < 500 {
		head = 500
	}
	if head > max/2 {
		head = max / 2
	}
	tail := max - head
	kept := make([]data.Entry, 0, max)
	kept = append(kept, p.Entries[:head]...)
	kept = append(kept, p.Entries[len(p.Entries)-tail:]...)
	// Re-sort after splicing.
	sortEntries(kept)
	p.Entries = kept
	return p
}

// CapAllPromptEntries applies CapPromptEntries to every prompt.
func CapAllPromptEntries(prompts []data.Prompt, max int) []data.Prompt {
	if max <= 0 {
		return prompts
	}
	out := make([]data.Prompt, len(prompts))
	for i, p := range prompts {
		out[i] = CapPromptEntries(p, max)
	}
	return out
}

// EnrichEntryAliases expands and folds aliases onto an entry in place.
func EnrichEntryAliases(e *data.Entry) {
	e.Aliases = ExpandAliases(e.Label, e.Aliases)
}
