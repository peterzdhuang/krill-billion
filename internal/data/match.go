package data

import (
	"strings"
	"unicode/utf8"
)

type matchKind int

const (
	kindExact matchKind = iota
	kindPrefix
	kindFuzzy
)

type matchCandidate struct {
	entryID string
	label   string
	kind    matchKind
	dist    int // exact: 0; prefix: remaining runes; fuzzy: OSA distance
	rarity  int
}

// Match maps a raw guess onto exactly one canonical entry of the prompt.
// Empty guesses never match. Exact normalized hits win; otherwise a prefix
// (#42 autocomplete) or fuzzy Damerau–Levenshtein (OSA) search runs within the
// length-based edit budget. Ties prefer tighter kind, then smaller distance,
// then lower rarity (more common / higher commonness), then entry id.
// On accept, callers rewrite the draft to the canonical Entry.Label.
func (s *Store) Match(promptID, guess string) (entryID string, ok bool) {
	best, ok := s.bestMatch(promptID, guess)
	if !ok {
		return "", false
	}
	return best.entryID, true
}

// Suggest returns the canonical label of the best Match candidate for guess.
// Used for lenient autocomplete; anything it returns is accepted by Match.
func (s *Store) Suggest(promptID, guess string) (label string, ok bool) {
	best, ok := s.bestMatch(promptID, guess)
	if !ok {
		return "", false
	}
	return best.label, true
}

func (s *Store) bestMatch(promptID, guess string) (*matchCandidate, bool) {
	if s == nil || guess == "" {
		return nil, false
	}
	ip, found := s.prompts[promptID]
	if !found {
		return nil, false
	}

	normGuess := Normalize(guess)
	if normGuess == "" {
		return nil, false
	}

	if idx, hit := ip.exact[normGuess]; hit {
		e := ip.entries[idx]
		return &matchCandidate{
			entryID: e.ID,
			label:   e.Label,
			kind:    kindExact,
			rarity:  e.Rarity,
		}, true
	}

	guessRunes := []rune(normGuess)
	guessLen := len(guessRunes)
	var best *matchCandidate

	// Prefix path (#42): keep autocomplete while typing. Prefer the shortest
	// remaining suffix (tightest prefix), then rarity/id via betterMatch.
	if guessLen >= 2 {
		for _, e := range ip.entries {
			for _, n := range e.norms {
				if !strings.HasPrefix(n, normGuess) {
					continue
				}
				nLen := utf8.RuneCountInString(n)
				cand := matchCandidate{
					entryID: e.ID,
					label:   e.Label,
					kind:    kindPrefix,
					dist:    nLen - guessLen,
					rarity:  e.Rarity,
				}
				if betterMatch(cand, best) {
					c := cand
					best = &c
				}
			}
		}
		if best != nil && best.kind == kindPrefix {
			return best, true
		}
	}

	budget := fuzzyBudget(guessLen)
	if budget == 0 {
		return nil, false
	}

	// Fuzzy path: only scan length buckets within ±budget of the guess.
	var sc osaScratch
	for nLen := guessLen - budget; nLen <= guessLen+budget; nLen++ {
		if nLen < 0 {
			continue
		}
		bucket := ip.byLen[nLen]
		for _, bn := range bucket {
			d := osaDistanceCutoff(&sc, guessRunes, bn.runes, budget)
			if d > budget {
				continue
			}
			e := ip.entries[bn.entryIdx]
			cand := matchCandidate{
				entryID: e.ID,
				label:   e.Label,
				kind:    kindFuzzy,
				dist:    d,
				rarity:  e.Rarity,
			}
			if betterMatch(cand, best) {
				c := cand
				best = &c
			}
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// betterMatch prefers exact over prefix over fuzzy, then lower distance, then
// lower rarity (more common / higher commonness), then lexicographically
// smaller entry id for a stable final tie-break.
func betterMatch(cand matchCandidate, best *matchCandidate) bool {
	if best == nil {
		return true
	}
	if cand.kind != best.kind {
		return cand.kind < best.kind
	}
	if cand.dist != best.dist {
		return cand.dist < best.dist
	}
	if cand.rarity != best.rarity {
		return cand.rarity < best.rarity
	}
	return cand.entryID < best.entryID
}
