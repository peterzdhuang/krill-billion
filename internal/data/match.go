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
// or fuzzy Damerau–Levenshtein (OSA) search runs within a length-based budget.
// Ties prefer tighter kind, then smaller distance, then lower rarity, then id.
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

	if id, hit := ip.exact[normGuess]; hit {
		e, ok := entryByID(ip, id)
		if !ok {
			return nil, false
		}
		return &matchCandidate{
			entryID: id,
			label:   e.Label,
			kind:    kindExact,
			rarity:  e.Rarity,
		}, true
	}

	budget := fuzzyBudget(utf8.RuneCountInString(normGuess))
	guessLen := utf8.RuneCountInString(normGuess)
	var best *matchCandidate

	for _, e := range ip.entries {
		for _, n := range e.norms {
			nLen := utf8.RuneCountInString(n)
			cand := matchCandidate{entryID: e.ID, label: e.Label, rarity: e.Rarity}

			if guessLen >= 2 && strings.HasPrefix(n, normGuess) {
				cand.kind = kindPrefix
				cand.dist = nLen - guessLen
				if betterMatch(cand, best) {
					c := cand
					best = &c
				}
				continue
			}

			if budget == 0 {
				continue
			}
			diff := guessLen - nLen
			if diff < 0 {
				diff = -diff
			}
			if diff > budget {
				continue
			}
			d := osaDistance(normGuess, n)
			if d > budget || !fuzzyCloseEnough(guessLen, nLen, d) {
				continue
			}
			cand.kind = kindFuzzy
			cand.dist = d
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

func entryByID(ip *indexedPrompt, id string) (Entry, bool) {
	for _, e := range ip.entries {
		if e.ID == id {
			return e.Entry, true
		}
	}
	return Entry{}, false
}

// fuzzyCloseEnough keeps a typo of a real entry and drops unrelated strings.
// Forms shorter than 3 characters stay exact/prefix-only. Distance must fit
// in half the longer string (generous autocomplete threshold).
func fuzzyCloseEnough(guessLen, normLen, dist int) bool {
	if dist == 0 || normLen < 3 {
		return false
	}
	longer := guessLen
	if normLen > longer {
		longer = normLen
	}
	limit := longer / 2
	if limit < 1 {
		limit = 1
	}
	return dist <= limit
}

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
