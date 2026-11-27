package data

import "unicode/utf8"

type matchCandidate struct {
	entryID string
	dist    int
	rarity  int
}

// Match maps a raw guess onto exactly one canonical entry of the prompt.
// Empty guesses never match. Exact normalized hits win; otherwise a fuzzy
// Damerau–Levenshtein (OSA) search runs within a length-based edit budget.
// Ties prefer smaller distance, then lower rarity, then lower entry id.
func (s *Store) Match(promptID, guess string) (entryID string, ok bool) {
	if s == nil || guess == "" {
		return "", false
	}
	ip, found := s.prompts[promptID]
	if !found {
		return "", false
	}

	normGuess := Normalize(guess)
	if normGuess == "" {
		return "", false
	}

	if id, hit := ip.exact[normGuess]; hit {
		return id, true
	}

	budget := fuzzyBudget(utf8.RuneCountInString(normGuess))
	if budget == 0 {
		return "", false
	}

	guessLen := utf8.RuneCountInString(normGuess)
	var best *matchCandidate

	for _, e := range ip.entries {
		for _, n := range e.norms {
			nLen := utf8.RuneCountInString(n)
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
			cand := matchCandidate{entryID: e.ID, dist: d, rarity: e.Rarity}
			if betterMatch(cand, best) {
				c := cand
				best = &c
			}
		}
	}
	if best == nil {
		return "", false
	}
	return best.entryID, true
}

// fuzzyCloseEnough keeps a typo of a real entry and drops unrelated strings.
// Forms shorter than 4 characters stay exact-only (the exact map already
// returned those). Distance must also fit in a third of the longer string.
func fuzzyCloseEnough(guessLen, normLen, dist int) bool {
	if dist == 0 || normLen < 4 {
		return false
	}
	longer := guessLen
	if normLen > longer {
		longer = normLen
	}
	return dist <= longer/3
}

func betterMatch(cand matchCandidate, best *matchCandidate) bool {
	if best == nil {
		return true
	}
	if cand.dist != best.dist {
		return cand.dist < best.dist
	}
	if cand.rarity != best.rarity {
		return cand.rarity < best.rarity
	}
	return cand.entryID < best.entryID
}
