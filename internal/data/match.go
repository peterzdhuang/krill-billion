package data

import (
	"strings"
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
// or fuzzy Damerau–Levenshtein (OSA) search runs within the length-based edit
// budget. Ties prefer tighter kind, then smaller distance, then lower rarity
// (more common / higher commonness), then entry id.
// Callers should keep the raw guess for display and use Entry.Label as the
// canonical matched label after a hit.
func (s *Store) Match(promptID, guess string) (entryID string, ok bool) {
	best, ok := s.bestMatch(promptID, guess)
	if !ok {
		return "", false
	}
	return best.entryID, true
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

	// Prefix path: scan length buckets ≥ guessLen for norms with this prefix.
	if guessLen >= 2 {
		for nLen, bucket := range ip.byLen {
			if nLen < guessLen {
				continue
			}
			for _, bn := range bucket {
				if !strings.HasPrefix(bn.norm, normGuess) {
					continue
				}
				e := ip.entries[bn.entryIdx]
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
			d := osaDistanceCutoff(&sc, guessRunes, []rune(bn.norm), budget)
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

// Suggest returns the canonical label of the best Match candidate for guess.
func (s *Store) Suggest(promptID, guess string) (label string, ok bool) {
	best, ok := s.bestMatch(promptID, guess)
	if !ok {
		return "", false
	}
	return best.label, true
}
