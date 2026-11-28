# Krillion reveal import

Dated JSON files here are pulled from [Krillion](https://krillion.io) (today’s `/api/reveal`) with archive backfill from krillion-game.com’s answer API. Used with permission from Krillion’s CEO — copy all prompts/answers Krillion exposes.

Regenerate (keeps prompts with ≥ 10,000 answers; writes `packs/krillion-v1.json` + `.gz`):

```bash
make packbuild-krillion
# or:
go run ./cmd/packbuild --fetch-krillion --krillion-import packs/import/krillion --out packs/krillion-v1.json
```

Raw `*.json` day files are gitignored. Built `krillion-v1.json` is also gitignored when large; prefer the `.gz` or rebuild before deploy/upsert.
