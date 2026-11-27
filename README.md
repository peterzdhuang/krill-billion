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

Leave `NEXT_PUBLIC_SERVER_URL` unset for local same-origin rewrites. Optional server CORS:

```bash
FRONTEND_ORIGIN=http://localhost:3000 make server
```

## Packs

Question packs live in `packs/*.json`. Main pack: `packs/core-v1.json` (hand-built for MVP; regenerate later via Wikidata when online). Sample: `packs/sample.json`.

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

## Deploy

### Go server → Fly.io (`krill-billion`, region `sjc`)

Multi-stage `Dockerfile` bakes `packs/` into the image. Listens on `$PORT` (default 8080). `/healthz` for checks.

```bash
flyctl launch --no-deploy   # or use existing fly.toml
flyctl secrets set FRONTEND_ORIGIN=https://YOUR_VERCEL_APP.vercel.app
flyctl deploy
```

`fly.toml` targets shared-cpu-1x / 256MB with `auto_stop_machines` and `min_machines_running = 0`. Force HTTPS is on; WebSocket upgrades go through the HTTP service.

### Frontend → Vercel

Set the public server URL so the browser talks to Fly (not localhost rewrites):

```
NEXT_PUBLIC_SERVER_URL=https://krill-billion.fly.dev
```

See `web/.env.example`. Deploy the `web/` directory as a Next.js app. Ensure Fly `FRONTEND_ORIGIN` matches the Vercel URL (comma-separated if you have preview + prod).

## CI

GitHub Actions runs `go vet` / `go test` and `npm ci` / lint / build under `web/`.
