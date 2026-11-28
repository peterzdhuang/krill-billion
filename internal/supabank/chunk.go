package supabank

import (
	"encoding/json"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

// restUpsertBudget is the approximate max JSON body size per REST RPC call.
// PostgREST / statement timeouts blow up on multi‑MB single upserts.
const restUpsertBudget = 300 * 1024

// chunkPacks splits a pack into smaller packs so each upsert JSON stays near
// restUpsertBudget. Categories/entities may repeat across chunks (upsert is
// idempotent). Prompt entity links are replaced per prompt id in each chunk.
func chunkPacks(pack *data.Pack, budget int) ([]*data.Pack, error) {
	if pack == nil {
		return nil, nil
	}
	if budget <= 0 {
		budget = restUpsertBudget
	}
	if len(pack.Prompts) == 0 {
		return []*data.Pack{pack}, nil
	}

	var chunks []*data.Pack
	var cur []data.Prompt
	curBytes := 0

	flush := func() {
		if len(cur) == 0 {
			return
		}
		cp := *pack
		cp.Prompts = append([]data.Prompt{}, cur...)
		chunks = append(chunks, &cp)
		cur = nil
		curBytes = 0
	}

	for _, p := range pack.Prompts {
		// Estimate size of this prompt's contribution.
		est, err := estimatePromptPayloadBytes(p)
		if err != nil {
			return nil, err
		}
		if len(cur) > 0 && curBytes+est > budget {
			flush()
		}
		// A single enormous prompt still goes out alone.
		cur = append(cur, p)
		curBytes += est
		if curBytes >= budget {
			flush()
		}
	}
	flush()
	return chunks, nil
}

func estimatePromptPayloadBytes(p data.Prompt) (int, error) {
	// Marshal a miniature payload for this prompt only.
	mini := data.Pack{PackID: "x", Version: 1, Prompts: []data.Prompt{p}}
	payload, err := payloadFromPack(&mini)
	if err != nil {
		return 0, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}
