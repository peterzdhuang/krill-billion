package game

import "testing"

func TestWorkedExampleReplay(t *testing.T) {
	a := SideState{HP: HP1v1, Mult: 1.0}
	b := SideState{HP: HP1v1, Mult: 1.0}

	// R1: A rarity 85 vs B rarity 30
	a.Score = PlayerScore(85)
	b.Score = PlayerScore(30)
	r1 := Resolve(a, b)
	if r1.Scores != [2]int{4250, 1500} {
		t.Fatalf("R1 scores: got %v", r1.Scores)
	}
	if r1.Gap != 2750 || r1.Damage != 2750 {
		t.Fatalf("R1 gap/dmg: gap=%d dmg=%d", r1.Gap, r1.Damage)
	}
	if r1.HPAfter != [2]int{10000, 7250} {
		t.Fatalf("R1 HP: got %v", r1.HPAfter)
	}
	if r1.MultAfter != [2]float64{1.5, 1.0} {
		t.Fatalf("R1 mult: got %v", r1.MultAfter)
	}
	if r1.Winner != 0 || r1.Tie {
		t.Fatalf("R1 winner: winner=%d tie=%v", r1.Winner, r1.Tie)
	}
	a.HP, a.Mult = r1.HPAfter[0], r1.MultAfter[0]
	b.HP, b.Mult = r1.HPAfter[1], r1.MultAfter[1]

	// R2: A rarity 60 vs B rarity 100
	a.Score = PlayerScore(60)
	b.Score = PlayerScore(100)
	r2 := Resolve(a, b)
	if r2.Scores != [2]int{3000, 5000} {
		t.Fatalf("R2 scores: got %v", r2.Scores)
	}
	if r2.Gap != 2000 || r2.Damage != 2000 {
		t.Fatalf("R2 gap/dmg: gap=%d dmg=%d", r2.Gap, r2.Damage)
	}
	if r2.HPAfter != [2]int{8000, 7250} {
		t.Fatalf("R2 HP: got %v", r2.HPAfter)
	}
	if r2.MultAfter != [2]float64{1.5, 1.5} {
		t.Fatalf("R2 mult: got %v", r2.MultAfter)
	}
	if r2.Winner != 1 || r2.Tie {
		t.Fatalf("R2 winner: winner=%d tie=%v", r2.Winner, r2.Tie)
	}
	a.HP, a.Mult = r2.HPAfter[0], r2.MultAfter[0]
	b.HP, b.Mult = r2.HPAfter[1], r2.MultAfter[1]

	// R3: A rarity 100 vs B invalid
	a.Score = PlayerScore(100)
	b.Score = PlayerScore(-1) // invalid
	r3 := Resolve(a, b)
	if r3.Scores != [2]int{5000, 0} {
		t.Fatalf("R3 scores: got %v", r3.Scores)
	}
	if r3.Gap != 5000 || r3.Damage != 7500 {
		t.Fatalf("R3 gap/dmg: gap=%d dmg=%d", r3.Gap, r3.Damage)
	}
	if r3.HPAfter != [2]int{8000, 0} {
		t.Fatalf("R3 HP: got %v", r3.HPAfter)
	}
	if r3.MultAfter != [2]float64{2.0, 1.5} {
		t.Fatalf("R3 mult: got %v", r3.MultAfter)
	}
	if r3.Winner != 0 || r3.Tie {
		t.Fatalf("R3 winner: winner=%d tie=%v", r3.Winner, r3.Tie)
	}
}

func TestTieBothGainMultNoDamage(t *testing.T) {
	a := SideState{Score: PlayerScore(40), HP: HP1v1, Mult: 1.0}
	b := SideState{Score: PlayerScore(40), HP: HP1v1, Mult: 1.5}
	r := Resolve(a, b)
	if r.Gap != 0 || r.Damage != 0 {
		t.Fatalf("tie gap/dmg: gap=%d dmg=%d", r.Gap, r.Damage)
	}
	if r.HPAfter != [2]int{HP1v1, HP1v1} {
		t.Fatalf("tie HP changed: %v", r.HPAfter)
	}
	if r.MultAfter != [2]float64{1.5, 2.0} {
		t.Fatalf("tie mult: got %v", r.MultAfter)
	}
	if !r.Tie || r.Winner != -1 {
		t.Fatalf("tie flags: winner=%d tie=%v", r.Winner, r.Tie)
	}
}

func TestTwoVTwoSideScoreDamage(t *testing.T) {
	scoreA := SideScore(PlayerScore(85), PlayerScore(60))
	scoreB := SideScore(PlayerScore(30), PlayerScore(30))
	if scoreA != 7250 || scoreB != 3000 {
		t.Fatalf("side scores: A=%d B=%d", scoreA, scoreB)
	}
	r := Resolve(
		SideState{Score: scoreA, HP: HP2v2, Mult: 1.0},
		SideState{Score: scoreB, HP: HP2v2, Mult: 1.0},
	)
	if r.Gap != 4250 || r.Damage != 4250 {
		t.Fatalf("2v2 gap/dmg: gap=%d dmg=%d", r.Gap, r.Damage)
	}
	if r.HPAfter != [2]int{20000, 15750} {
		t.Fatalf("2v2 HP: got %v", r.HPAfter)
	}
	if r.Winner != 0 || r.Tie {
		t.Fatalf("2v2 winner: winner=%d tie=%v", r.Winner, r.Tie)
	}
}
