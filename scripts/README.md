# Scripts

| Script | Purpose |
|---|---|
| `gen_platformspec.py` | Builds `backend/internal/platformspec` from `matrix.json`. |
| `gen_capabilities.py` | Builds `docs/developer-guide/platform-capabilities.md` from `matrix.json`. |
| `check_generated.py` | Fails when either generated output is stale. |

`matrix.json` at the repository root is the capability matrix export: `platforms` (column order), `login` (login table), `commands` (support per platform) and `notes` (inline limitations). It comes from the catbus `docs/capabilities.md` migration; see `docs/developer-guide/catbus-migration.md`.

Regenerate after any matrix change:

```bash
python3 scripts/gen_platformspec.py matrix.json backend/internal/platformspec
python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
python3 scripts/check_generated.py matrix.json
```

`go test ./internal/platformspec/` runs the checker through `TestGeneratedFilesAreCurrent`. Generated Go files go through `gofmt` automatically.
