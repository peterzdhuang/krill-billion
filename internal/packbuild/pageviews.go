package packbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const pageviewsBase = "https://wikimedia.org/api/rest_v1/metrics/pageviews/per-article/en.wikipedia/all-access/user"

type pageviewsResponse struct {
	Items []struct {
		Views int64 `json:"views"`
	} `json:"items"`
}

// PageviewsClient sums the last 12 months of enwiki pageviews with disk cache.
type PageviewsClient struct {
	HTTP       *http.Client
	CacheDir   string
	Offline    bool
	Concurrency int
	MinGap     time.Duration

	mu     sync.Mutex
	lastAt time.Time
}

func (c *PageviewsClient) SumLast12Months(ctx context.Context, article string) (int64, error) {
	key := cacheKey(article)
	cachePath := filepath.Join(c.CacheDir, "pageviews", key+".json")
	if b, err := os.ReadFile(cachePath); err == nil {
		var cached struct {
			Views int64  `json:"views"`
			Start string `json:"start"`
			End   string `json:"end"`
		}
		if err := json.Unmarshal(b, &cached); err == nil {
			start, end := monthWindow(time.Now().UTC())
			if cached.Start == start && cached.End == end {
				return cached.Views, nil
			}
		}
	}
	if c.Offline {
		// Offline: use any cached value regardless of window.
		if b, err := os.ReadFile(cachePath); err == nil {
			var cached struct {
				Views int64 `json:"views"`
			}
			if err := json.Unmarshal(b, &cached); err == nil {
				return cached.Views, nil
			}
		}
		return 0, fmt.Errorf("offline: no pageviews cache for %q", article)
	}

	start, end := monthWindow(time.Now().UTC())
	views, err := c.fetch(ctx, article, start, end)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return 0, err
	}
	payload, _ := json.MarshalIndent(struct {
		Views   int64  `json:"views"`
		Start   string `json:"start"`
		End     string `json:"end"`
		Article string `json:"article"`
	}{Views: views, Start: start, End: end, Article: article}, "", "  ")
	if err := os.WriteFile(cachePath, payload, 0o644); err != nil {
		return 0, err
	}
	return views, nil
}

// FetchMany loads pageviews for many articles with bounded concurrency.
func (c *PageviewsClient) FetchMany(ctx context.Context, articles []string) (map[string]int64, error) {
	out := make(map[string]int64, len(articles))
	n := c.Concurrency
	if n <= 0 {
		n = 4
	}
	type result struct {
		article string
		views   int64
		err     error
	}
	ch := make(chan string)
	resCh := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for art := range ch {
				v, err := c.SumLast12Months(ctx, art)
				resCh <- result{article: art, views: v, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resCh)
	}()
	go func() {
		defer close(ch)
		seen := make(map[string]struct{}, len(articles))
		for _, a := range articles {
			if a == "" {
				continue
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			select {
			case <-ctx.Done():
				return
			case ch <- a:
			}
		}
	}()

	var firstErr error
	failed := 0
	for r := range resCh {
		if r.err != nil {
			failed++
			if firstErr == nil {
				firstErr = r.err
			}
			out[r.article] = 0
			continue
		}
		out[r.article] = r.views
	}
	if failed > 0 && len(out) == failed {
		return out, firstErr
	}
	return out, nil
}

func (c *PageviewsClient) fetch(ctx context.Context, article, start, end string) (int64, error) {
	title := strings.ReplaceAll(article, " ", "_")
	escaped := url.PathEscape(title)
	// PathEscape encodes "/" which we need; but Wikimedia wants %20→_ already done.
	// Keep PathEscape for other chars; undo encoding of "/" if any.
	u := fmt.Sprintf("%s/%s/monthly/%s/%s", pageviewsBase, escaped, start, end)

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 400 * time.Millisecond
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(backoff):
			}
		}
		if err := c.wait(ctx); err != nil {
			return 0, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")

		client := c.HTTP
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			// No pageviews data → treat as 0 views (very rare).
			return 0, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("pageviews HTTP %d: %s", resp.StatusCode, truncate(string(body), 120))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("pageviews HTTP %d for %q: %s", resp.StatusCode, article, truncate(string(body), 120))
		}
		var parsed pageviewsResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return 0, err
		}
		var sum int64
		for _, it := range parsed.Items {
			sum += it.Views
		}
		return sum, nil
	}
	return 0, lastErr
}

func (c *PageviewsClient) wait(ctx context.Context) error {
	gap := c.MinGap
	if gap <= 0 {
		gap = 50 * time.Millisecond
	}
	for {
		c.mu.Lock()
		wait := time.Duration(0)
		if !c.lastAt.IsZero() {
			wait = gap - time.Since(c.lastAt)
		}
		if wait <= 0 {
			c.lastAt = time.Now()
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func monthWindow(now time.Time) (start, end string) {
	// Inclusive last 12 calendar months ending at the current month.
	endMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startMonth := endMonth.AddDate(0, -11, 0)
	// API end date can be any day in the end month; use today.
	endDay := now
	if endDay.Before(endMonth) {
		endDay = endMonth
	}
	return startMonth.Format("20060102"), endDay.Format("20060102")
}

func cacheKey(article string) string {
	s := strings.ToLower(strings.ReplaceAll(article, " ", "_"))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if len(out) > 180 {
		out = out[:180]
	}
	if out == "" {
		out = "empty"
	}
	return out
}
