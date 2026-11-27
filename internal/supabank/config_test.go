package supabank

import "testing"

func TestFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		ok   bool
		mode string
	}{
		{name: "unset", ok: false},
		{
			name: "url only",
			env:  map[string]string{envURL: "https://example.supabase.co"},
			ok:   false,
		},
		{
			name: "postgres",
			env: map[string]string{
				envURL: "https://example.supabase.co/",
				envDB:  "postgres://user:secret@db.example:5432/postgres",
			},
			ok:   true,
			mode: "postgres",
		},
		{
			name: "service role",
			env: map[string]string{
				envURL: "https://example.supabase.co",
				envKey: "service-role",
			},
			ok:   true,
			mode: "rest",
		},
		{
			name: "both prefers postgres",
			env: map[string]string{
				envURL: "https://example.supabase.co",
				envDB:  "postgres://user:secret@db.example:5432/postgres",
				envKey: "service-role",
			},
			ok:   true,
			mode: "postgres",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envURL, "")
			t.Setenv(envDB, "")
			t.Setenv(envKey, "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg, ok := FromEnv()
			if ok != tt.ok {
				t.Fatalf("FromEnv ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if cfg.Mode() != tt.mode {
				t.Fatalf("mode = %q, want %q", cfg.Mode(), tt.mode)
			}
			if cfg.URL != "https://example.supabase.co" {
				t.Fatalf("url = %q", cfg.URL)
			}
		})
	}
}
