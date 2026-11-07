# Getting started

## Introduction

E-SP-Line2 is a platform-based adapter management system that supports message ingestion, routing, and delivery for multiple platforms (Xianyu, Taobao, and others). The system uses a four-layer architecture and supports asynchronous message processing, multiple availability safeguards, and multilingual internationalization.

## System requirements

- Go 1.22+
- Node.js 20.11+
- Python 3.9+ (for adapters)
- Redis (optional, for caching)
- PostgreSQL or SQLite

## Installation steps

### 1. Clone the project

```bash
git clone <repository-url>
cd E-SP-Line2
```

### 2. Backend installation

```bash
go mod download
```

### 3. Frontend installation

```bash
cd web
npm install
```

### 4. Configure the database

Edit `config/config.yaml`:

```yaml
database:
  driver: sqlite  # or postgres
  dsn: ./data/e-sp-line2.db
  
server:
  host: 0.0.0.0
  port: 8080
```

### 5. Start the services

```bash
# Start the backend
go run main.go

# Start the frontend (in a new terminal)
cd web
npm run dev
```

Visit http://localhost:3000 to use the management interface.

## First use

### 1. Create an administrator account

You need to create an administrator account the first time you access the system.

### 2. Review the platforms

The 12 active platforms (Xiaohongshu, Douyin, TikTok, Bilibili, Kuaishou, Weibo, Xianyu, Taobao, JD, X, YouTube, 12306) are seeded from the platform registry on first start, so the Platforms page already lists them. Jimeng, Maoyan Shows and M-Team are registered as inactive until they ship.

Each platform's supported commands and limitations are in the [capability matrix](../developer-guide/platform-capabilities.md).

### 3. Create an adapter instance

Create an adapter instance on the Adapters page:

1. Select a platform
2. Enter the account information (Cookie, etc.)
3. Configure routing rules

### 4. Start the adapter

Click the "Start" button to start the adapter instance; the system connects to the platform automatically and begins receiving messages.

## Core concepts

### Four-layer architecture

1. **Platform layer**: defines a business platform type (Taobao, Xianyu, etc.)
2. **Adapter package layer**: the adapter implementation for a specific platform, with a version and capability declarations
3. **Adapter instance layer**: the runtime configuration of a real account or shop
4. **Adapter session layer**: the current online connection state of an instance

### Message flow

```
Platform WebSocket → adapter instance → event bus → core routing → downstream application
                                              ↓
                                        outbound command queue
                                              ↓
                                        adapter instance → platform send
```

### Routing rules

Routing rules determine how messages are dispatched to downstream applications:

- **Platform matching**: match by platform type
- **Instance matching**: match by a specific instance
- **Condition matching**: match by message content, sender, and other conditions

## FAQ

### Q: The adapter fails to start?

A: Check the following:
1. Whether the Cookie is valid
2. Whether the network connection works
3. Check the backend logs for detailed error information

### Q: Messages are not routed to the application?

A: Check:
1. Whether any routing rules are configured
2. Whether the routing rules are enabled
3. Whether the downstream application is online

### Q: How do I add a new platform?

A: See "Adapter development guide" in the developer documentation.

## Next steps

- Read [Adapter configuration](./adapters.md) for detailed configuration
- Read [Routing rules](./routing.md) to learn about message routing
- Read [Troubleshooting](./troubleshooting.md) to solve common problems
