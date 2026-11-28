package data

import (
	"path/filepath"
	"runtime"
	"testing"
	"unicode/utf8"
)

func testStore(t *testing.T, prompts ...Prompt) *Store {
	t.Helper()
	s, err := NewFromPacks(Pack{PackID: "test", Prompts: prompts})
	if err != nil {
		t.Fatalf("NewFromPacks: %v", err)
	}
	return s
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Thé Citiés", "city"},
		{"  The   Cat  ", "cat"},
		{"an apple", "apple"},
		{"A Brand!", "brand"},
		{"café", "cafe"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
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
			{ID: "banana", Label: "banana", Rarity: 25},
		},
	})

	tests := []struct {
		name   string
		guess  string
		wantID string
		wantOK bool
	}{
		{
			name:   "accent article plural fold exact",
			guess:  "Thé Citiés",
			wantID: "city",
			wantOK: true,
		},
		{
			name:   "adjacent transposition is 1 edit",
			guess:  "tset",
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
			name:   "3-letter one-edit not over-matched",
			guess:  "cot",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "short unrelated rejected",
			guess:  "zz",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "4-7 one substitution",
			guess:  "banama",
			wantID: "banana",
			wantOK: true,
		},
		{
			name:   "4-7 two edits rejected",
			guess:  "banxxa",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "long label one insertion",
			guess:  "Afghanistaan",
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "long label two substitutions",
			guess:  "Afthanistan",
			wantID: "afghan",
			wantOK: true,
		},
		{
			name:   "long label three edits rejected",
			guess:  "Xxthanistan",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "ambiguity tie-break picks lower rarity (more common)",
			guess:  "xart",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, ok := s.Match("p1", tt.guess)
			if ok != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("Match(%q) = (%q, %v), want (%q, %v)", tt.guess, gotID, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestMatchCanonicalLabel(t *testing.T) {
	s := testStore(t, Prompt{
		ID:   "p1",
		Text: "birds",
		Entries: []Entry{
			{ID: "eagle", Label: "Eagle", Rarity: 40},
		},
	})
	id, ok := s.Match("p1", "eagel")
	if !ok || id != "eagle" {
		t.Fatalf("Match(eagel) = (%q, %v), want eagle", id, ok)
	}
	ent, found := s.Entry("p1", id)
	if !found || ent.Label != "Eagle" {
		t.Fatalf("Entry = %+v, want Label Eagle", ent)
	}
}

func TestFuzzyBudget(t *testing.T) {
	cases := map[int]int{0: 0, 1: 0, 3: 0, 4: 1, 7: 1, 8: 2, 20: 2}
	for len, want := range cases {
		if got := fuzzyBudget(len); got != want {
			t.Fatalf("fuzzyBudget(%d) = %d, want %d", len, got, want)
		}
	}
}

func TestOsaDistanceCutoff(t *testing.T) {
	if d := osaDistance("test", "tset"); d != 1 {
		t.Fatalf("transposition distance = %d, want 1", d)
	}
	var sc osaScratch
	if d := osaDistanceCutoff(&sc, []rune("abcdef"), []rune("abXdef"), 1); d != 1 {
		t.Fatalf("one sub within cutoff = %d, want 1", d)
	}
	if d := osaDistanceCutoff(&sc, []rune("abcdef"), []rune("abXXef"), 1); d != 2 {
		t.Fatalf("two subs over cutoff = %d, want 2 (max+1)", d)
	}
	if d := osaDistanceCutoff(&sc, []rune("short"), []rune("muchlonger"), 2); d != 3 {
		t.Fatalf("length skip = %d, want 3", d)
	}
}

func packsDir(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "packs")
}

func loadShippedStore(t testing.TB) *Store {
	t.Helper()
	s, err := Load(packsDir(t))
	if err != nil {
		t.Fatalf("Load packs: %v", err)
	}
	return s
}

func largestPrompt(s *Store) (id string, entries int) {
	for _, pid := range s.order {
		ip := s.prompts[pid]
		n := len(ip.entries)
		if n > entries {
			entries = n
			id = pid
		}
	}
	return id, entries
}

func TestLargestPromptShipped(t *testing.T) {
	s := loadShippedStore(t)
	id, n := largestPrompt(s)
	if n == 0 || id == "" {
		t.Skip("no shipped packs present (rebuild with make packbuild-deep)")
	}
	t.Logf("largest prompt %q has %d entries (%d prompts, %d packs)", id, n, s.PromptCount(), s.PackCount())
}

func BenchmarkMatchMissLargest(b *testing.B) {
	s := loadShippedStore(b)
	id, n := largestPrompt(s)
	if n == 0 {
		b.Skip("empty store")
	}
	// Unrelated medium-length miss so the fuzzy path scans ±budget buckets.
	guess := "zzzxqqwvutsr"
	if utf8.RuneCountInString(Normalize(guess)) < 8 {
		b.Fatal("guess should use budget 2")
	}
	b.ReportMetric(float64(n), "entries")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := s.Match(id, guess); ok {
			b.Fatal("expected miss")
		}
	}
}
