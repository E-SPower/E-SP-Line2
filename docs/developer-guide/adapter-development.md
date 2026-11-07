# Adapter development guide

## Overview

This guide explains how to develop a new platform adapter for E-SP-Line2. An adapter talks to a specific e-commerce platform and implements message receiving and sending.

## Architecture overview

```
┌─────────────────┐     HTTP/gRPC     ┌─────────────────┐
│   Go Backend    │ ◄──────────────► │ Python Adapter  │
│                 │                   │                 │
│  Adapter Manager│                   │  Platform SDK   │
└─────────────────┘                   └─────────────────┘
```

## Adapter structure

An adapter has two halves: static metadata that is generated from the platform registry, and a runtime that is written by hand.

```
backend/internal/platformspec/     static metadata, generated from matrix.json
backend/internal/adapter/          AdapterInfo / manifest / registry, generated per platform
adapters/<platform>/               runtime: client, api, normalize, resolve
```

### 1. Adapter package (Adapter Package)

An adapter package is the metadata and configuration definition of an adapter.

```go
type AdapterInfo struct {
    ID              string            `json:"id"`
    PlatformID      string            `json:"platform_id"`
    Name            string            `json:"name"`
    Version         string            `json:"version"`
    RuntimeType     string            `json:"runtime_type"` // python, node, go
    ProtocolVersion string            `json:"protocol_version"`
    Capabilities    []string          `json:"capabilities"`
    ConfigSchema    map[string]interface{} `json:"config_schema"`
    I18n            map[string]I18nResource `json:"i18n"`
    Operations      OperationPolicy   `json:"operations"`
    Security        SecurityPolicy    `json:"security"`
    Status          string            `json:"status"`
}
```

### 2. Adapter instance (Adapter Instance)

An adapter instance is a concrete account or store configuration.

```go
type Instance struct {
    ID            string                 `json:"id"`
    AdapterID     string                 `json:"adapter_id"`
    PlatformID    string                 `json:"platform_id"`
    Name          string                 `json:"name"`
    Status        InstanceStatus         `json:"status"`
    Config        map[string]interface{} `json:"config"`
    Credentials   map[string]interface{} `json:"credentials"`
    ConnectedAt   *time.Time             `json:"connected_at,omitempty"`
    LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
    MessageCount  int64                  `json:"message_count"`
    ErrorCount    int64                  `json:"error_count"`
    LastError     string                 `json:"last_error,omitempty"`
}
```

## Development steps

### Step 1: Add the platform to the registry

Platforms are data, not hand-written code. Everything static about a platform — its name, item noun, aliases, upstream source, login methods, config fields, capability list and per-command support — lives in `backend/internal/platformspec`, generated from `matrix.json`:

```bash
python3 scripts/gen_platformspec.py matrix.json backend/internal/platformspec
python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
```

To add a platform, add a column to `matrix.json` and one entry to the `META`/`CONFIG` tables in `scripts/gen_platformspec.py`, then rerun both commands. Never edit the generated files under `platformspec/`.

The generator refuses nothing on its own, so follow these rules:

- `platforms` fixes the column order and must stay in sync with `META`.
- Command support uses exactly `implemented`, `partial`, `planned` or `unsupported`; omit a platform to mean `unsupported`.
- Every platform needs at least one of `Name`, `NameEN`, `Item`, `ItemEN`; `TestEveryPlatformHasMeta` enforces it.
- Only the known config field names are accepted (`TestConfigFieldsAreKnown`), so a genuinely new credential type also needs a `ConfigField` in `CONFIG` and a line in the capability matrix notes.

### Step 2: Adapter info is generated for you

`adapter.BuiltinAdapterInfo` turns a `platformspec.Platform` into the `AdapterInfo` the registry stores: it builds `ConfigSchema` from `ConfigFields`, both `I18n` locales from the platform name and login methods, `OperationPolicy` from the platform's reconnect defaults and `SecurityPolicy` from the credential fields. There is nothing to hand-write and nothing to register — `RegisterBuiltinAdapters` walks the active platforms.

The pieces you would otherwise hand-write:

| Field | Where it comes from |
|---|---|
| `ConfigSchema.required` | `ConfigField.Required` |
| `ConfigSchema.properties.<name>.default` | `ConfigField.Default` |
| `I18n.*.InstallGuide` | `LoginMethods` |
| `Security.SensitiveFields` / `EncryptedFields` | every non-integer config field |
| `Security.PermissionScopes` | derived from capabilities (`send_text` → `message:write`) |
| `Operations.HeartbeatInterval` / `ReconnectDelay` | config field defaults |
| `Status` | `active` for the active stage, `inactive` for upcoming |

Upcoming platforms are registered as `inactive`, so a fresh install does not try to connect them.

### Step 3: Implement the runtime

Write the platform's `client`, `api`, `normalize` and `resolve` in its own directory. Follow the message field contract in [message-fields.md](./message-fields.md): every field exists, values are `null`/`[]` when missing, IDs are strings, times are ISO 8601 with an offset, and `id` and `url` are always populated on emitted objects.

Implement only the commands the matrix marks `implemented` or `partial`. `planned` commands must return `NOT_IMPLEMENTED`, `unsupported` ones `UNSUPPORTED`; `platformspec.Supports` answers both.

#### Example runtime

The Node runtime replaces the old Python example below; it shows the message shape the backend expects.

```ts
export async function msgSend(ctx: Ctx): Promise<Message> {
  const conversation = await resolveConversation(ctx)
  const id = await sendText(conversation, ctx.args.text)
  return messageOf({ id, conversation_id: conversation.id, text: ctx.args.text }, conversation)
}
```

The Python sketch below still works as a description of the bridge protocol.


```python
import asyncio
import json
import logging
from typing import Dict, Any, Optional
from dataclasses import dataclass

import aiohttp
from websockets import connect

logger = logging.getLogger(__name__)

@dataclass
class AdapterConfig:
    api_key: str
    api_secret: str
    webhook_url: str
    heartbeat_interval: int = 30

class MyPlatformAdapter:
    def __init__(self, config: AdapterConfig):
        self.config = config
        self.ws: Optional[any] = None
        self.session: Optional[aiohttp.ClientSession] = None
        self.running = False
        
    async def start(self):
        """Start the adapter"""
        self.running = True
        self.session = aiohttp.ClientSession()
        
        # Connect to platform WebSocket
        await self._connect_websocket()
        
        # Start heartbeat
        asyncio.create_task(self._heartbeat_loop())
        
        logger.info("Adapter started")
        
    async def stop(self):
        """Stop the adapter"""
        self.running = False
        if self.ws:
            await self.ws.close()
        if self.session:
            await self.session.close()
        logger.info("Adapter stopped")
        
    async def _connect_websocket(self):
        """Connect to platform WebSocket"""
        url = "wss://api.myplatform.com/ws"
        headers = {
            "Authorization": f"Bearer {self.config.api_key}"
        }
        
        self.ws = await connect(url, extra_headers=headers)
        logger.info("WebSocket connected")
        
        # Start message listener
        asyncio.create_task(self._message_loop())
        
    async def _message_loop(self):
        """Listen for incoming messages"""
        async for message in self.ws:
            try:
                data = json.loads(message)
                await self._handle_message(data)
            except Exception as e:
                logger.error(f"Message handling error: {e}")
                
    async def _handle_message(self, data: Dict[str, Any]):
        """Handle incoming message"""
        # Convert to V3 protocol format
        envelope = {
            "protocol_version": "v3",
            "event_type": "message.received",
            "platform": "myplatform",
            "payload": {
                "conversation_id": data.get("conversation_id"),
                "sender_id": data.get("sender_id"),
                "sender_name": data.get("sender_name"),
                "message_type": data.get("message_type"),
                "message_content": data.get("content"),
            }
        }
        
        # Send to backend
        await self._send_to_backend(envelope)
        
    async def _send_to_backend(self, envelope: Dict[str, Any]):
        """Send message to backend"""
        url = f"{self.config.webhook_url}/api/v1/messages"
        headers = {
            "Content-Type": "application/json",
            "X-API-Key": self.config.api_key
        }
        
        async with self.session.post(url, json=envelope, headers=headers) as resp:
            if resp.status != 200:
                logger.error(f"Failed to send message: {resp.status}")
                
    async def _heartbeat_loop(self):
        """Send heartbeat periodically"""
        while self.running:
            try:
                if self.ws:
                    await self.ws.ping()
                await asyncio.sleep(self.config.heartbeat_interval)
            except Exception as e:
                logger.error(f"Heartbeat error: {e}")
                await self._reconnect()
                
    async def _reconnect(self):
        """Reconnect to platform"""
        logger.info("Reconnecting...")
        await asyncio.sleep(5)
        await self._connect_websocket()
        
    async def send_message(self, target_id: str, message: Dict[str, Any]):
        """Send message to platform"""
        if not self.ws:
            raise RuntimeError("WebSocket not connected")
            
        payload = {
            "action": "send_message",
            "target_id": target_id,
            "message_type": message.get("type"),
            "content": message.get("content")
        }
        
        await self.ws.send(json.dumps(payload))
        logger.info(f"Message sent to {target_id}")

# Entry point
async def main():
    config = AdapterConfig(
        api_key="your_api_key",
        api_secret="your_api_secret",
        webhook_url="http://localhost:8080"
    )
    
    adapter = MyPlatformAdapter(config)
    
    try:
        await adapter.start()
        # Keep running
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        await adapter.stop()

if __name__ == "__main__":
    asyncio.run(main())
```

### Step 4: Create requirements.txt

Create `adapters/myplatform/requirements.txt`:

```
aiohttp>=3.9.0
websockets>=12.0
```

### Step 5: Create a Dockerfile (optional)

Create `adapters/myplatform/Dockerfile`:

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "adapter.py"]
```

## Capability declaration

Capabilities are derived, not written. `capabilities()` in `scripts/gen_platformspec.py` walks the matrix and emits a name per supported command, plus the plain words below when the corresponding commands are supported. Do not add capabilities by hand.

Top-level words, kept for consumers that predate the command-level list:

| Capability | Emitted when |
|------|------|
| `receive_message` | the platform supports `msg listen` |
| `send_text`, `send_image`, `create_conversation` | the platform supports `msg send` |
| `upload_media` | the platform supports `media upload` or `item publish` |
| `send_video` | the platform supports `item publish` |
| `get_history` | the platform supports `msg history` |
| `get_product_info` | the platform is Taobao, Xianyu, JD or Douyin and supports `item get` |
| `get_order_info` | the platform supports `order list` |
| `refresh_token` | the platform supports `auth status` |

Command-level names are `<resource>.<action>`; a `planned` command also gets `<resource>.<action>.planned`. Filter on those when a caller only needs commands the runtime can actually execute, and on `platformspec.Supports` when it needs the distinction between planned and unsupported.

## Configuration schema

### Base configuration

```json
{
  "type": "object",
  "properties": {
    "api_key": {
      "type": "string",
      "description": "API Key"
    },
    "api_secret": {
      "type": "string",
      "description": "API Secret"
    },
    "webhook_url": {
      "type": "string",
      "description": "Webhook URL",
      "default": "http://localhost:8080"
    }
  },
  "required": ["api_key", "api_secret"]
}
```

### Cookie configuration (Taobao/Xianyu)

```json
{
  "type": "object",
  "properties": {
    "cookie": {
      "type": "string",
      "description": "Platform login cookie"
    },
    "device_id": {
      "type": "string",
      "description": "Device ID (optional)"
    }
  },
  "required": ["cookie"]
}
```

## Multi-language support

### Configure i18n resources

```go
I18n: map[string]v3.I18nResource{
    "zh-CN": {
        DisplayName:  "MyPlatform adapter",
        Description:  "Supports sending and receiving MyPlatform messages",
        InstallGuide: "1. Log in to the platform\n2. Get an API Key\n3. Configure the adapter",
        ErrorMessages: map[string]string{
            "auth_failed": "Authentication failed, please check the API Key",
            "connection_failed": "Connection failed, please check the network",
        },
    },
    "en-US": {
        DisplayName:  "MyPlatform Adapter",
        Description:  "Supports MyPlatform message sending and receiving",
        InstallGuide: "1. Login to platform\n2. Get API Key\n3. Configure adapter",
        ErrorMessages: map[string]string{
            "auth_failed": "Authentication failed, please check API Key",
            "connection_failed": "Connection failed, please check network",
        },
    },
}
```

## Testing

### Registry tests

```bash
go test ./internal/platformspec/ ./internal/adapter/ ./internal/messagefields/
```

The registry tests check aliases, the stage partition, the derived capabilities, config field names and the adapter manifest. Write your own tests for the runtime against the field contract in [message-fields.md](./message-fields.md).

### Integration tests

```python
import pytest
from adapter import MyPlatformAdapter, AdapterConfig

@pytest.mark.asyncio
async def test_adapter_start():
    config = AdapterConfig(
        api_key="test_key",
        api_secret="test_secret",
        webhook_url="http://localhost:8080"
    )
    
    adapter = MyPlatformAdapter(config)
    await adapter.start()
    
    assert adapter.running
    await adapter.stop()
```

## Best practices

### 1. Error handling

- Implement an automatic reconnect mechanism
- Log detailed errors
- Provide user-friendly error messages

### 2. Performance optimization

- Use a connection pool
- Implement a message queue
- Process messages in batches

### 3. Security considerations

- Store sensitive information encrypted
- Validate all input
- Use HTTPS/WSS

### 4. Observability

- Add a health check endpoint
- Implement metrics collection
- Support distributed tracing

## References

- [Taobao adapter implementation](https://github.com/cv-cat/TaoBaoApis)
- [Xianyu adapter implementation](https://github.com/cv-cat/XianYuApis)
- [Message fields](./message-fields.md)
- [Platform capability matrix](./platform-capabilities.md)
- [catbus migration notes](./catbus-migration.md)
- [V3 protocol documentation](./protocol-v3.md)
- [API reference](./api-reference.md)
