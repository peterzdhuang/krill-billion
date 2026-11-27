# Krillion reveal import

Dated JSON files here are pulled from [krillion.io](https://krillion.io) (today’s `/api/reveal`) with archive backfill from krillion-game.com’s answer API. Used with permission from Krillion.

Regenerate:

```bash
make packbuild-krillion
```

Raw `*.json` day files are gitignored; the built `packs/krillion-v1.json` is committed so the server works without Supabase or a re-fetch.
