package main

import (
	"log"
	"net/http"
	"os"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/lobby"
	"github.com/peterzdhuang/krill-billion/internal/match"
	"github.com/peterzdhuang/krill-billion/internal/ws"
)

func main() {
	packsDir := os.Getenv("PACKS_DIR")
	if packsDir == "" {
		packsDir = "./packs"
	}
	store, err := data.Load(packsDir)
	if err != nil {
		log.Fatalf("load packs: %v", err)
	}
	log.Printf("loaded %d prompts from %d packs", store.PromptCount(), store.PackCount())

	reg := lobby.NewRegistry(store, match.RealClock{})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/lobbies", ws.CreateLobbyHTTP(reg))
	mux.Handle("/ws", &ws.Handler{Registry: reg})

	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
