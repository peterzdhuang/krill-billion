package match

// NextStreak is the streak each side carries after this round.
// A side advances only when its answer is valid and strictly rarer than the
// other side. A tie, a loss, or an invalid answer resets that side to 0.
// An invalid opponent does not block the valid side: rarity -1 loses to any
// valid rarity.
func NextStreak(prev [2]int, a, b SideAnswer) [2]int {
	var out [2]int
	if streakAdvances(a, b) {
		out[0] = prev[0] + 1
	}
	if streakAdvances(b, a) {
		out[1] = prev[1] + 1
	}
	return out
}

func streakAdvances(self, other SideAnswer) bool {
	if !self.Valid {
		return false
	}
	if !other.Valid {
		return true
	}
	return self.Rarity > other.Rarity
}
