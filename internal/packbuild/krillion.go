package packbuild

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

const (
	krillionIOBase   = "https://krillion.io"
	krillionGameBase = "https://krillion-game.com"
	// krillionMaxEntries caps answers per prompt when > 0. Zero means keep all
	// answers (preferred for the deep bank; large JSON is gitignored / gzipped).
	krillionMaxEntries = 0
)

// KrillionBuildOptions controls aggregation of dated Krillion reveal files.
type KrillionBuildOptions struct {
	ImportDir  string
	OutPath    string
	PackID     string
	MaxEntries int // 0 = DefaultMaxEntries; negative = unlimited
	MinEntries int // 0 = keep all prompts
}

// KrillionDay is one dated reveal in the shape of krillion.io/api/reveal.
type KrillionDay struct {
	Date    string           `json:"date"`
	Source  string           `json:"source,omitempty"`
	Prompts []KrillionPrompt `json:"prompts"`
}

// KrillionPrompt is one prompt plus its accepted answers.
type KrillionPrompt struct {
	ID      string           `json:"id"`
	Text    string           `json:"text"`
	Answers []KrillionAnswer `json:"answers"`
}

// KrillionAnswer is one accepted spelling with optional aliases and score.
type KrillionAnswer struct {
	Answer  string   `json:"answer"`
	Aliases []string `json:"aliases,omitempty"`
	Tier    string   `json:"tier,omitempty"`
	Score   int      `json:"score"`
}

type krillionGameAnswers struct {
	Answers []KrillionAnswer `json:"answers"`
}

// FetchKrillion writes dated reveal JSON files under dir.
// It pulls today's sheet from krillion.io, then backfills recent archive
// days from krillion-game.com (full answer banks keyed like daily:DATE:pN).
func FetchKrillion(ctx context.Context, dir string, httpClient *http.Client) error {
	if dir == "" {
		return fmt.Errorf("krillion import dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	today, err := fetchKrillionIOToday(ctx, httpClient)
	if err != nil {
		return err
	}
	if err := writeKrillionDay(dir, today); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "krillion.io %s: %d prompts\n", today.Date, len(today.Prompts))

	dates, err := listKrillionGameArchiveDates(ctx, httpClient)
	if err != nil {
		return fmt.Errorf("list archive dates: %w", err)
	}
	for _, date := range dates {
		if date == today.Date {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, date+".json")); err == nil {
			fmt.Fprintf(os.Stderr, "krillion-game %s: already cached\n", date)
			continue
		}
		var day KrillionDay
		var err error
		for attempt := 0; attempt < 5; attempt++ {
			day, err = fetchKrillionGameDay(ctx, httpClient, date)
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "429") {
				break
			}
			wait := time.Duration(45+attempt*30) * time.Second
			fmt.Fprintf(os.Stderr, "  rate limited on %s; waiting %s\n", date, wait)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", date, err)
			continue
		}
		if err := writeKrillionDay(dir, day); err != nil {
			return err
		}
		n := 0
		for _, p := range day.Prompts {
			n += len(p.Answers)
		}
		fmt.Fprintf(os.Stderr, "krillion-game %s: %d prompts, %d answers\n", date, len(day.Prompts), n)
		// Be gentle between days so the archive API stays happy.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return nil
}

// BuildKrillionPack aggregates every dated JSON under importDir into one pack,
// merging duplicate prompt texts and mapping Krillion scores onto rarity.
func BuildKrillionPack(importDir, outPath, packID string) (*data.Pack, error) {
	return BuildKrillionPackOpts(KrillionBuildOptions{
		ImportDir:  importDir,
		OutPath:    outPath,
		PackID:     packID,
		MaxEntries: krillionMaxEntries,
	})
}

// BuildKrillionPackOpts is BuildKrillionPack with explicit caps / min-entries.
func BuildKrillionPackOpts(opt KrillionBuildOptions) (*data.Pack, error) {
	importDir := opt.ImportDir
	outPath := opt.OutPath
	packID := opt.PackID
	if importDir == "" {
		return nil, fmt.Errorf("krillion import dir required")
	}
	if outPath == "" {
		outPath = "packs/krillion-v1.json"
	}
	if packID == "" {
		packID = strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	}
	maxEntries := opt.MaxEntries
	if maxEntries == 0 {
		maxEntries = DefaultMaxEntries
	} else if maxEntries < 0 {
		maxEntries = 0 // unlimited
	}

	entries, err := os.ReadDir(importDir)
	if err != nil {
		return nil, fmt.Errorf("read krillion import: %w", err)
	}
	type aggAnswer struct {
		label   string
		aliases map[string]struct{}
		score   int
	}
	type aggPrompt struct {
		text    string
		answers map[string]*aggAnswer // normalizeKey -> answer
	}
	byText := map[string]*aggPrompt{}

	dayCount := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(importDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var day KrillionDay
		if err := json.Unmarshal(b, &day); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		dayCount++
		for _, p := range day.Prompts {
			text := strings.TrimSpace(html.UnescapeString(p.Text))
			if text == "" {
				continue
			}
			key := strings.ToLower(text)
			ap := byText[key]
			if ap == nil {
				ap = &aggPrompt{text: text, answers: map[string]*aggAnswer{}}
				byText[key] = ap
			}
			for _, a := range p.Answers {
				label := strings.TrimSpace(a.Answer)
				if label == "" {
					continue
				}
				nk := normalizeKey(label)
				if nk == "" {
					continue
				}
				score := a.Score
				if score <= 0 {
					score = tierScore(a.Tier)
				}
				if score < 0 {
					score = 0
				}
				if score > 100 {
					score = 100
				}
				cur := ap.answers[nk]
				if cur == nil {
					cur = &aggAnswer{label: label, aliases: map[string]struct{}{}, score: score}
					ap.answers[nk] = cur
				} else if score > cur.score {
					cur.score = score
					cur.label = label
				}
				for _, al := range a.Aliases {
					al = strings.TrimSpace(al)
					if al == "" || strings.EqualFold(al, label) {
						continue
					}
					cur.aliases[al] = struct{}{}
				}
			}
		}
	}
	if dayCount == 0 {
		return nil, fmt.Errorf("no krillion day JSON in %s", importDir)
	}

	prompts := make([]data.Prompt, 0, len(byText))
	for _, ap := range byText {
		type row struct {
			label   string
			aliases []string
			score   int
		}
		rows := make([]row, 0, len(ap.answers))
		for _, a := range ap.answers {
			aliases := make([]string, 0, len(a.aliases))
			for al := range a.aliases {
				aliases = append(aliases, al)
			}
			sort.Strings(aliases)
			rows = append(rows, row{label: a.label, aliases: aliases, score: a.score})
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].score != rows[j].score {
				return rows[i].score < rows[j].score
			}
			return strings.ToLower(rows[i].label) < strings.ToLower(rows[j].label)
		})
		if maxEntries > 0 && len(rows) > maxEntries {
			// Keep the rarest tail (highest scores) plus a common head so
			// easy answers still exist on huge banks.
			head := 500
			if head > len(rows)/5 {
				head = len(rows) / 5
			}
			tail := maxEntries - head
			kept := append([]row{}, rows[:head]...)
			kept = append(kept, rows[len(rows)-tail:]...)
			rows = kept
			sort.SliceStable(rows, func(i, j int) bool {
				if rows[i].score != rows[j].score {
					return rows[i].score < rows[j].score
				}
				return strings.ToLower(rows[i].label) < strings.ToLower(rows[j].label)
			})
		}

		entriesOut := make([]data.Entry, 0, len(rows))
		for _, r := range rows {
			aliases := ExpandAliases(r.label, r.aliases)
			entriesOut = append(entriesOut, data.Entry{
				ID:      krillionEntityID(r.label),
				Label:   r.label,
				Aliases: aliases,
				Rarity:  r.score,
			})
		}
		entriesOut = FoldCaseVariantEntries(entriesOut)
		sortEntries(entriesOut)
		prompts = append(prompts, data.Prompt{
			ID:       krillionPromptID(ap.text),
			Category: krillionCategory(ap.text),
			Text:     stripKrillionNamePrefix(ap.text),
			Entries:  entriesOut,
		})
	}
	sort.SliceStable(prompts, func(i, j int) bool {
		if prompts[i].Category != prompts[j].Category {
			return prompts[i].Category < prompts[j].Category
		}
		return prompts[i].ID < prompts[j].ID
	})

	removed := 0
	if opt.MinEntries > 0 {
		prompts, removed = FilterPromptsMinEntries(prompts, opt.MinEntries)
		fmt.Fprintf(os.Stderr, "krillion min-entries %d: kept %d, removed %d\n", opt.MinEntries, len(prompts), removed)
	}

	pack := &data.Pack{
		PackID:  packID,
		Version: 1,
		BuiltAt: time.Now().UTC().Format(time.RFC3339),
		Attribution: []string{
			"Krillion (krillion.io) — prompts and answer banks used with permission",
			"Archive backfill via krillion-game.com answer API",
		},
		Prompts: prompts,
	}
	if err := writePack(outPath, pack); err != nil {
		return nil, err
	}
	_ = writePackGZ(outPath+".gz", pack)
	fmt.Fprintf(os.Stderr, "krillion pack: %d prompts from %d day files → %s\n", len(prompts), dayCount, outPath)
	return pack, nil
}

func fetchKrillionIOToday(ctx context.Context, client *http.Client) (KrillionDay, error) {
	var today struct {
		Date string `json:"date"`
	}
	if err := getJSON(ctx, client, krillionIOBase+"/api/today", &today); err != nil {
		return KrillionDay{}, err
	}
	if today.Date == "" {
		return KrillionDay{}, fmt.Errorf("krillion.io/api/today missing date")
	}
	var day KrillionDay
	if err := getJSON(ctx, client, krillionIOBase+"/api/reveal?date="+url.QueryEscape(today.Date), &day); err != nil {
		return KrillionDay{}, err
	}
	day.Date = today.Date
	day.Source = "krillion.io"
	return day, nil
}

func listKrillionGameArchiveDates(ctx context.Context, client *http.Client) ([]string, error) {
	body, err := getBytes(ctx, client, krillionGameBase+"/archive/")
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`/archive/(20\d{2}-\d{2}-\d{2})/`)
	seen := map[string]struct{}{}
	var dates []string
	for _, m := range re.FindAllStringSubmatch(string(body), -1) {
		d := m[1]
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates, nil
}

func fetchKrillionGameDay(ctx context.Context, client *http.Client, date string) (KrillionDay, error) {
	body, err := getBytes(ctx, client, krillionGameBase+"/archive/"+date+"/")
	if err != nil {
		return KrillionDay{}, err
	}
	html := string(body)
	re := regexp.MustCompile(`<details class="prompt" id="(daily:` + regexp.QuoteMeta(date) + `:p\d+)"[^>]*>[\s\S]*?<span class="prompt-title"[^>]*>([^<]+)</span>`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return KrillionDay{}, fmt.Errorf("no prompts in archive page")
	}
	day := KrillionDay{Date: date, Source: "krillion-game.com", Prompts: make([]KrillionPrompt, 0, len(matches))}
	for _, m := range matches {
		id, text := m[1], strings.TrimSpace(m[2])
		var payload krillionGameAnswers
		if err := getJSON(ctx, client, krillionGameBase+"/api/answers/?prompt="+url.QueryEscape(id), &payload); err != nil {
			return KrillionDay{}, fmt.Errorf("%s: %w", id, err)
		}
		day.Prompts = append(day.Prompts, KrillionPrompt{
			ID:      id,
			Text:    text,
			Answers: payload.Answers,
		})
	}
	return day, nil
}

func writeKrillionDay(dir string, day KrillionDay) error {
	if day.Date == "" {
		return fmt.Errorf("krillion day missing date")
	}
	b, err := json.MarshalIndent(day, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(dir, day.Date+".json"), b, 0o644)
}

func getJSON(ctx context.Context, client *http.Client, rawURL string, dest any) error {
	b, err := getBytes(ctx, client, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func getBytes(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "krill-billion-packbuild (+https://github.com/peterzdhuang/krill-billion)")
	req.Header.Set("Accept", "application/json, text/html;q=0.9,*/*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d: %s", rawURL, resp.StatusCode, truncate(string(b), 180))
	}
	return b, nil
}

func tierScore(tier string) int {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "krillion":
		return 100
	case "deepcut":
		return 85
	case "rare":
		return 60
	case "schooler":
		return 30
	case "plankton", "tooclever":
		return 10
	default:
		return 50
	}
}

func krillionEntityID(label string) string {
	sum := sha1.Sum([]byte("k:" + strings.ToLower(strings.TrimSpace(label))))
	return "k" + hex.EncodeToString(sum[:10])
}

func krillionPromptID(text string) string {
	slug := slugify(stripKrillionNamePrefix(text))
	if slug == "" {
		sum := sha1.Sum([]byte(text))
		return "k-" + hex.EncodeToString(sum[:6])
	}
	if len(slug) > 48 {
		slug = slug[:48]
	}
	sum := sha1.Sum([]byte(strings.ToLower(text)))
	return "k-" + slug + "-" + hex.EncodeToString(sum[:3])
}

func stripKrillionNamePrefix(text string) string {
	t := strings.TrimSpace(html.UnescapeString(text))
	lower := strings.ToLower(t)
	for _, p := range []string{"name a ", "name an ", "name "} {
		if strings.HasPrefix(lower, p) {
			rest := strings.TrimSpace(t[len(p):])
			if rest == "" {
				return t
			}
			// Keep "A/An …" style used by existing packs.
			r := []rune(rest)
			if unicode.IsLower(r[0]) {
				r[0] = unicode.ToUpper(r[0])
			}
			article := "A "
			if startsWithVowelSound(string(r)) {
				article = "An "
			}
			return article + string(r)
		}
	}
	return t
}

func startsWithVowelSound(s string) bool {
	if s == "" {
		return false
	}
	switch unicode.ToLower([]rune(s)[0]) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func krillionCategory(text string) string {
	l := strings.ToLower(text)
	switch {
	case strings.Contains(l, "pokemon") || strings.Contains(l, "pokémon"):
		return "Pokemon"
	case strings.Contains(l, "nba") || strings.Contains(l, "basketball player"):
		return "NBA"
	case strings.Contains(l, "country") || strings.Contains(l, "nation") || strings.Contains(l, "capital"):
		return "Geography"
	case strings.Contains(l, "fruit") || strings.Contains(l, "vegetable") || strings.Contains(l, "food") || strings.Contains(l, "cheese"):
		return "Food"
	case strings.Contains(l, "band") || strings.Contains(l, "singer") || strings.Contains(l, "song") || strings.Contains(l, "album") || strings.Contains(l, "musician"):
		return "Music"
	case strings.Contains(l, "programming language") || strings.Contains(l, "javascript") || strings.Contains(l, "software") || strings.Contains(l, "computer"):
		return "Tech"
	case strings.Contains(l, "movie") || strings.Contains(l, "film") || strings.Contains(l, "actor") || strings.Contains(l, "tv show"):
		return "Entertainment"
	case strings.Contains(l, "sport") || strings.Contains(l, "olympic") || strings.Contains(l, "athlete"):
		return "Sports"
	case strings.Contains(l, "animal") || strings.Contains(l, "bird") || strings.Contains(l, "dog") || strings.Contains(l, "cat") || strings.Contains(l, "breed"):
		return "Animals"
	case strings.Contains(l, "element") || strings.Contains(l, "planet") || strings.Contains(l, "chemical") || strings.Contains(l, "science"):
		return "Science"
	default:
		return "Krillion"
	}
}
