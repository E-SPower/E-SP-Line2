<!--
Keep the headings. Delete nothing that applies; write "none" plus a reason where
it genuinely does not. A reviewer should be able to name the changed files after
reading "What changed" — if not, the paragraph is too vague.

One concern per PR (CONTRIBUTING §6). Unrelated cleanup goes in its own PR.
The reviewer checklist and merge gate are in docs/developer-guide/pr-qa.md.
-->

## What changed

<!-- The files and the behaviour, not the journey. -->

## Why

<!-- The problem, the issue it closes, and why this shape of fix. "Closes #123". -->

## How it was verified

<!--
Paste the commands and their actual result. "Local tests pass" alone is not
verification — say which test, which page, which path. CI (`.github/workflows/ci.yml`)
runs the same commands offline; a green run here should predict a green CI.
For a UI change, attach before/after screenshots or a short recording.
-->

- [ ] `CGO_ENABLED=1 go test ./...` from the repository root (cgo + gcc are required, see pr-qa.md)
- [ ] `go vet ./... && go build ./...`
- [ ] `gofmt -l .` printed nothing
- [ ] `cd web && npm ci && npx tsc --noEmit && npm run build`
- [ ] `cd web && npm run lint`
- [ ] `python3 scripts/check_generated.py matrix.json` (if `matrix.json` or a generator changed)

## Docs affected

<!--
README, docs/user-guide/*, docs/developer-guide/*, web/UI-SPEC.md — or "none, and
why". A changed endpoint, field, config key or user-facing string is a doc change
even when the code is the only diff.
-->

## Protocol v3 impact

<!--
Explicit answer required. Does this touch the envelope, the message chain, the
signing, or any field in docs/developer-guide/message-fields.md? "No" is a valid
answer; blank is not.

If yes: version bump, the `protocol:breaking` label when a consumer breaks, and
the protocol-v3.md / message-fields.md update in the same PR.
-->

## Risk and rollback

<!--
What can break, who it hits, how it surfaces in production, and how to undo it.
A config or schema change needs a forward path too: a rollback that loses the new
column is not a rollback. "Low risk" alone is not an answer — say why it is low.
-->

## Checklist

- [ ] One concern per PR, rebased on `main` (not merged `main` in)
- [ ] Tests cover the change — a fix has a regression test that fails without it
- [ ] No unit test calls a real Taobao/Xianyu endpoint
- [ ] No secret, cookie, token or default JWT secret in the diff or the pasted logs
- [ ] New user-facing strings are in both `web/src/locales/zh-CN.json` and `en-US.json`, none deleted
- [ ] Commits follow Conventional Commits (CONTRIBUTING §4); no force-push after review starts
- [ ] `do-not-merge` added if this should not merge yet
