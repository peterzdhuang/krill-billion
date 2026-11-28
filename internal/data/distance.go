package data

// osaScratch holds reusable DP rows for osaDistanceCutoff so Match can scan
// thousands of bucketed candidates without allocating per compare.
type osaScratch struct {
	row0, row1, row2 []int
}

func (sc *osaScratch) ensure(cols int) {
	need := cols + 1
	if cap(sc.row0) < need {
		sc.row0 = make([]int, need)
		sc.row1 = make([]int, need)
		sc.row2 = make([]int, need)
		return
	}
	sc.row0 = sc.row0[:need]
	sc.row1 = sc.row1[:need]
	sc.row2 = sc.row2[:need]
}

// osaDistance is the optimal string alignment (restricted Damerau–Levenshtein)
// distance: insertions, deletions, substitutions, and adjacent transpositions.
func osaDistance(a, b string) int {
	var sc osaScratch
	return osaDistanceCutoff(&sc, []rune(a), []rune(b), -1)
}

// osaDistanceCutoff computes OSA distance but aborts early once every cell in a
// row exceeds maxDist. Pass maxDist < 0 for no cutoff. On abort (or when the
// final distance exceeds maxDist) it returns maxDist+1.
func osaDistanceCutoff(sc *osaScratch, ar, br []rune, maxDist int) int {
	m, n := len(ar), len(br)
	if maxDist >= 0 {
		diff := m - n
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDist {
			return maxDist + 1
		}
	}
	if m == 0 {
		if maxDist >= 0 && n > maxDist {
			return maxDist + 1
		}
		return n
	}
	if n == 0 {
		if maxDist >= 0 && m > maxDist {
			return maxDist + 1
		}
		return m
	}

	sc.ensure(n)
	row0, row1, row2 := sc.row0, sc.row1, sc.row2
	for j := 0; j <= n; j++ {
		row1[j] = j
	}

	for i := 1; i <= m; i++ {
		row2[0] = i
		rowMin := row2[0]
		for j := 1; j <= n; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := row1[j] + 1
			ins := row2[j-1] + 1
			sub := row1[j-1] + cost
			best := min(del, ins, sub)
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				trans := row0[j-2] + 1
				if trans < best {
					best = trans
				}
			}
			row2[j] = best
			if best < rowMin {
				rowMin = best
			}
		}
		if maxDist >= 0 && rowMin > maxDist {
			return maxDist + 1
		}
		row0, row1, row2 = row1, row2, row0
	}
	d := row1[n]
	if maxDist >= 0 && d > maxDist {
		return maxDist + 1
	}
	return d
}

// fuzzyBudget is the max OSA distance allowed for a normalized guess of the
// given rune length: 0 for ≤3, 1 for 4–7, 2 for ≥8.
func fuzzyBudget(runeLen int) int {
	switch {
	case runeLen <= 0:
		return 0
	case runeLen <= 3:
		return 0
	case runeLen <= 7:
		return 1
	default:
		return 2
	}
}
