// Package supabank loads and upserts the answer bank in Supabase.
// Credentials come from the environment only.
package supabank

import (
	"os"
	"strings"
)

const (
	envURL = "SUPABASE_URL"
	envDB  = "SUPABASE_DB_URL"
	envKey = "SUPABASE_SERVICE_ROLE_KEY"
)

// Config is a non-secret description of how to reach Supabase.
// DBURL and ServiceKey are secrets; do not log them.
type Config struct {
	URL        string
	DBURL      string
	ServiceKey string
}

// FromEnv reports whether Supabase is configured.
// Both the project URL and either a Postgres URL or the service role key
// are required. A URL alone is not enough.
func FromEnv() (Config, bool) {
	cfg := Config{
		URL:        strings.TrimRight(strings.TrimSpace(os.Getenv(envURL)), "/"),
		DBURL:      strings.TrimSpace(os.Getenv(envDB)),
		ServiceKey: strings.TrimSpace(os.Getenv(envKey)),
	}
	if cfg.URL == "" || cfg.Mode() == "" {
		return Config{}, false
	}
	return cfg, true
}

// Mode is "postgres" when SUPABASE_DB_URL is set, otherwise "rest"
// when the service role key is set. Postgres wins when both are set
// because the upsert is one transaction.
func (c Config) Mode() string {
	if strings.TrimSpace(c.DBURL) != "" {
		return "postgres"
	}
	if strings.TrimSpace(c.ServiceKey) != "" && strings.TrimSpace(c.URL) != "" {
		return "rest"
	}
	return ""
}
