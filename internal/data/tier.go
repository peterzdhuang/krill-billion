package data

// Tier is a CS career band mapped from an entry's rarity score.
type Tier string

const (
	TierIntern        Tier = "Intern"
	TierJunior        Tier = "Junior"
	TierSenior        Tier = "Senior"
	TierStaff         Tier = "Staff"
	TierPrincipal     Tier = "Principal"
	TierDistinguished Tier = "Distinguished"
	TierCTO           Tier = "CTO"
	TierJeffyB        Tier = "Jeffy B"
)

// TierFor returns the career tier for a rarity score in 0–100 (inclusive bands).
func TierFor(rarity int) Tier {
	switch {
	case rarity <= 24:
		return TierIntern
	case rarity <= 44:
		return TierJunior
	case rarity <= 64:
		return TierSenior
	case rarity <= 79:
		return TierStaff
	case rarity <= 89:
		return TierPrincipal
	case rarity <= 94:
		return TierDistinguished
	case rarity <= 98:
		return TierCTO
	default:
		return TierJeffyB
	}
}
