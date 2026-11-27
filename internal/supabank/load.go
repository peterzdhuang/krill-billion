package supabank

import (
	"context"
	"fmt"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

// Load reads the answer bank. It returns an empty pack (not an error) when
// the schema is reachable and holds no prompts.
func Load(ctx context.Context, cfg Config) (*data.Pack, error) {
	var pack *data.Pack
	var err error
	switch cfg.Mode() {
	case "postgres":
		pack, err = loadPostgres(ctx, cfg.DBURL)
	case "rest":
		pack, err = loadREST(ctx, cfg.URL, cfg.ServiceKey)
	default:
		err = fmt.Errorf("supabase is not configured")
	}
	if err != nil {
		return nil, fmt.Errorf("%w — if the schema is missing, run db/migrations/001_answer_bank.sql", err)
	}
	return pack, nil
}

// Upsert writes pack into Supabase. Prompts included in pack have their
// entity links replaced. Prompts omitted from pack are left in place.
func Upsert(ctx context.Context, cfg Config, pack *data.Pack) error {
	var err error
	switch cfg.Mode() {
	case "postgres":
		err = upsertPostgres(ctx, cfg.DBURL, pack)
	case "rest":
		err = upsertREST(ctx, cfg.URL, cfg.ServiceKey, pack)
	default:
		err = fmt.Errorf("supabase is not configured")
	}
	if err != nil {
		return fmt.Errorf("%w — if the schema is missing, run db/migrations/001_answer_bank.sql", err)
	}
	return nil
}

// LoadStore returns the in-memory store and a source tag:
// "supabase", "json", or "json-fallback" (Supabase was configured but empty).
// A configured Supabase that errors does not fall back.
func LoadStore(ctx context.Context, packsDir string) (*data.Store, string, error) {
	cfg, ok := FromEnv()
	if !ok {
		store, err := data.Load(packsDir)
		return store, "json", err
	}
	pack, err := Load(ctx, cfg)
	if err != nil {
		return nil, "", err
	}
	if pack == nil || len(pack.Prompts) == 0 {
		store, err := data.Load(packsDir)
		return store, "json-fallback", err
	}
	store, err := data.NewFromPacks(*pack)
	return store, "supabase", err
}
