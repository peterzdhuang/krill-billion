package lobby

import "github.com/peterzdhuang/krill-billion/internal/match"

// Mode is lobby capacity / seating layout.
type Mode string

const (
	Mode1v1 Mode = "1v1"
	Mode2v2 Mode = "2v2"
)

func SeatsFor(mode Mode) []string {
	switch mode {
	case Mode2v2:
		return []string{match.SeatA1, match.SeatA2, match.SeatB1, match.SeatB2}
	default:
		return []string{match.SeatA1, match.SeatB1}
	}
}

func SideOf(seat string) int {
	switch seat {
	case match.SeatA1, match.SeatA2:
		return 0
	default:
		return 1
	}
}

func ValidSeat(mode Mode, seat string) bool {
	for _, s := range SeatsFor(mode) {
		if s == seat {
			return true
		}
	}
	return false
}
