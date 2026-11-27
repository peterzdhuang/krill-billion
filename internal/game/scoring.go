package game

import "math"

const (
	HP1v1 = 10000
	HP2v2 = 20000
)

// PlayerScore returns rarity * 50. Negative rarity is treated as 0 (invalid/empty/rejected).
func PlayerScore(rarity int) int {
	if rarity < 0 {
		return 0
	}
	return rarity * 50
}

// SideScore sums player scores for a side (1v1 = one score; 2v2 = both teammates).
func SideScore(playerScores ...int) int {
	sum := 0
	for _, s := range playerScores {
		sum += s
	}
	return sum
}

// SideState is one side's score and HP/multiplier going into a round.
type SideState struct {
	Score int
	HP    int
	Mult  float64
}

// RoundResult is the outcome of Resolve.
type RoundResult struct {
	Scores    [2]int
	Gap       int
	Damage    int
	HPAfter   [2]int
	MultAfter [2]float64
	Winner    int // 0 or 1, or -1 for tie
	Tie       bool
}

// Resolve applies one round of scoring: gap, damage, HP, and multiplier updates.
func Resolve(a, b SideState) RoundResult {
	res := RoundResult{
		Scores:  [2]int{a.Score, b.Score},
		HPAfter: [2]int{a.HP, b.HP},
		MultAfter: [2]float64{a.Mult, b.Mult},
		Winner:  -1,
		Tie:     true,
	}

	if a.Score == b.Score {
		res.Gap = 0
		res.Damage = 0
		res.MultAfter[0] = a.Mult + 0.5
		res.MultAfter[1] = b.Mult + 0.5
		return res
	}

	if a.Score > b.Score {
		res.Gap = a.Score - b.Score
		res.Damage = int(math.Round(float64(res.Gap) * a.Mult))
		res.HPAfter[1] = max(0, b.HP-res.Damage)
		res.MultAfter[0] = a.Mult + 0.5
		res.Winner = 0
		res.Tie = false
		return res
	}

	res.Gap = b.Score - a.Score
	res.Damage = int(math.Round(float64(res.Gap) * b.Mult))
	res.HPAfter[0] = max(0, a.HP-res.Damage)
	res.MultAfter[1] = b.Mult + 0.5
	res.Winner = 1
	res.Tie = false
	return res
}

// SuddenDeath compares scores only (multipliers frozen, no damage).
// winner is 0 or 1; draw is true on equal scores (winner is then -1).
func SuddenDeath(scoreA, scoreB int) (winner int, draw bool) {
	if scoreA == scoreB {
		return -1, true
	}
	if scoreA > scoreB {
		return 0, false
	}
	return 1, false
}

// CapWinner picks the higher-HP side at cap end. Equal HP requires sudden death.
func CapWinner(hpA, hpB int) (winner int, needSuddenDeath bool) {
	if hpA == hpB {
		return -1, true
	}
	if hpA > hpB {
		return 0, false
	}
	return 1, false
}
