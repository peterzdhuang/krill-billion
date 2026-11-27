package data

// osaDistance is the optimal string alignment (restricted Damerau–Levenshtein)
// distance: insertions, deletions, substitutions, and adjacent transpositions.
func osaDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	m, n := len(ar), len(br)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	// dp[i][j] = distance between ar[:i] and br[:j]
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i
	}
	for j := 1; j <= n; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := dp[i-1][j] + 1
			ins := dp[i][j-1] + 1
			sub := dp[i-1][j-1] + cost
			dp[i][j] = min(del, ins, sub)
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				trans := dp[i-2][j-2] + 1
				if trans < dp[i][j] {
					dp[i][j] = trans
				}
			}
		}
	}
	return dp[m][n]
}

func fuzzyBudget(runeLen int) int {
	switch {
	case runeLen <= 0:
		return 0
	case runeLen <= 3:
		return 0
	case runeLen <= 6:
		return 1
	case runeLen <= 10:
		return 2
	default:
		return 3
	}
}
