package data

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalize prepares a guess, label, or alias for matching:
// lowercase, strip accents, strip punctuation, strip leading articles,
// collapse whitespace, and apply a simple plural fold.
func Normalize(s string) string {
	s = strings.ToLower(s)
	s = stripAccents(s)
	s = stripPunctuation(s)
	s = collapseWhitespace(s)
	s = stripLeadingArticle(s)
	s = collapseWhitespace(s)
	s = pluralFold(s)
	return s
}

func stripAccents(s string) string {
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

func stripPunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func stripLeadingArticle(s string) string {
	for _, article := range []string{"a ", "an ", "the "} {
		if strings.HasPrefix(s, article) {
			return strings.TrimSpace(s[len(article):])
		}
	}
	return s
}

func pluralFold(s string) string {
	if s == "" {
		return s
	}
	switch {
	case strings.HasSuffix(s, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "es") && len(s) > 2:
		return s[:len(s)-2]
	case strings.HasSuffix(s, "s") && len(s) > 1:
		return s[:len(s)-1]
	default:
		return s
	}
}
