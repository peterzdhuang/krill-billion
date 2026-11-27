package supabank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

const restPage = 1000

func loadREST(ctx context.Context, projectURL, serviceKey string) (*data.Pack, error) {
	var rows []Row
	for offset := 0; ; offset += restPage {
		if offset > restPage*200 {
			return nil, fmt.Errorf("answer bank page limit exceeded")
		}
		q := url.Values{}
		q.Set("select", "prompt_id,prompt_text,category_name,entity_id,label,rarity,alias")
		q.Set("order", "prompt_id.asc,entity_id.asc,alias.asc")
		q.Set("limit", strconv.Itoa(restPage))
		q.Set("offset", strconv.Itoa(offset))
		endpoint := projectURL + "/rest/v1/answer_bank?" + q.Encode()
		body, err := restDo(ctx, http.MethodGet, endpoint, serviceKey, nil)
		if err != nil {
			return nil, err
		}
		var page []struct {
			PromptID     string  `json:"prompt_id"`
			PromptText   string  `json:"prompt_text"`
			CategoryName string  `json:"category_name"`
			EntityID     string  `json:"entity_id"`
			Label        string  `json:"label"`
			Rarity       int     `json:"rarity"`
			Alias        *string `json:"alias"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode answer bank: %w", err)
		}
		for _, p := range page {
			r := Row{
				PromptID:     p.PromptID,
				PromptText:   p.PromptText,
				CategoryName: p.CategoryName,
				EntityID:     p.EntityID,
				Label:        p.Label,
				Rarity:       p.Rarity,
			}
			if p.Alias != nil {
				r.Alias = *p.Alias
			}
			rows = append(rows, r)
		}
		if len(page) < restPage {
			break
		}
	}
	return Assemble(rows)
}

func upsertREST(ctx context.Context, projectURL, serviceKey string, pack *data.Pack) error {
	payload, err := payloadFromPack(pack)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"payload": payload})
	if err != nil {
		return err
	}
	_, err = restDo(ctx, http.MethodPost, projectURL+"/rest/v1/rpc/upsert_answer_bank", serviceKey, body)
	return err
}

func restDo(ctx context.Context, method, endpoint, serviceKey string, body []byte) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Prefer", "return=minimal")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, redact(fmt.Errorf("supabase rest: %w", err), serviceKey)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, redact(err, serviceKey)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, redact(fmt.Errorf("supabase rest HTTP %d: %s", resp.StatusCode, msg), serviceKey)
	}
	return respBody, nil
}
