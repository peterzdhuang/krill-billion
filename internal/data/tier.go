package data

// Tier is a rarity band label for UI display.
type Tier string

func (t Tier) String() string { return string(t) }

const (
	TierCommon    Tier = "common"
	TierUncommon  Tier = "uncommon"
	TierRare      Tier = "rare"
	TierEpic      Tier = "epic"
	TierLegendary Tier = "legendary"
)

// TierFor maps rarity 0–100 to a display tier.
//
// STUB NOTE: sibling data branch may refine thresholds; keep signature stable.
func TierFor(rarity int) Tier {
	switch {
	case rarity >= 80:
		return TierLegendary
	case rarity >= 60:
		return TierEpic
	case rarity >= 40:
		return TierRare
	case rarity >= 20:
		return TierUncommon
	default:
		return TierCommon
	}
}
