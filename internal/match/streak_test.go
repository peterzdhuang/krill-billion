package match

import "testing"

func TestNextStreak(t *testing.T) {
	valid := func(rarity int) SideAnswer {
		return SideAnswer{Valid: true, Rarity: rarity}
	}
	invalid := SideAnswer{Valid: false, Rarity: -1}

	tests := []struct {
		name string
		prev [2]int
		a, b SideAnswer
		want [2]int
	}{
		{"rarer side continues", [2]int{2, 1}, valid(80), valid(10), [2]int{3, 0}},
		{"tie resets both", [2]int{4, 4}, valid(50), valid(50), [2]int{0, 0}},
		{"invalid does not advance", [2]int{3, 2}, invalid, valid(90), [2]int{0, 3}},
		{"both invalid reset", [2]int{1, 1}, invalid, invalid, [2]int{0, 0}},
		{"loss resets and winner continues", [2]int{0, 5}, valid(10), valid(70), [2]int{0, 6}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextStreak(tc.prev, tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("NextStreak() = %v, want %v", got, tc.want)
			}
		})
	}
}
