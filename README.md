# E-SP-Line2

Platform adapter management system for e-commerce customer service.

**English** · [简体中文](README.zh-CN.md)

[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8)](backend/go.mod)
[![React](https://img.shields.io/badge/react-18-61DAFB)](web/package.json)
[![Protocol](https://img.shields.io/badge/protocol-v3-8A2BE2)](docs/developer-guide/protocol-v3.md)

[Quick start](docs/user-guide/getting-started.md) · [Adapters](docs/user-guide/adapters.md) · [Routing](docs/user-guide/routing.md) · [API reference](docs/developer-guide/api-reference.md) · [Protocol v3](docs/developer-guide/protocol-v3.md)

## Overview

E-SP-Line2 is an adapter management system built on a platform-oriented architecture. It ingests messages from multiple e-commerce platforms (Xianyu, Taobao and others), routes them and delivers them downstream. The system uses a four-layer architecture, asynchronous message handling, multiple availability safeguards and multi-language internationalization.

## Features

- **Platform management**: add multiple e-commerce platforms dynamically
- **Adapter architecture**: one adapter contract and one lifecycle for every platform
- **Asynchronous message handling**: an event-bus-backed message flow
- **Multiple availability safeguards**: primary/standby leases, fault isolation and automatic recovery
- **Multiple languages**: localized front end and localized adapter manifests
- **WebSocket communication**: real-time message push and receive

## Tech stack

### Backend

- **Language**: Go 1.22+
- **Framework**: Gin
- **Database**: GORM + SQLite/PostgreSQL
- **Cache**: Redis
- **WebSocket**: Gorilla WebSocket

### Front end

- **Framework**: React 18 + TypeScript
- **Build tool**: Vite
- **UI**: Tailwind CSS + shadcn/ui
- **Internationalization**: i18next
- **Routing**: React Router DOM

## Project structure

```
E-SP-Line2/
├── backend/              # Go backend service
│   ├── cmd/             # Command-line tools
│   ├── internal/        # Internal packages
│   │   ├── config/     # Configuration
│   │   ├── handler/    # HTTP handlers
│   │   ├── middleware/ # Middleware
│   │   ├── models/     # Data models
│   │   ├── repository/ # Data access layer
│   │   ├── server/     # Server
│   │   └── service/    # Business logic
│   ├── pkg/            # Shared packages
│   └── config/         # Configuration files
├── web/                # React front end
│   ├── src/
│   │   ├── components/ # Components
│   │   ├── pages/      # Pages
│   │   ├── locales/    # Locale resources
│   │   └── i18n.ts     # i18n setup
│   └── package.json
└── docs/               # Documentation
```

## Quick start

### Requirements

- Go 1.22+
- Node.js 20.11+
- pnpm or npm
- Redis (optional)

### Run the backend

```bash
cd backend

# Install dependencies
make deps

# Run the service
make run
```

The backend starts on `http://localhost:8080`.

### Run the front end

```bash
cd web

# Install dependencies
pnpm install

# Start the dev server
pnpm dev
```

The dev server starts on `http://localhost:3000`.

## API

### Auth

- `POST /api/v1/auth/register` - Register a user
- `POST /api/v1/auth/login` - Log in
- `GET /api/v1/auth/me` - Get the current user

### Platforms

- `GET /api/v1/platforms` - List platforms
- `GET /api/v1/platforms/:id` - Get a platform

### Adapters

- `GET /api/v1/adapters` - List adapters
- `POST /api/v1/adapters` - Create an adapter
- `GET /api/v1/adapters/:id` - Get an adapter
- `PUT /api/v1/adapters/:id` - Update an adapter
- `DELETE /api/v1/adapters/:id` - Delete an adapter
- `POST /api/v1/adapters/:id/start` - Start an adapter
- `POST /api/v1/adapters/:id/stop` - Stop an adapter

### WebSocket

- `ws://localhost:8080/ws/adapter?instance_id=xxx` - Adapter WebSocket
- `ws://localhost:8080/ws/app?app_id=xxx` - Application WebSocket

The full contract lives in the [API reference](docs/developer-guide/api-reference.md).

## Architecture

### Four layers

1. **Platform layer**: defines a business platform type (Xianyu, Taobao and others)
2. **Adapter package layer**: the integration for one platform, with a version and capability declarations
3. **Adapter instance layer**: the runtime configuration for a real account or shop
4. **Adapter session layer**: the current online connection state of an instance

### Message flow

```
Platform WebSocket → Adapter instance → Event bus → Core routing → Downstream application
                                                        ↓
                                                  Outbound command queue
                                                        ↓
                                                  Adapter instance → Platform send
```

The wire format is documented in [Protocol v3](docs/developer-guide/protocol-v3.md).

## Development

### Add a platform

Platforms are data. Add a column to `matrix.json` and an entry to the `META`/`CONFIG` tables in `scripts/gen_platformspec.py`, then regenerate:

```bash
python3 scripts/gen_platformspec.py matrix.json backend/internal/platformspec
python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
```

The registry feeds the adapter manifests, the capability matrix and the public `GET /api/v1/platforms/spec` endpoint, and built-in adapters are synced into the database on startup. Details: [adapter development guide](docs/developer-guide/adapter-development.md).

### Write an adapter

Follow the message field contract in [message fields](docs/developer-guide/message-fields.md) and the [adapter development guide](docs/developer-guide/adapter-development.md). Capability support levels gate what a runtime must implement. The upstream projects — `Spider_XHS`, `DouYin_Spider`, `TiktokApis`, `BilibiliApis`, `KuaiShou-Spider`, `WeiboApis`, `XianYuApis`, `TaoBaoApis`, `JdApis`, `XApis` — are the reference implementations; their platform metadata was migrated from [catbus](https://github.com/cv-cat/catbus) ([migration notes](docs/developer-guide/catbus-migration.md), [notice](NOTICE-catbus.md)).

### Docs

- [Platform capability matrix](docs/developer-guide/platform-capabilities.md) — per-platform support and limitations
- [Message fields](docs/developer-guide/message-fields.md) — every field, type and nullability
- [API reference](docs/developer-guide/api-reference.md)
- [Pull request QA](docs/developer-guide/pr-qa.md) — reviewer checklist and merge gate

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) before opening a pull request.

## Community

- [Contributing](.github/CONTRIBUTING.md)
- [Code of conduct](.github/CODE_OF_CONDUCT.md)
- [Security policy](.github/SECURITY.md)

## Contact

Questions and suggestions go to [GitHub Issues](https://github.com/E-SPower/E-SP-Line2/issues).

## License

Released under the [Apache License 2.0](LICENSE) plus the additional terms in [LICENSE-CUSTOM.md](LICENSE-CUSTOM.md), which must be read together. The project is also offered under the [MIT license](LICENSE-MIT) for consumers who prefer it; where the two differ for your use, say which one you are relying on.
