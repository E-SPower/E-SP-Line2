# Docs links

The in-app Documentation page links to the Markdown files under `docs/` at the repository root, served under `/docs`.

- `vite.config.ts` mounts a `serve-docs` middleware so the dev server answers `/docs/**` with the file contents.
- A production build must copy `docs/` next to `dist/`, or put a reverse proxy in front of `/docs`. The page opens each link in a new tab, so a plain static host works once `docs/` is served.
- All links are relative to the repository root, so renaming a doc file means updating `web/src/pages/Docs.tsx` and the `docs.link*` keys together.

## Locale keys

`docs.link*` keys exist in both `en-US.json` and `zh-CN.json`; the two files must stay in sync (a test in the web build compares them).
