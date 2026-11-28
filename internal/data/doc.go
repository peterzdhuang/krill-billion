// Package data loads question packs and matches player guesses.
//
// Matching (per prompt, never hits the DB):
//  1. Normalize: lowercase, strip accents/diacritics and punctuation, collapse
//     whitespace, drop a leading "the"/"a"/"an", then a simple plural fold.
//  2. Exact hit on the precomputed normalized label/alias → entry map.
//  3. Else prefix match (#42 autocomplete) when the guess has ≥2 runes.
//  4. Else fuzzy optimal string alignment (Damerau–Levenshtein) against length
//     buckets within the edit budget: 0 for ≤3 runes, 1 for 4–7, 2 for ≥8.
//     Distance ties prefer lower rarity (more common / higher commonness),
//     then lexicographically smaller entry id.
package data
