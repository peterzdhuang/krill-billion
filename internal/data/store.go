package data

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// indexedEntry holds an entry. Aliases are folded into exact/byLen at index
// time and not retained, cutting RAM on multi‑million‑entry banks.
type indexedEntry struct {
	ID     string
	Label  string
	Rarity int
}

// bucketNorm is one precomputed normalized form, keyed by rune length for
// fuzzy scans. The string is interned and shared with exact/norms.
type bucketNorm struct {
	norm     string
	entryIdx int
}

type indexedPrompt struct {
	ID       string
	Category string
	Text     string
	entries  []indexedEntry
	// exact maps normalized form -> entry index (first wins at load).
	exact map[string]int
	// byLen buckets normalized forms by rune length for fuzzy matching.
	byLen map[int][]bucketNorm
}

type packMeta struct {
	PackID      string
	Version     int
	BuiltAt     string
	Attribution []string
	PromptCount int
}

// Store holds all loaded packs in memory for matching and prompt selection.
type Store struct {
	packs   []packMeta
	prompts map[string]*indexedPrompt // prompt id -> prompt
	order   []string                  // prompt ids in load order
	intern  *stringIntern
}

// Load reads every *.json / *.json.gz pack under dir (skips nested dirs and
// files named sample.json so the tiny fixture is not mixed into production).
func Load(dir string) (*Store, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read pack dir %q: %w", dir, err)
	}

	haveJSON := map[string]struct{}{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".json") && !strings.HasSuffix(lower, ".json.gz") {
			haveJSON[lower] = struct{}{}
		}
	}
	var jsonFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if lower == "sample.json" {
			continue
		}
		switch {
		case strings.HasSuffix(lower, ".json.gz"):
			// Prefer uncompressed twin when both exist.
			base := strings.TrimSuffix(lower, ".gz")
			if _, ok := haveJSON[base]; ok {
				continue
			}
			jsonFiles = append(jsonFiles, name)
		case strings.HasSuffix(lower, ".json"):
			jsonFiles = append(jsonFiles, name)
		}
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
		intern:  newStringIntern(1 << 20),
	}
	promptSeen := make(map[string]string) // prompt id -> pack id
	for _, pack := range packs {
		if pack.PackID == "" {
			return nil, fmt.Errorf("pack missing pack_id")
		}
		if err := s.indexPack(pack, promptSeen); err != nil {
			return nil, fmt.Errorf("pack %q: %w", pack.PackID, err)
		}
		s.packs = append(s.packs, packMeta{
			PackID:      pack.PackID,
			Version:     pack.Version,
			BuiltAt:     pack.BuiltAt,
			Attribution: pack.Attribution,
			PromptCount: len(pack.Prompts),
		})
	}
	return s, nil
}

func loadPackFile(path string) (Pack, error) {
	f, err := os.Open(path)
	if err != nil {
		return Pack{}, fmt.Errorf("read pack %q: %w", path, err)
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return Pack{}, fmt.Errorf("gunzip pack %q: %w", path, err)
		}
		defer gz.Close()
		r = gz
	}
	var pack Pack
	if err := json.NewDecoder(r).Decode(&pack); err != nil {
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
		ip, err := s.indexPrompt(p)
		if err != nil {
			return err
		}
		promptSeen[p.ID] = pack.PackID
		s.prompts[p.ID] = ip
		s.order = append(s.order, p.ID)
	}
	return nil
}

func (s *Store) indexPrompt(p Prompt) (*indexedPrompt, error) {
	ip := &indexedPrompt{
		ID:       s.intern.Get(p.ID),
		Category: s.intern.Get(p.Category),
		Text:     s.intern.Get(p.Text),
		entries:  make([]indexedEntry, 0, len(p.Entries)),
		exact:    make(map[string]int, len(p.Entries)*2),
		byLen:    make(map[int][]bucketNorm),
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

		norms := uniqueNorms(e.Label, e.Aliases, s.intern)
		entryIdx := len(ip.entries)
		ie := indexedEntry{
			ID:     s.intern.Get(e.ID),
			Label:  s.intern.Get(e.Label),
			Rarity: e.Rarity,
		}
		ip.entries = append(ip.entries, ie)
		for _, n := range norms {
			if n == "" {
				continue
			}
			if _, exists := ip.exact[n]; !exists {
				ip.exact[n] = entryIdx
				// Only index the first owner of a norm into byLen (same as exact).
				nLen := len([]rune(n))
				ip.byLen[nLen] = append(ip.byLen[nLen], bucketNorm{
					norm:     n,
					entryIdx: entryIdx,
				})
			}
		}
	}
	return ip, nil
}

func uniqueNorms(label string, aliases []string, intern *stringIntern) []string {
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
		if intern != nil {
			n = intern.Get(n)
		}
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

// EntryCount returns the total number of answer entries across all prompts.
func (s *Store) EntryCount() int {
	if s == nil {
		return 0
	}
	n := 0
	for _, id := range s.order {
		n += len(s.prompts[id].entries)
	}
	return n
}

// Prompt returns a prompt by id (without the full entry list — use Entry/Match).
func (s *Store) Prompt(id string) (Prompt, bool) {
	ip, ok := s.prompts[id]
	if !ok {
		return Prompt{}, false
	}
	return Prompt{ID: ip.ID, Category: ip.Category, Text: ip.Text}, true
}

// Entry returns an entry by prompt id and entry id.
func (s *Store) Entry(promptID, entryID string) (Entry, bool) {
	ip, ok := s.prompts[promptID]
	if !ok {
		return Entry{}, false
	}
	for _, e := range ip.entries {
		if e.ID == entryID {
			return Entry{ID: e.ID, Label: e.Label, Rarity: e.Rarity}, true
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
// Returned prompts omit Entries (clients only need id/text/category).
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
		ip := s.prompts[id]
		out = append(out, Prompt{ID: ip.ID, Category: ip.Category, Text: ip.Text})
	}
	return out
}

// HeapInUseMB returns approximate heap in-use after a GC, for sizing reports.
func HeapInUseMB() float64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapInuse) / (1024 * 1024)
}
