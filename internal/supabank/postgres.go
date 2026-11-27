package supabank

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/peterzdhuang/krill-billion/internal/data"
)

func loadPostgres(ctx context.Context, dbURL string) (*data.Pack, error) {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, redact(fmt.Errorf("connect: %w", err), dbURL)
	}
	defer conn.Close(ctx)

	pgxRows, err := conn.Query(ctx, `
		select prompt_id, prompt_text, category_name, entity_id, label, rarity, alias
		from answer_bank`)
	if err != nil {
		return nil, redact(fmt.Errorf("query answer_bank: %w", err), dbURL)
	}
	defer pgxRows.Close()

	var rows []Row
	for pgxRows.Next() {
		var r Row
		var alias *string
		if err := pgxRows.Scan(&r.PromptID, &r.PromptText, &r.CategoryName, &r.EntityID, &r.Label, &r.Rarity, &alias); err != nil {
			return nil, redact(fmt.Errorf("scan answer_bank: %w", err), dbURL)
		}
		if alias != nil {
			r.Alias = *alias
		}
		rows = append(rows, r)
	}
	if err := pgxRows.Err(); err != nil {
		return nil, redact(fmt.Errorf("read answer_bank: %w", err), dbURL)
	}
	pack, err := Assemble(rows)
	if err != nil {
		return nil, err
	}
	return pack, nil
}

func upsertPostgres(ctx context.Context, dbURL string, pack *data.Pack) error {
	payload, err := payloadFromPack(pack)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return redact(fmt.Errorf("connect: %w", err), dbURL)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `select upsert_answer_bank($1::jsonb)`, string(body)); err != nil {
		return redact(fmt.Errorf("upsert_answer_bank: %w", err), dbURL)
	}
	return nil
}
