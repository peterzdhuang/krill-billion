package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/lobby"
	"github.com/peterzdhuang/krill-billion/internal/match"
	"github.com/peterzdhuang/krill-billion/internal/supabank"
	"github.com/peterzdhuang/krill-billion/internal/ws"
)

func main() {
	packsDir := os.Getenv("PACKS_DIR")
	if packsDir == "" {
		packsDir = "./packs"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	store, source, err := supabank.LoadStore(ctx, packsDir)
	cancel()
	if err != nil {
		log.Fatalf("load packs: %v", err)
	}
	if source == "json-fallback" {
		log.Printf("supabase answer bank is empty; using JSON packs in %s", packsDir)
	}
	log.Printf("loaded %d prompts / %d entries from %d packs (%s); heap≈%.0f MiB",
		store.PromptCount(), store.EntryCount(), store.PackCount(), source, data.HeapInUseMB())

	frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	reg := lobby.NewRegistry(store, match.RealClock{})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/lobbies", ws.CreateLobbyHTTP(reg))
	mux.Handle("/ws", &ws.Handler{Registry: reg, AllowedOrigin: frontendOrigin})

	handler := withCORS(mux, frontendOrigin)

	addr := listenAddr()
	log.Printf("server listening on %s (FRONTEND_ORIGIN=%q)", addr, frontendOrigin)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func listenAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	if v := os.Getenv("ADDR"); v != "" {
		return v
	}
	return ":8080"
}

func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originOK(origin, allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func originOK(origin, allowed string) bool {
	if origin == "" {
		return false
	}
	if allowed == "" || allowed == "*" {
		// Local/dev: reflect any origin when unset.
		return true
	}
	for _, a := range strings.Split(allowed, ",") {
		if strings.TrimSpace(a) == origin {
			return true
		}
	}
	return false
}
