package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// indexedEntry holds an entry plus precomputed normalized forms for matching.
type indexedEntry struct {
	Entry
	norms []string // normalized label + aliases (deduped)
}

// bucketNorm is one precomputed normalized form, keyed by rune length for
// fuzzy scans. runes are stored so Match avoids re-decoding on every compare.
type bucketNorm struct {
	runes    []rune
	entryIdx int
}

type indexedPrompt struct {
	Prompt
	entries []indexedEntry
	// exact maps normalized form -> entry index (first wins at load).
	exact map[string]int
	// byLen buckets normalized forms by rune length for fuzzy matching.
	byLen map[int][]bucketNorm
}

// Store holds all loaded packs in memory for matching and prompt selection.
type Store struct {
	packs   []Pack
	prompts map[string]*indexedPrompt // prompt id -> prompt
	order   []string                  // prompt ids in load order
}

// Load reads every *.json pack under dir, validates them, and builds a Store.
// On any invalid pack it returns a clear error suitable for aborting boot.
func Load(dir string) (*Store, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read pack dir %q: %w", dir, err)
	}

	var jsonFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		jsonFiles = append(jsonFiles, e.Name())
	}
	sort.Strings(jsonFiles)

	packs := make([]Pack, 0, len(jsonFiles))
	for _, name := range jsonFiles {
		path := filepath.Join(dir, name)
		pack, err := loadPackFile(path)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return NewFromPacks(packs...)
}

// NewFromPacks validates packs and builds a Store. An empty list is an empty store.
func NewFromPacks(packs ...Pack) (*Store, error) {
	s := &Store{
		prompts: make(map[string]*indexedPrompt),
	}
	promptSeen := make(map[string]string) // prompt id -> pack id
	for _, pack := range packs {
		if pack.PackID == "" {
			return nil, fmt.Errorf("pack missing pack_id")
		}
		if err := s.indexPack(pack, promptSeen); err != nil {
			return nil, fmt.Errorf("pack %q: %w", pack.PackID, err)
		}
		s.packs = append(s.packs, pack)
	}
	return s, nil
}

func loadPackFile(path string) (Pack, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, fmt.Errorf("read pack %q: %w", path, err)
	}
	var pack Pack
	if err := json.Unmarshal(b, &pack); err != nil {
		return Pack{}, fmt.Errorf("parse pack %q: %w", path, err)
	}
	if pack.PackID == "" {
		return Pack{}, fmt.Errorf("pack %q: missing pack_id", path)
	}
	return pack, nil
}

func (s *Store) indexPack(pack Pack, promptSeen map[string]string) error {
	for _, p := range pack.Prompts {
		if p.ID == "" {
			return fmt.Errorf("prompt with empty id")
		}
		if prev, ok := promptSeen[p.ID]; ok {
			return fmt.Errorf("duplicate prompt id %q (also in pack %q)", p.ID, prev)
		}
		if p.Text == "" {
			return fmt.Errorf("prompt %q: empty text", p.ID)
		}
		ip, err := indexPrompt(p)
		if err != nil {
			return err
		}
		promptSeen[p.ID] = pack.PackID
		s.prompts[p.ID] = ip
		s.order = append(s.order, p.ID)
	}
	return nil
}

func indexPrompt(p Prompt) (*indexedPrompt, error) {
	ip := &indexedPrompt{
		Prompt:  p,
		entries: make([]indexedEntry, 0, len(p.Entries)),
		exact:   make(map[string]int),
		byLen:   make(map[int][]bucketNorm),
	}
	entrySeen := make(map[string]struct{})
	for _, e := range p.Entries {
		if e.ID == "" {
			return nil, fmt.Errorf("prompt %q: entry with empty id", p.ID)
		}
		if _, ok := entrySeen[e.ID]; ok {
			return nil, fmt.Errorf("prompt %q: duplicate entry id %q", p.ID, e.ID)
		}
		entrySeen[e.ID] = struct{}{}
		if strings.TrimSpace(e.Label) == "" {
			return nil, fmt.Errorf("prompt %q: entry %q has empty label", p.ID, e.ID)
		}
		if e.Rarity < 0 || e.Rarity > 100 {
			return nil, fmt.Errorf("prompt %q: entry %q rarity %d out of range 0-100", p.ID, e.ID, e.Rarity)
		}

		norms := uniqueNorms(e.Label, e.Aliases)
		entryIdx := len(ip.entries)
		ie := indexedEntry{Entry: e, norms: norms}
		ip.entries = append(ip.entries, ie)
		for _, n := range norms {
			if n == "" {
				continue
			}
			if _, exists := ip.exact[n]; !exists {
				ip.exact[n] = entryIdx
			}
			runes := []rune(n)
			ip.byLen[len(runes)] = append(ip.byLen[len(runes)], bucketNorm{
				runes:    runes,
				entryIdx: entryIdx,
			})
		}
	}
	return ip, nil
}

func uniqueNorms(label string, aliases []string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		n := Normalize(raw)
		if n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	add(label)
	for _, a := range aliases {
		add(a)
	}
	return out
}

// PromptCount returns the number of prompts loaded across all packs.
func (s *Store) PromptCount() int {
	if s == nil {
		return 0
	}
	return len(s.order)
}

// PackCount returns the number of packs loaded.
func (s *Store) PackCount() int {
	if s == nil {
		return 0
	}
	return len(s.packs)
}

// Prompt returns a prompt by id.
func (s *Store) Prompt(id string) (Prompt, bool) {
	ip, ok := s.prompts[id]
	if !ok {
		return Prompt{}, false
	}
	return ip.Prompt, true
}

// Entry returns an entry by prompt id and entry id.
func (s *Store) Entry(promptID, entryID string) (Entry, bool) {
	ip, ok := s.prompts[promptID]
	if !ok {
		return Entry{}, false
	}
	for _, e := range ip.entries {
		if e.ID == entryID {
			return e.Entry, true
		}
	}
	return Entry{}, false
}

// EntryRarity returns the rarity for prompt+entry, if found.
func (s *Store) EntryRarity(promptID, entryID string) (int, bool) {
	e, ok := s.Entry(promptID, entryID)
	if !ok {
		return 0, false
	}
	return e.Rarity, true
}

// EntryTier returns the career tier for prompt+entry, if found.
func (s *Store) EntryTier(promptID, entryID string) (Tier, bool) {
	r, ok := s.EntryRarity(promptID, entryID)
	if !ok {
		return "", false
	}
	return TierFor(r), true
}

// RandomPrompts returns up to n prompts whose ids are not in exclude.
// Selection is without replacement within the returned slice.
func (s *Store) RandomPrompts(n int, exclude []string) []Prompt {
	if s == nil || n <= 0 {
		return nil
	}
	skip := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		skip[id] = struct{}{}
	}
	var pool []string
	for _, id := range s.order {
		if _, ok := skip[id]; ok {
			continue
		}
		pool = append(pool, id)
	}
	if len(pool) == 0 {
		return nil
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]Prompt, 0, n)
	for _, id := range pool[:n] {
		out = append(out, s.prompts[id].Prompt)
	}
	return out
}
