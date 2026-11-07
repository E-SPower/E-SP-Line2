# Pull request QA

Reviewer-facing companion to [`.github/CONTRIBUTING.md`](../../.github/CONTRIBUTING.md) §6. It says what a merge needs; this says how to check it.

The goal is a green CI that means something. A job that passes for the wrong reason is worse than no job, because it moves the real check into a reviewer's head — and a reviewer's head is not always available.

## Merge gate

A PR merges when all of these hold:

1. **CI is green** on the head commit. Every job in [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) — `backend`, `frontend`, `generated`, `format-lint`. Green on an older commit does not count; the newest push must be the one that ran.
2. **A code owner approved** the areas it touches ([`CODEOWNERS`](../../.github/CODEOWNERS)). Push access to `main` is not a substitute.
3. **The PR body answers the template.** "What changed", "why", "how it was verified" filled in, protocol-v3 impact answered explicitly, risk and rollback named.
4. **No unresolved review thread**, no `do-not-merge` label, not a draft.
5. **The change is reviewable in one sitting.** If the diff cannot be read in the reviewer's time box, ask for a split rather than reading it badly.

No review approval survives a force-push to a new base unless the diff is unchanged. That is why CONTRIBUTING §6 asks you not to force-push after review starts.

### Two things that do not work until a human sets them up

Both fail **silently**, which is the reason they are called out here rather than in a setup ticket nobody reads:

- **`CODEOWNERS` names one handle, `@TyperBody`.** A handle or team that does not have access to the repository matches nothing and GitHub requests nobody — the PR just looks owned. If the maintainer set grows, replace every entry in the file in one commit and confirm a test PR actually requests a reviewer. Do not add a team name that has not been created.
- **The labeler will not run on the PR that introduces it.** It triggers on `pull_request_target`, which reads the *base* branch, and the base has no workflow yet. The first PR is unlabelled; check `.github/labels.yml` was applied to the repository (`area:*`, `protocol:breaking`, `do-not-merge`) before relying on the labels for triage.

### Exceptions

| Situation | What is required |
|---|---|
| CI is broken on `main` for a reason unrelated to the PR | Merge on a maintainer's written say-so in the PR thread, and open the issue for the broken job. Do not disable the job in the same PR. |
| Docs-only change under `docs/` | `backend` and `frontend` still run; they are cheap. Do not wave them off — a docs PR can carry a code block that no longer compiles. |
| Version bump / dependency update | Bundled alone, with the upstream changelog linked and `dependencies` label. Never mixed into a feature. |
| Hotfix to a released tag | Branch from the tag, not `main`, and open the backport PR to `main` in the same sitting. |
| Adding a new job | Pin every action to a major tag and every toolchain version explicitly. An unpinned tool is a future red build nobody owns. |
| Dependency is unavailable to the linter | Fix it in the PR that needs it. Disabling a linter for the whole repo to unblock one PR is a separate commit with the reason in the config, as `errcheck` has in `.golangci.yml`. |

## What the reviewer checks

Ordered by the cost of missing it. Stop at the first problem, do not review the rest against a flawed base.

### 1. Correctness

- Does the code do what the description claims, for the normal path **and** the failure path?
- Error returns carry context (`fmt.Errorf("...: %w", err)`) — not logged and swallowed.
- HTTP status codes match the failure: 400 for bad input, 401 vs 403 not conflated, 404 vs 409 on a state conflict.
- Nil maps, nil slices and zero values survive: a missing adapter, an empty `config` blob, a user with no instances.
- Anything that writes to the database is inside a transaction when it is more than one statement.
- A handler that reads request input validates it; `BindJSON` alone is not validation.

### 2. Layering

- The direction stays `handler → service → repository`. A handler importing GORM or the message layer importing a handler is a defect, not a style nit.
- A new adapter follows [adapter-development.md](adapter-development.md); it does not reach into `service` internals.
- `platformspec` and the capability doc are generated — an edit there without a `matrix.json` change gets reverted by the next generator run.

### 3. Tests

- A fix has a regression test that fails before the fix. Verify the claim: read the test, check the assertion actually exercises the bug.
- A feature has a test. `httptest` for handlers, a fixture payload through the normalizer for an adapter.
- **No test calls a real Taobao or Xianyu endpoint.** A test with a live network dependency is a test that fails in a week.
- A test that only asserts "no error" and not the returned value is decoration.

### 4. Security

This is the area most often waived because "it is small". It is also the one whose mistakes cannot be undone after a release.

- Nothing new skips `middleware.AuthRequired`; a public route is deliberate and named.
- A query is scoped to the owning user. Another user's instance id is still reachable unless the repository filters.
- Adapter `config` blobs (they hold platform cookies) are not returned unmasked in a list endpoint.
- No secret, token, cookie or default JWT secret in the diff, the fixtures, or the logs pasted into the PR.
- `CheckOrigin`, CORS and `Allow-Credentials` are consistent with each other — a `*` origin plus credentials is a browser-side no-op.
- File paths from a request are joined and normalised, not concatenated.

Report a vulnerability that predates the PR through [SECURITY.md](../../.github/SECURITY.md) rather than dragging it into the review, unless the PR makes it worse.

### 5. Docs and protocol sync

- A changed endpoint updates `docs/developer-guide/api-reference.md`.
- A changed field updates `docs/developer-guide/message-fields.md`.
- An envelope, chain or signing change updates `docs/developer-guide/protocol-v3.md`, bumps the version, and carries the `protocol:breaking` label.
- A changed user-facing string has both locale keys (`zh-CN.json` and `en-US.json`), and no existing key was deleted.
- A UI change respects [web/UI-SPEC.md](../../web/UI-SPEC.md): tokens, not hand-rolled `bg-blue-600`.

## Commands to run by hand

CI runs these. Run them locally when CI is red for a reason you do not trust, or when you suspect the diff only passes because of cached state.

```bash
# from the repository root, once:
export CGO_ENABLED=1

make test          # CGO_ENABLED=1 is mandatory, see below
go vet ./...                  # the Go module root is the repository root
make lint                     # golangci-lint, config in .golangci.yml
gofmt -l backend              # must print nothing

# `vite build` writes a `vite.config.ts.timestamp-*.mjs` next to the config, so run
# it in a writable tree. On a read-only mount it dies with EROFS before bundling;
# CI checks out writable and is unaffected.
(cd web && npx tsc --noEmit && npm run lint && npm run build)

python3 scripts/check_generated.py matrix.json   # only if matrix.json or a generator changed
```

### The two lint ratchets

Both lint surfaces pass on `main` today and are wired to fail on a *new* problem, not on the accumulated old ones. Neither budget is a licence to add debt.

- **`golangci-lint`** runs with `errcheck` disabled. The tree has 22 unchecked returns in non-test code — mostly `conn.Conn.Close()`, `resp.Body.Close()`, `rand.Read` and `logger.Sync()`, calls whose failure has no recovery path — plus 3 across the test files (`message/dispatcher_test.go` 2, `adapter/main_test.go` 1). Re-enable it in a PR that fixes them; the reason lives in the config file, in place.
- **eslint** stays clean because `react-refresh` is an error and the legitimate non-component exports are named in the config, while `@typescript-eslint/no-explicit-any` runs against `--max-warnings` in `web/package.json`. A new `any` over the budget is red; remove one and lower the budget in the same PR.

Both live in the `format-lint` job. There is no separate lint job to skip: a red lint fails the whole gate.

### Why `CGO_ENABLED=1` and gcc

`gorm.io/driver/sqlite` wraps `mattn/go-sqlite3`, which is cgo. With `CGO_ENABLED=0` the driver still compiles — against a stub whose `Open` fails at runtime — so every test that opens a database fails with a connection error that looks like a code defect. It is not:

```text
Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.
```

The `backend` job installs `gcc` and sets `CGO_ENABLED=1` for exactly this. If a contributor reports "the tests pass locally but CI hates me", check the cgo setup before reading their diff. Conversely, a reviewer who runs `go test ./...` with cgo off will reject a working PR — that is the failure mode this section exists to prevent.

## Requesting changes

- Say what is wrong and what would fix it. "This is not how we do it" costs a round trip.
- Separate **blocking** from **nice to have**; mark the actionable ones with a suggestion block so the fix is one click.
- One concern per comment. Do not bury a blocker in a paragraph about naming.
- Link the reference — the doc section, the precedent in `main`, the failing line — instead of restating policy.
- Re-review after a fix by reading the fix, not the whole diff again, unless the fix touched the base.

## After merge

- Delete the branch.
- If the PR closed an issue, confirm it auto-closed; if the PR should have but the keyword was missing, close it manually.
- Watch `main` for the next CI run. A merge that breaks `main` is a `do-not-merge` on every other open PR within minutes.
