package packbuild

import "testing"

func TestKeepIntersection(t *testing.T) {
	cases := []struct {
		wiki, hit int
		want      bool
	}{
		{10, 6, true},  // drop 40% exactly
		{10, 5, false}, // drop 50%
		{5, 3, true},   // drop 40%
		{5, 2, false},  // drop 60%
		{3, 2, true},   // drop 33%
		{3, 1, false},  // drop 67%
		{0, 0, true},
	}
	for _, tc := range cases {
		if got := keepIntersection(tc.wiki, tc.hit); got != tc.want {
			t.Fatalf("keepIntersection(%d, %d) = %v, want %v", tc.wiki, tc.hit, got, tc.want)
		}
	}
}

func TestApplySecondSource(t *testing.T) {
	list := &ExternalList{}
	list.add([]string{"Côte d'Ivoire", "Ivory Coast", "CI"})
	list.add([]string{"Beta"})
	list.add([]string{"Gamma"})

	rows := []raw{
		{Member: Member{ID: "Q1", Label: "Ivory Coast", Aliases: []string{"Cote d'Ivoire"}}},
		{Member: Member{ID: "Q2", Label: "Beta"}},
		{Member: Member{ID: "Q3", Label: "Gamma"}},
		{Member: Member{ID: "Q4", Label: "Delta"}},
		{Member: Member{ID: "Q5", Label: "Epsilon"}},
	}
	// 3/5 match, drop is 40%, so the intersection is kept.
	kept, disagreed := applySecondSource(rows, list)
	if disagreed || len(kept) != 3 {
		t.Fatalf("40%% boundary: disagreed=%v len=%d, want false and 3", disagreed, len(kept))
	}
	var ivory raw
	for _, row := range kept {
		if row.Member.ID == "Q1" {
			ivory = row
		}
		if row.Member.ID == "Q4" || row.Member.ID == "Q5" {
			t.Fatalf("unmatched id %s kept", row.Member.ID)
		}
	}
	foundCI := false
	for _, alias := range ivory.Member.Aliases {
		if alias == "CI" {
			foundCI = true
		}
	}
	if !foundCI {
		t.Fatalf("matched row missing second-source spelling CI: %#v", ivory.Member.Aliases)
	}

	// 2/5 match, drop is 60%, so Wikidata membership stays and the note is set.
	short := &ExternalList{}
	short.add([]string{"Ivory Coast", "CI"})
	short.add([]string{"Beta"})
	kept, disagreed = applySecondSource(rows, short)
	if !disagreed || len(kept) != 5 {
		t.Fatalf("over 40%% drop: disagreed=%v len=%d, want true and 5", disagreed, len(kept))
	}
	ids := map[string]bool{}
	for _, row := range kept {
		ids[row.Member.ID] = true
		if row.Member.ID == "Q1" {
			for _, alias := range row.Member.Aliases {
				if alias == "CI" {
					foundCI = true
				}
			}
		}
	}
	for _, id := range []string{"Q1", "Q2", "Q3", "Q4", "Q5"} {
		if !ids[id] {
			t.Fatalf("disagreement dropped %s", id)
		}
	}
}

func TestNormalizeKey(t *testing.T) {
	pairs := [][2]string{
		{"Côte d'Ivoire", "Cote d'Ivoire"},
		{"Mr. Mime", "mr-mime"},
		{"The Gambia", "Gambia"},
		{"Ho-Oh", "ho oh"},
		{"Type: Null", "type-null"},
		{"Farfetch'd", "farfetchd"},
	}
	for _, pair := range pairs {
		a, b := normalizeKey(pair[0]), normalizeKey(pair[1])
		if a == "" || a != b {
			t.Fatalf("normalizeKey(%q)=%q, normalizeKey(%q)=%q", pair[0], a, pair[1], b)
		}
	}
}

func TestParseSecondSources(t *testing.T) {
	rawCountries := []byte(`[
		{"name":{"common":"Gambia","official":"Republic of the Gambia"},"altSpellings":["GM","Republic of the Gambia"]},
		{"name":{"common":"France","official":"French Republic"},"altSpellings":["FR"]}
	]`)
	list, err := parseRESTCountries(rawCountries)
	if err != nil {
		t.Fatal(err)
	}
	ok, extras := list.Match("The Gambia", nil)
	if !ok {
		t.Fatal("The Gambia did not match REST Countries")
	}
	found := false
	for _, extra := range extras {
		if extra == "GM" {
			found = true
		}
	}
	if !found {
		t.Fatalf("alt spelling GM missing: %#v", extras)
	}

	rawPoke := []byte(`{"count":1,"results":[{"name":"mr-mime"}]}`)
	poke, err := parsePokeAPI(rawPoke)
	if err != nil {
		t.Fatal(err)
	}
	ok, _ = poke.Match("Mr. Mime", nil)
	if !ok {
		t.Fatal("Mr. Mime did not match PokeAPI")
	}
}

func TestParseIntLiteral(t *testing.T) {
	n, ok := parseIntLiteral("427")
	if !ok || n != 427 {
		t.Fatalf("parseIntLiteral(427) = %d, %v", n, ok)
	}
	if _, ok := parseIntLiteral(""); ok {
		t.Fatal("empty literal should be unknown")
	}
}
