# catbus adapter migration

Porting the platform adapters of [catbus](https://github.com/cv-cat/catbus) (MIT) into E-SP-Line2, together with documentation for every message field they produce.

## Why port it

E-SP-Line2 had two adapters, Taobao and Xianyu, with capabilities written out as string literals and field meanings buried in code. catbus reaches 15 platforms with one command vocabulary and one JSON shape, and its docs state where every field comes from and when it is permanently null. The two message models line up:

| catbus | E-SP-Line2 |
|---|---|
| `{ok, data, page, error}` envelope | V3 envelope + `MessageEnvelope` |
| normalized `Message` / `Conversation` / `Notice` / `Event` | `InboundMessage` + `MessageChain` elements |
| capability matrix (✓ ◐ ○ —) | adapter `capabilities` |
| `NOT_IMPLEMENTED` / `UNSUPPORTED` | the same error codes, exit 4 / 2 |
| `--raw` | `raw_message` |

## What moved

- **Platform registry** `backend/internal/platformspec`: 15 platforms, per-command support for 102 commands, login methods, config fields, capability lists and inline limitation notes, all generated from the capability matrix.
- **Adapter package** `backend/internal/adapter`: `BuiltinAdapterInfo` translates a registry entry into `AdapterInfo`, a config JSON Schema, both locales, and the operation and security policies. `RegisterBuiltinAdapters` registers the 12 active platforms; the three upcoming ones are registered as `inactive`.
- **Message field contract** `backend/internal/messagefields`: fields, types, nullability, enums and meaning for 17 normalized objects and 11 message element kinds, served by `GET /api/v1/message-fields`.
- **Startup sync** `backend/internal/adapter/sync.go`: writes the active platforms and their adapter packages into the database, idempotently, so the existing platform and adapter APIs work without hand-entry.
- **Docs**: this page, [message-fields.md](./message-fields.md), [platform-capabilities.md](./platform-capabilities.md).

## Generated, do not hand-edit

The capability matrix is the single source of truth. `matrix.json` is an export of it, and two scripts grow code and docs from that:

```bash
# regenerate the platform registry
python3 scripts/gen_platformspec.py matrix.json backend/internal/platformspec
# regenerate the capability matrix doc
python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
```

The `platforms`, `login`, `commands` and `notes` keys of `matrix.json` mirror the matrix's column order, login table, command table and inline notes. To change a capability, change `matrix.json` (or re-export it from catbus) and run both commands. `python3 scripts/check_generated.py matrix.json` reports generated files that were not refreshed (see `scripts/README.md`); `TestGeneratedFilesAreCurrent` runs it as a test.

## How support levels reach the runtime

| Matrix symbol | `Support` | Runtime result |
|---|---|---|
| ✓ | `implemented` | runs |
| ◐ | `partial` | runs; the limitation is in `Platform.Notes` |
| ○ | `planned` | `NOT_IMPLEMENTED`, exit code 4, do not retry |
| — | `unsupported` | `UNSUPPORTED`, exit code 2 |

`platformspec.Supports(platform, resource, action)` is the only gate; `platformspec.Canonical` resolves aliases (`tb`, `bili`, `dy`) to canonical ids first.

## Runtime layout

catbus's "one platform, one set of files" splits in two here:

- Static metadata always goes back to `platformspec` and is not copied per platform.
- The parts that make requests (client, api, normalize, resolve) live in a per-platform directory, see [adapter development](./adapter-development.md).

| catbus file | Responsibility | E-SP-Line2 location |
|---|---|---|
| `web/client.ts` | session, cookies, signing | `adapters/<platform>/client` |
| `web/api.ts` | one function per upstream method | `adapters/<platform>/api` |
| `web/normalize.ts` | raw object → normalized type | `adapters/<platform>/normalize`, fields per [message-fields.md](./message-fields.md) |
| `web/resolve.ts` | IDs, links, short links and `me` all work as arguments | `adapters/<platform>/resolve` |
| `UPSTREAM` | upstream repo and commit | `platformspec.Platform.Upstream` |

## Where the two differ

- catbus is a Node CLI, so adapter `runtime_type` becomes `node`; the old value, `python`, did not match the implementation.
- catbus runs a command directly; E-SP-Line2 goes through an adapter instance. Capabilities therefore list `<resource>.<action>` and mark planned ones `.planned`, while keeping the old top-level words (`receive_message`, `send_text`) so existing consumers do not break.
- catbus's exit codes do not apply to E-SP-Line2's HTTP API. The error codes carry over; exit codes only matter on the adapter side.
- 12306 needs no login in catbus, so its config field list is empty here.
- Jimeng needs macOS and the vendor `dreamina` CLI, so it stays upcoming.

## Licence

catbus is MIT. The platform metadata, capability matrix and field definitions keep their provenance; the terms are in `NOTICE-catbus.md` at the repository root.
