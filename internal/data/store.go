package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// Store holds loaded packs and supports prompt selection / guess matching.
//
// STUB: minimal implementation so lobby/match builds alone. Sibling #2/#4
// should replace Load/Match/RandomPrompts with the real pack index.
type Store struct {
	mu      sync.RWMutex
	packs   []Pack
	prompts []Prompt
	byID    map[string]Prompt
	index   map[string]map[string]string // promptID -> normalized guess -> entryID
}

// Load reads all *.json packs from dir.
func Load(dir string) (*Store, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read packs dir: %w", err)
	}
	s := &Store{
		byID:  make(map[string]Prompt),
		index: make(map[string]map[string]string),
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var pack Pack
		if err := json.Unmarshal(b, &pack); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		s.packs = append(s.packs, pack)
		for _, p := range pack.Prompts {
			s.prompts = append(s.prompts, p)
			s.byID[p.ID] = p
			norm := make(map[string]string)
			for _, ent := range p.Entries {
				norm[normalize(ent.Label)] = ent.ID
				for _, a := range ent.Aliases {
					norm[normalize(a)] = ent.ID
				}
			}
			s.index[p.ID] = norm
		}
	}
	if len(s.prompts) == 0 {
		return nil, fmt.Errorf("no prompts loaded from %s", dir)
	}
	return s, nil
}

func (s *Store) PromptCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.prompts)
}

func (s *Store) PackCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.packs)
}

// RandomPrompts returns up to n prompts not in exclude. If the pack is too
// small, prompts may be reused with synthetic IDs for long matches.
func (s *Store) RandomPrompts(n int, exclude []string) []Prompt {
	s.mu.Lock()
	defer s.mu.Unlock()
	ex := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		ex[id] = struct{}{}
	}
	var pool []Prompt
	for _, p := range s.prompts {
		if _, skip := ex[p.ID]; !skip {
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		pool = append([]Prompt(nil), s.prompts...)
	}
	shuffled := append([]Prompt(nil), pool...)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	out := make([]Prompt, 0, n)
	baseIdx := 0
	for len(out) < n {
		src := shuffled[baseIdx%len(shuffled)]
		baseIdx++
		cp := src
		if _, used := ex[cp.ID]; used {
			cp.ID = fmt.Sprintf("%s~%d", src.ID, len(out)+len(exclude))
			s.byID[cp.ID] = cp
			s.index[cp.ID] = s.index[src.ID]
		}
		ex[cp.ID] = struct{}{}
		out = append(out, cp)
	}
	return out
}

// Match finds an entry for guess under promptID.
func (s *Store) Match(promptID, guess string) (entryID string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.index[promptID]
	if m == nil {
		if i := strings.LastIndex(promptID, "~"); i > 0 {
			m = s.index[promptID[:i]]
		}
	}
	if m == nil {
		return "", false
	}
	id, ok := m[normalize(guess)]
	return id, ok
}

// Entry returns an entry by id under a prompt.
func (s *Store) Entry(promptID, entryID string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[promptID]
	if !ok {
		if i := strings.LastIndex(promptID, "~"); i > 0 {
			p, ok = s.byID[promptID[:i]]
		}
	}
	if !ok {
		return Entry{}, false
	}
	for _, e := range p.Entries {
		if e.ID == entryID {
			return e, true
		}
	}
	return Entry{}, false
}

func normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
