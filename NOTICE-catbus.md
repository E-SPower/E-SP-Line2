# catbus attribution

The platform adapter metadata, capability matrix and message field definitions in this repository are ported from [catbus](https://github.com/cv-cat/catbus), which is MIT licensed. The original licence follows in full.

The upstream repositories that catbus itself tracks (`src/platforms/<p>/UPSTREAM`) are recorded on `platformspec.Platform.Upstream`; each of them has its own licence and usage terms.

See `docs/developer-guide/catbus-migration.md` for what moved.

```
MIT License

Copyright (c) 2026 cv-cat

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## What was ported

| Content | Original location | Location here |
|---|---|---|
| Platform registry, login methods, config fields, capability matrix, inline limits | `src/platforms/*/index.ts`, `docs/capabilities.md` | `backend/internal/platformspec`, `docs/developer-guide/platform-capabilities.md` |
| Normalized types and field rules | `src/core/schemas.ts`, `AGENTS.md` 6.2 | `backend/internal/messagefields`, `docs/developer-guide/message-fields.md` |
| Output envelope, error codes, paging, exit codes | `docs/guide/output.md`, `src/core/errors.ts` | `docs/developer-guide/message-fields.md` |
