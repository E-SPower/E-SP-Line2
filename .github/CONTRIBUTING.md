# Contributing to E-SP-Line2

Thanks for wanting to make E-SP-Line2 better. This guide covers setting up a dev environment, the conventions for code and commits, and how to get a change merged. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) first — it applies to every issue, PR and review.

## 1. Before you start

- Search the [open issues](https://github.com/E-SPower/E-SP-Line2/issues) before you file one; a duplicate splits the discussion.
- For anything larger than a bug fix or a small docs patch, open an issue describing the problem and the shape of your fix first. It saves you from building something the project does not want.
- Report security problems privately — see [SECURITY.md](SECURITY.md), not a public issue.

## 2. Development environment

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.22+ | build, test and run the backend |
| Node | 20.11+ | build and run the web front end (`web/package.json` `engines`; CI reads it) |
| npm or pnpm | current | front-end dependencies |
| Git | any recent | |

Component versions matter: `go.mod` and `web/package.json` pin what maintainers and the toolchain use. Do not bump a dependency just because — bump it in its own PR with a reason.

### Backend

```bash
git clone https://github.com/E-SPower/E-SP-Line2.git
cd E-SP-Line2/backend
go mod download
go run main.go   # reads ./config/config.yaml, then ./config.yaml, then /etc/e-sp-line2
```

The API listens on `:8080`. Before you expose it beyond localhost, change `jwt.secret` in the config — see [SECURITY.md](SECURITY.md).

```bash
make build     # build to bin/e-sp-line2
make test      # go test ./... — needs CGO_ENABLED=1 and gcc, see below
make fmt       # go fmt ./...
make lint      # golangci-lint, config in .golangci.yml
```

`make lint` is not optional: it is a CI step. The linter is not vendored, so install the pinned version once:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
# ~/go/bin must be on PATH
```

Backend tests need a C toolchain: `gorm.io/driver/sqlite` is cgo, and with cgo off the driver compiles against a stub that fails at `Open`, so the database tests die with an error that looks like a code defect. Run `export CGO_ENABLED=1` (and install `gcc`) before `make test`.

### Front end

```bash
cd web
npm install
npm run dev    # vite on :3000, proxying /api and /ws to :8080
```

```bash
npm run build   # tsc then vite build
npm run lint    # eslint
```

Read [web/UI-SPEC.md](../web/UI-SPEC.md) before you touch a page. Every colour comes from a token; every state uses the shared component (`Badge`, `Card`, `Skeleton`, `EmptyState`, `ErrorState`, `PageHeader`, `FormField`). Do not hand-roll `bg-blue-600` into a page, and do not add a dependency without a reason.

### Database

GORM `AutoMigrate` creates the schema on startup, so a fresh checkout needs nothing. Hand-written SQL migrations live in `migrations/` with their own [README](../migrations/README.md); note that `main.go` does not currently expose a `migrate` subcommand, so the SQL files are reference material until it does. Add a migration when you change a persisted model, and keep `IF NOT EXISTS` / `ON CONFLICT` so re-runs are safe.

## 3. Project layout

The Go module is the repository root (`go.mod`, `main.go`).

```
internal/
├── adapter/          # adapter registry, manager and bridge
├── adapters/         # embedded adapter bundles
├── adaptergateway/   # gateway client for the bridges
├── config/           # configuration loading
├── handler/          # HTTP and WebSocket handlers
├── message/          # event hub, router and dispatcher
├── middleware/       # auth, CORS and logging middleware
├── models/           # GORM models
├── protocol/v3/      # v3 envelope, message chain, signing
├── repository/       # data access
├── server/           # server wiring and routes
└── service/          # business logic
adapters/             # per-platform bridge implementations
cmd/esp-desktop/      # desktop shell
migrations/           # SQL migrations
pkg/logger/           # shared logging helpers
web/src/
├── api/              # typed API client
├── components/       # layout and UI component library
├── locales/          # zh-CN / en-US strings
└── pages/            # one file per route
docs/                 # user and developer guides
```

Backend layering is `handler → service → repository`; handlers do not touch GORM directly, and the message layer does not import handlers. New adapters follow the contract in [docs/developer-guide/adapter-development.md](../docs/developer-guide/adapter-development.md).

## 4. Conventions

### Code

- **Go**: [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Run `make fmt` and `make lint` before pushing — CI runs both, and `make lint` uses `.golangci.yml`.
- **TypeScript/React**: the repo's ESLint config (`web/.eslintrc.cjs`), `npm run lint` must be clean, and `tsc` must pass — `npm run build` runs it. The `--max-warnings` budget in `web/package.json` is a ratchet: a new `any` over it is red, and removing one means lowering the number in the same PR.
- **Comments and user-facing strings in English.** User-facing text goes through i18next; add the key to both `web/src/locales/zh-CN.json` and `en-US.json`, and never delete an existing key.
- Keep error returns wrapped with context (`fmt.Errorf("...: %w", err)`) rather than logged and swallowed.

### Commits

[Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`. Scope is the area touched (`adapter`, `router`, `web`, `docs`). Subject in the imperative mood, lowercase, no trailing period.

```
feat(adapter): add Pinduoduo adapter package

- implement the Pinduoduo message client
- add the product card message type
- cover the normalizer with unit tests

Closes #123
```

One logical change per commit. Do not commit `node_modules`, build output or a local database with real credentials.

### Branches

`feat/<short-name>`, `fix/<short-name>`, `docs/<short-name>` off `main`. Rebase on `main` rather than merging it in, so history stays linear.

## 5. Tests

- A new feature needs tests; a bug fix needs a regression test that fails without it.
- Backend: `go test ./...` (table-driven, `httptest` for handlers). Front end: whatever the page exercises — at minimum the API client and the reducer-ish logic around it.
- Do not call a real Taobao or Xianyu endpoint from a unit test. Fixture the raw payload and pass it through the normalizer.
- If a change alters the v3 envelope, message chain or signing, say so in the PR and update [docs/developer-guide/protocol-v3.md](../docs/developer-guide/protocol-v3.md).

## 6. Pull requests

- Keep a PR to one concern. Unrelated cleanup goes in its own PR.
- Describe what changed, why, and how you verified it. Link the issue.
- Update the docs the change invalidates — README, the matching `docs/` page, or the UI spec.
- CI must be green. `.github/workflows/ci.yml` runs the commands above, and `format-lint` is part of the gate: a red job is answered by fixing the code, not by disabling the job. A maintainer reviews for correctness, layering, test coverage and security; an unreviewed PR does not merge just because it is small.
- The full reviewer checklist, merge gate and exception process is [docs/developer-guide/pr-qa.md](../docs/developer-guide/pr-qa.md).
- Do not force-push after review starts unless asked; reviewers lose the diff.

## 7. Reporting bugs and proposing features

A good bug report has the version or commit, OS, Go/Node versions, exact reproduction steps, what you expected, what happened, and the relevant log lines with Cookies and tokens stripped. A good feature request states the problem first and the proposed solution second, plus alternatives you considered.

## 8. Sign-off

By contributing you agree your contribution is licensed under the terms in [LICENSE](../LICENSE) and [LICENSE-CUSTOM.md](../LICENSE-CUSTOM.md).

## 9. Getting help

Comment on the issue you are working on, or open a discussion. Do not @-mention a maintainer for a status nudge before a few days have passed.
