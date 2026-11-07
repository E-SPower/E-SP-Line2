# Security policy

## Supported versions

| Version | Security fixes |
|---|---|
| 1.0.x | Yes |
| < 1.0 | No |

E-SP-Line2 is a young project that moves fast, so fixes ship on the latest release only. If you built from source, follow the newest commit on `main`.

## Reporting a vulnerability

**Please do not report security problems in a public issue, pull request or discussion.**

Use GitHub Security Advisories instead:

**https://github.com/E-SPower/E-SP-Line2/security/advisories/new**

Please include:

- the affected version or commit, plus your OS and Go/Node versions;
- the kind of problem and its impact, for example a leaked credential, an adapter config readable by other users, a token written to the logs, arbitrary file writes, or command injection;
- reproduction steps or a proof of concept;
- a suggested fix, if you have one.

**Strip every Cookie, token, JWT secret and account name before you attach logs, screenshots or your SQLite database.** We will never ask you for real credentials.

Maintainers will work with you in the advisory on the timeline for confirmation, fix and disclosure. Please keep the details private until the fix ships.

## Scope

In scope, for example:

- credentials shown in API responses, logs or error messages where they should be masked;
- an adapter `config` blob (which holds a platform Cookie) readable by a user who does not own the instance;
- the API reachable without a valid token because an auth check is missing;
- a request sent anywhere other than the platform endpoint it targets;
- the JWT secret accepted in its default `change-this-secret-in-production` value in a deployment that never changed it;
- path traversal through upload or file handling;
- the WebSocket upgrade accepting a cross-origin connection that should be rejected, or an unauthenticated socket being accepted at all;
- a request body read without a size ceiling, or a connection held open without a header deadline.

Out of scope:

- security problems in the platform APIs themselves — report those to the platform;
- the upstream `TaoBaoApis` and `XianYuApis` projects — report those upstream (though a problem E-SP-Line2 inherits is still welcome here);
- platform risk control, account bans and captchas — those are usage risk, not a vulnerability here;
- scenarios where the attacker can already read and write files as your user.

## How E-SP-Line2 protects your credentials

### Backend

- Passwords are stored as bcrypt hashes, never in plaintext.
- API tokens are JWTs signed with HS256 and carry an expiry (`jwt.expiration`, 24 hours by default). Rotate `jwt.secret` in `backend/config/config.yaml` per deployment.
- The `/api/v1` surface except `/auth/register` and `/auth/login` sits behind `AuthRequired`. `AdminRequired` exists but is not wired into any route yet.
- Both WebSocket routes (`/ws/adapter`, `/ws/app`) require a valid JWT at the handshake (`?token=`) and reject an unlisted `Origin` before upgrading. The gateway routes (`/ws`, `/ws/adapter-gateway/...`) keep their own per-adapter `?key=`.
- CORS and the WebSocket `CheckOrigin` read one allow-list, `server.allowed_origins`. The accepted origin is echoed rather than sent as `*`, which is the only form where the accompanying `Access-Control-Allow-Credentials` has an effect.
- `http.Server` runs with header, read, write and idle timeouts, and every request body is capped at `server.max_body_bytes` (8 MiB default) — an unbounded body used to be read whole into memory.
- An adapter's `security` policy declares `sensitive_fields` (for Taobao and Xianyu, `cookie`) so a manifest can mark what must not be echoed back.

### What is not there yet

Be aware of these when you deploy, and please send a report if one of them bites you:

- The adapter instance `config` column stores its Cookie as plain JSON in the database, despite the `encrypted_fields` declaration. Treat the SQLite/PostgreSQL file as a secret.
- `AuthRequired` is still a placeholder: it checks the presence and `Bearer` shape of the header but does not yet verify the JWT signature, and it sets a fixed `user-123` identity. Do not expose the API to the internet and assume per-user isolation until that lands.
- The WebSocket handshake check proves the caller holds a valid token, not that it owns the `instance_id` / `app_id` it asks to bind. One logged-in user can still attach to another user's instance.
- The default `jwt.secret` is a placeholder. Startup logs a WARN while it is unchanged; change it before you deploy. It is not fatal, so a forgotten deployment still comes up — quietly forgeable.


### Front end

- Access tokens live in `localStorage` under the `auth_token` key and are attached as `Authorization: Bearer` by `src/api/client.ts`.
- The token is cleared on logout (the sign-out control in `src/components/Layout.tsx`). A `401` from the API is surfaced as an error but does not auto-clear the token, so a stale token lingers until the next sign-out.
- Adapter Cookies are pasted into the browser on the Adapters page and posted to the API, where they are stored in plain JSON. Any instance response (`GET /api/v1/instances...`) returns that config verbatim, so a Cookie is visible to anyone holding a valid token.

### Repository hygiene

- Never commit real Cookies, tokens or JWT secrets in code, fixtures or logs.
- `backend/data/e-sp-line2.db` is tracked and holds the seeded data of the dev instance — check it for real credentials before you push.
- `backend/bin/e-sp-line2` is a tracked build artifact; rebuild it rather than trusting the committed binary to match the source.
