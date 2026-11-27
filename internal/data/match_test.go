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
			{ID: "afghan", Label: "Afghanistan", Rarity: 40},
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
			name:   "3-letter requires exact (no fuzzy)",
			guess:  "cot",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "11+ chars with 3 typos matches",
			guess:  "Xyghanistaa", // Afghanistan with 3 substitutions
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "ambiguity tie-break picks lower rarity",
			guess:  "xart", // distance 1 to both cart (r50) and bart (r20)
			wantID: "bart",
			wantOK: true,
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
