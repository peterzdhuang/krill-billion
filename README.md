# Krill Billion

PvP rarity-guessing game: closed-category prompts (rarer valid answers score higher) plus GeoGuessr Duels-style HP and damage multipliers. Friend lobbies only, via shareable link.

Stack: Go backend (WebSocket, in-memory lobbies/matches, JSON question packs) + Next.js frontend.

## Prerequisites

- Go 1.22+
- Node.js 22+ / npm

## Run locally

```bash
# Terminal 1 — API + WebSocket server (:8080)
make server

# Terminal 2 — Next.js frontend (:3000), proxies /api and /ws to :8080
make web
```

Open http://localhost:3000. Health check: `curl http://localhost:8080/healthz` → `ok`.

## Packs

Question packs live in `packs/*.json`. Main pack: `packs/core-v1.json`. Sample: `packs/sample.json`.

Prompt definitions for the builder are YAML under `packs/defs/`. A two-prompt acceptance fixture is in `packs/defs-sample/`.

Rebuild packs:

```bash
make packbuild
# or: go run ./cmd/packbuild --defs packs/defs --out packs/core-v1.json
# flags: --only <prompt id>  --offline  --cache .packbuild-cache
```

## Tests

```bash
make test
```

## CI

GitHub Actions runs `go vet` / `go test` and `npm ci` / lint / build under `web/`.
