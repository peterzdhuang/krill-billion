# Krill Billion

PvP rarity-guessing game: closed-category prompts (rarer valid answers score higher) plus GeoGuessr Duels-style HP and damage multipliers. Friend lobbies only, via shareable link.

Stack: Go backend (WebSocket, in-memory lobbies/matches, JSON question packs) + Next.js frontend.

## Sound and marks

Locks, rejects, reveals, and match-end stingers are synthesized in the browser with the Web Audio API, following the sound design of [Krillion](https://krillion.io). Visuals pull Krillion assets directly (used with permission): tier marks and gold variants under `web/public/krillion/tiers/`, buddy auras in `web/public/krillion/fx/`, descent background `web/public/krillion/bg/water-bg.webp`, and party puffers in `web/public/krillion/party/`. Mute persists in `localStorage` under `kb-muted` (`"1"` is muted; absent or `"0"` leaves sound on).

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

JSON packs under `packs/*.json` are loaded together when Supabase is not configured:

| Pack | How to build | Source |
| --- | --- | --- |
| `core-v1.json` | `make packbuild` | Wikidata + pageviews (+ REST Countries / PokeAPI checks) |
| `broad-v1.json` | `make packbuild-broad` | Static broad categories (programming languages, bands, fruit, NBA) |
| `krillion-v1.json` | `make packbuild-krillion` | [Krillion](https://krillion.io) prompts & answer banks (used with permission) |
| `sample.json` | hand-authored | Tiny local fixture |

Prompt definitions: YAML under `packs/defs/` (Wikidata SPARQL) and `packs/defs-broad/` (static `entries:` lists). A two-prompt acceptance fixture is in `packs/defs-sample/`. Krillion day files land in `packs/import/krillion/` (gitignored); re-fetch with `make packbuild-krillion`.

The server keeps one in-memory copy of the bank. On boot it loads that copy from Supabase when credentials are set, and from `packs/*.json` otherwise.

### Supabase answer bank

Apply `db/migrations/001_answer_bank.sql` in the Supabase SQL editor once. It creates categories, prompts, entities, aliases, and per-prompt rarity. Row level security is on and there are no anon policies.

Then set both of these (never commit them):

| Variable | What |
| --- | --- |
| `SUPABASE_URL` | Project URL, `https://<ref>.supabase.co` |
| `SUPABASE_DB_URL` | Postgres connection string |

If you do not have the database string, set `SUPABASE_SERVICE_ROLE_KEY` instead of `SUPABASE_DB_URL`. When both are set, the server uses Postgres. A URL with neither secret is ignored, and the JSON packs load.

```bash
# Fill Supabase from a built pack. Omitted prompts are left as they are.
SUPABASE_URL=https://<ref>.supabase.co \
SUPABASE_DB_URL='postgresql://...' \
  make packbuild
```

Without those variables, `make packbuild` only writes JSON and logs that the upsert was skipped. If Supabase is configured but the bank has no prompts, the server logs that and uses the JSON packs. If it is configured and the query fails, the process exits.

Fly (do not put the values in `fly.toml`):

```bash
flyctl secrets set \
  SUPABASE_URL=https://<ref>.supabase.co \
  SUPABASE_DB_URL='postgresql://...'
# or SUPABASE_SERVICE_ROLE_KEY=... instead of SUPABASE_DB_URL
```

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
