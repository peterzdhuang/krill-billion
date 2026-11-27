package data

import "testing"

func testStore(t *testing.T, prompts ...Prompt) *Store {
	t.Helper()
	s := &Store{prompts: make(map[string]*indexedPrompt)}
	for _, p := range prompts {
		ip, err := indexPrompt(p)
		if err != nil {
			t.Fatalf("indexPrompt(%q): %v", p.ID, err)
		}
		s.prompts[p.ID] = ip
		s.order = append(s.order, p.ID)
	}
	return s
}

func TestMatch(t *testing.T) {
	s := testStore(t, Prompt{
		ID:   "p1",
		Text: "test",
		Entries: []Entry{
			{ID: "city", Label: "City", Rarity: 10},
			{ID: "test", Label: "test", Rarity: 15},
			{ID: "cat", Label: "cat", Rarity: 30},
			{ID: "afghan", Label: "Afghanistan", Rarity: 40, Aliases: []string{"AF"}},
			{ID: "cart", Label: "cart", Rarity: 50},
			{ID: "bart", Label: "bart", Rarity: 20},
		},
	})

	tests := []struct {
		name      string
		guess     string
		wantID    string
		wantOK    bool
		checkNorm string // optional: assert Normalize(guess)
	}{
		{
			name:      "accent article plural fold",
			guess:     "Thé Citiés",
			wantID:    "city",
			wantOK:    true,
			checkNorm: "city",
		},
		{
			name:   "teh-style transposition is 1 edit",
			guess:  "tset", // adjacent transposition of "test"
			wantID: "test",
			wantOK: true,
		},
		{
			name:   "3-letter exact match",
			guess:  "cat",
			wantID: "cat",
			wantOK: true,
		},
		{
			name:   "3-letter one-edit fuzzy",
			guess:  "cot",
			wantID: "cat",
			wantOK: true,
		},
		{
			name:   "11+ chars with typos matches",
			guess:  "Xyghanistaa", // Afghanistan with substitutions
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "ambiguity tie-break picks lower rarity",
			guess:  "xart", // distance 1 to both cart (r50) and bart (r20)
			wantID: "bart",
			wantOK: true,
		},
		{
			name:   "unrelated xyz is rejected",
			guess:  "xyz",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "one-edit typo of a real label",
			guess:  "Afghanistaan",
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "prefix of label",
			guess:  "afghanis",
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "alias exact",
			guess:  "af",
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "short unrelated rejected",
			guess:  "zz",
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkNorm != "" {
				if got := Normalize(tt.guess); got != tt.checkNorm {
					t.Fatalf("Normalize(%q) = %q, want %q", tt.guess, got, tt.checkNorm)
				}
			}
			gotID, ok := s.Match("p1", tt.guess)
			if ok != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("Match(%q) = (%q, %v), want (%q, %v)", tt.guess, gotID, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestSuggest(t *testing.T) {
	s := testStore(t, Prompt{
		ID:   "p1",
		Text: "test",
		Entries: []Entry{
			{ID: "afghan", Label: "Afghanistan", Rarity: 40, Aliases: []string{"AF"}},
			{ID: "test", Label: "test", Rarity: 15},
		},
	})

	label, ok := s.Suggest("p1", "afghanis")
	if !ok || label != "Afghanistan" {
		t.Fatalf("Suggest(prefix) = (%q, %v), want Afghanistan", label, ok)
	}
	label, ok = s.Suggest("p1", "tset")
	if !ok || label != "test" {
		t.Fatalf("Suggest(typo) = (%q, %v), want test", label, ok)
	}
	label, ok = s.Suggest("p1", "af")
	if !ok || label != "Afghanistan" {
		t.Fatalf("Suggest(alias) = (%q, %v), want Afghanistan", label, ok)
	}
	if _, ok := s.Suggest("p1", "zzz"); ok {
		t.Fatal("Suggest(unrelated) should fail")
	}
	// Accepted suggestion must always Match.
	for _, q := range []string{"afghanis", "tset", "af", "Afghanistaan"} {
		sug, ok := s.Suggest("p1", q)
		if !ok {
			t.Fatalf("Suggest(%q) failed", q)
		}
		id, mok := s.Match("p1", sug)
		if !mok {
			t.Fatalf("Match(Suggest(%q)=%q) failed", q, sug)
		}
		id2, mok2 := s.Match("p1", q)
		if !mok2 || id2 != id {
			t.Fatalf("Match(%q) = (%q, %v), want same entry as suggestion %q (%q)", q, id2, mok2, sug, id)
		}
	}
}
