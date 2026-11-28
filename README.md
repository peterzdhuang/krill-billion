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

JSON packs under `packs/*.json` (and `*.json.gz`) are loaded together when Supabase is not configured. `packs/sample.json` is a tiny fixture and is **not** loaded by the server.

**Shipped prompts require ≥ 10,000 accepted answers.** Narrow defs under `packs/defs/` remain for experimentation but are not part of the deep bank.

| Pack | How to build | Source |
| --- | --- | --- |
| `deep-v1.json` | `make packbuild-deep` | Wikidata (paginated) + GeoNames + IMDb + Open Food Facts |
| `krillion-v1.json` | `make packbuild-krillion` | [Krillion](https://krillion.io) prompts & answer banks (used with permission) |
| `core-v1.json` | `make packbuild` | Legacy Wikidata defs (mostly &lt;10k; optional) |
| `broad-v1.json` | `make packbuild-broad` | Legacy static broad categories |
| `sample.json` | hand-authored | Tiny local fixture (tests only) |

Large generated `*-v1.json` files are **gitignored**. Rebuild locally (also writes `*.json.gz`). Optional: commit the `.gz` for Docker JSON fallback.

```bash
make packbuild-deep          # → packs/deep-v1.json[.gz]
make packbuild-krillion      # fetch + build, min 10k answers/prompt
# or both:
make packbuild-all
```

Equivalent `go run` forms:

```bash
go run ./cmd/packbuild --deep --out packs/deep-v1.json
# Faster datasets-only (GeoNames/IMDb/OFF); add Wikidata with full --deep:
go run ./cmd/packbuild --deep --skip-wikidata --out packs/deep-v1.json
go run ./cmd/packbuild --fetch-krillion --krillion-import packs/import/krillion --out packs/krillion-v1.json
# flags: --min-entries 10000  --offline  --cache .packbuild-cache
#         --skip-datasets  --skip-wikidata
```

Per-prompt banks are capped at 40,000 answers (common head + rare tail) so the in-memory store fits a 1 GiB Fly VM while still meeting the 10,000 minimum. Measure heap after load:

```bash
go run ./cmd/heapcheck packs
```

The server keeps one in-memory copy of the bank (string-interned, compact indexes). On boot it loads from Supabase when credentials are set, otherwise from `packs/*.json` / `*.json.gz`.

### Data sources and licenses

Documented in each pack’s `attribution` field and summarized here:

| Source | License / terms | Used for |
| --- | --- | --- |
| [Wikidata](https://www.wikidata.org/) | CC0 | Large classes (actors, musicians, games, books, companies, compounds, species, songs, albums, rivers, mountains, brands, …) via paginated SPARQL; English `skos:altLabel` aliases; sitelinks for rarity |
| [Wikimedia Pageviews API](https://wikimedia.org/api/rest_v1/) | public API terms | Optional rarity signal for smaller Wikidata builds |
| [GeoNames](https://www.geonames.org/) `cities5000` | [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/) | Cities / towns (population → rarity) |
| [IMDb non-commercial datasets](https://developer.imdb.com/non-commercial-datasets/) | IMDb non-commercial license | Films and TV shows (`title.basics` + `title.ratings`) |
| [Open Food Facts](https://world.openfoodfacts.org/) ingredient taxonomy | ODbL / CC-BY-SA | Food ingredients |
| [Krillion](https://krillion.io) (+ archive via krillion-game.com) | **Used with permission** (CEO) | Full prompt/answer banks Krillion exposes |
| REST Countries / PokeAPI | their public terms | Optional membership checks on legacy `packs/defs` |
| MusicBrainz / Wikipedia | CC0 / CC BY-SA via Wikidata sitelinks | Covered indirectly through Wikidata entities that cite them |

Answer variants also synthesize short forms from source labels: drop `The`, titles/middle names, parenthetical disambiguators, `&`/`and`, small number words, and accent-free forms. Case-variant labels fold into one entity (first spelling wins; others become aliases) so Supabase upserts do not conflict.

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
# Upserts are chunked (~300 KB JSON per RPC) to avoid statement timeouts.
SUPABASE_URL=https://<ref>.supabase.co \
SUPABASE_DB_URL='postgresql://...' \
  make packbuild-deep

SUPABASE_URL=https://<ref>.supabase.co \
SUPABASE_DB_URL='postgresql://...' \
  make packbuild-krillion
```

Without those variables, packbuild only writes JSON and logs that the upsert was skipped. If Supabase is configured but the bank has no prompts, the server logs that and uses the JSON packs. If it is configured and the query fails, the process exits.

Fly (do not put the values in `fly.toml`):

```bash
flyctl secrets set \
  SUPABASE_URL=https://<ref>.supabase.co \
  SUPABASE_DB_URL='postgresql://...'
# or SUPABASE_SERVICE_ROLE_KEY=... instead of SUPABASE_DB_URL
```

## Tests

```bash
make test
```

## Deploy

### Go server → Fly.io (`krill-billion`, region `sjc`)

Multi-stage `Dockerfile` bakes `packs/` into the image (decompresses any `*.json.gz`). Listens on `$PORT` (default 8080). `/healthz` for checks. VM memory is **1 GB** (see `fly.toml`) so the deep bank can load without OOM.

```bash
flyctl launch --no-deploy   # or use existing fly.toml
flyctl secrets set FRONTEND_ORIGIN=https://YOUR_VERCEL_APP.vercel.app
flyctl deploy
```

### Frontend → Vercel

Set the public server URL so the browser talks to Fly (not localhost rewrites):

```
NEXT_PUBLIC_SERVER_URL=https://krill-billion.fly.dev
```

See `web/.env.example`. Deploy the `web/` directory as a Next.js app. Ensure Fly `FRONTEND_ORIGIN` matches the Vercel URL (comma-separated if you have preview + prod).

## CI

GitHub Actions runs `go vet` / `go test` and `npm ci` / lint / build under `web/`.
