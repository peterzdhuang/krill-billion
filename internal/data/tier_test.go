package data

import "testing"

func TestTierForBandEdges(t *testing.T) {
	tests := []struct {
		rarity int
		want   Tier
	}{
		{0, TierIntern},
		{24, TierIntern},
		{25, TierJunior},
		{44, TierJunior},
		{45, TierSenior},
		{64, TierSenior},
		{65, TierStaff},
		{79, TierStaff},
		{80, TierPrincipal},
		{89, TierPrincipal},
		{90, TierDistinguished},
		{94, TierDistinguished},
		{95, TierCTO},
		{98, TierCTO},
		{99, TierJeffyB},
		{100, TierJeffyB},
	}
	for _, tt := range tests {
		if got := TierFor(tt.rarity); got != tt.want {
			t.Errorf("TierFor(%d) = %q, want %q", tt.rarity, got, tt.want)
		}
	}
}
