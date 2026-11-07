# API reference

## Overview

E-SP-Line2 provides a RESTful API and WebSocket interfaces for managing platforms, adapters, messages, and routes.

## Base information

- **Base URL**: `http://localhost:8080/api/v1`
- **Authentication**: Bearer Token
- **Response format**: JSON

## Authentication endpoints

### User registration

**POST** `/auth/register`

**Request body**:
```json
{
  "username": "admin",
  "password": "password123",
  "email": "admin@example.com"
}
```

**Response**:
```json
{
  "id": "user-001",
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### User login

**POST** `/auth/login`

**Request body**:
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-001",
    "username": "admin",
    "role": "admin"
  }
}
```

### Get current user

**GET** `/auth/me`

**Headers**:
```
Authorization: Bearer <token>
```

**Response**:
```json
{
  "id": "user-001",
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin"
}
```

## Platform endpoints

### List platforms

**GET** `/platforms`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)

**Response**:
```json
{
  "data": [
    {
      "id": "platform-001",
      "name": "Taobao",
      "code": "taobao",
      "description": "Taobao e-commerce platform",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 2
}
```

### List platform specs

**GET** `/platforms/spec`

Public. Returns the platform registry migrated from catbus: per-platform metadata plus the per-command support matrix. `stage` is `active` (default), `upcoming` or `all`.

**Response**:
```json
{
  "data": [
    {
      "id": "xhs",
      "name": "小红书",
      "item": "笔记",
      "aliases": ["xiaohongshu", "rednote"],
      "stage": "active",
      "upstream": "https://github.com/cv-cat/Spider_XHS",
      "site": "https://www.xiaohongshu.com",
      "login_methods": ["qrcode"],
      "capabilities": ["msg.listen", "msg.send", "send_text"],
      "notes": ["msg send：text only"],
      "support_matrix": {"msg send": "implemented", "item delete": "planned", "live get": "unsupported"}
    }
  ],
  "commands": [{"Resource": "msg", "Action": "send", "Section": "notice 与 msg"}],
  "total": 12
}
```

`support_matrix` covers every command, including `unsupported` ones, so a caller can tell "not this platform" from "not yet". Support levels are `implemented`, `partial`, `planned` and `unsupported`; see [message fields](./message-fields.md) for the exit codes they map to.

### Get one platform spec

**GET** `/platforms/spec/:id`

Public. Accepts an alias. Lists only the commands the platform has, so `planned` and `implemented` are visible while `unsupported` is left out.

**Response**: the platform spec plus `dm`, `config_fields` and a `commands` array of `{resource, action, section, support}`. `support` is `implemented` or `partial` here — planned commands are left out, because this view answers "what can I call now".

### List message fields

**GET** `/message-fields`

Public. Returns the message field contract that [message-fields.md](./message-fields.md) documents, keyed by object kind. `?kind=message` returns a single object.

### Get platform details

**GET** `/platforms/:id`

**Response**:
```json
{
  "id": "platform-001",
  "name": "Taobao",
  "code": "taobao",
  "description": "Taobao e-commerce platform",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Create platform

**POST** `/platforms`

**Request body**:
```json
{
  "name": "Taobao",
  "code": "taobao",
  "description": "Taobao e-commerce platform"
}
```

**Response**:
```json
{
  "id": "platform-001",
  "name": "Taobao",
  "code": "taobao",
  "description": "Taobao e-commerce platform",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Update platform

**PUT** `/platforms/:id`

**Request body**:
```json
{
  "name": "Taobao",
  "description": "Taobao e-commerce platform (updated)",
  "status": "active"
}
```

**Response**:
```json
{
  "id": "platform-001",
  "name": "Taobao",
  "description": "Taobao e-commerce platform (updated)",
  "status": "active",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### Delete platform

**DELETE** `/platforms/:id`

**Response**:
```json
{
  "message": "platform deleted"
}
```

## Adapter endpoints

### List adapters

**GET** `/adapters`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)

**Response**:
```json
{
  "data": [
    {
      "id": "adapter-001",
      "platform_id": "platform-001",
      "name": "Taobao message adapter",
      "version": "1.0.0",
      "runtime_type": "python",
      "protocol_version": "v3",
      "status": "active",
      "capabilities": ["receive_message", "send_text", "send_image"],
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Create adapter

**POST** `/adapters`

**Request body**:
```json
{
  "platform_id": "platform-001",
  "name": "Taobao message adapter",
  "version": "1.0.0",
  "runtime_type": "python",
  "protocol_version": "v3",
  "manifest": "{}"
}
```

**Response**:
```json
{
  "id": "adapter-001",
  "platform_id": "platform-001",
  "name": "Taobao message adapter",
  "version": "1.0.0",
  "runtime_type": "python",
  "protocol_version": "v3",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Get adapter details

**GET** `/adapters/:id`

**Response**:
```json
{
  "id": "adapter-001",
  "platform_id": "platform-001",
  "name": "Taobao message adapter",
  "version": "1.0.0",
  "runtime_type": "python",
  "protocol_version": "v3",
  "status": "active",
  "capabilities": ["receive_message", "send_text", "send_image"],
  "manifest": "{}",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update adapter

**PUT** `/adapters/:id`

**Request body**:
```json
{
  "name": "Taobao message adapter (updated)",
  "version": "1.0.1",
  "status": "active",
  "manifest": "{}"
}
```

**Response**:
```json
{
  "id": "adapter-001",
  "name": "Taobao message adapter (updated)",
  "version": "1.0.1",
  "status": "active",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### Delete adapter

**DELETE** `/adapters/:id`

**Response**:
```json
{
  "message": "adapter deleted"
}
```

### Start adapter

**POST** `/adapters/:id/start`

**Response**:
```json
{
  "message": "adapter started",
  "instance_id": "instance-001"
}
```

### Stop adapter

**POST** `/adapters/:id/stop`

**Response**:
```json
{
  "message": "adapter stopped"
}
```

## Instance endpoints

### List instances

**GET** `/instances`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)
- `platform_id`: Platform ID (optional)
- `status`: Status (optional)

**Response**:
```json
{
  "data": [
    {
      "id": "instance-001",
      "adapter_id": "adapter-001",
      "platform_id": "platform-001",
      "name": "Store A",
      "status": "running",
      "config": "{}",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Create instance

**POST** `/instances`

**Request body**:
```json
{
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "Store A",
  "config": {
    "cookie": "your_cookie_here"
  }
}
```

**Response**:
```json
{
  "id": "instance-001",
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "Store A",
  "status": "stopped",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Get instance details

**GET** `/instances/:id`

**Response**:
```json
{
  "id": "instance-001",
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "Store A",
  "status": "running",
  "config": "{}",
  "session": {
    "connected_at": "2024-01-01T00:00:00Z",
    "last_heartbeat": "2024-01-01T00:05:00Z"
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update instance

**PUT** `/instances/:id`

**Request body**:
```json
{
  "name": "Store A (updated)",
  "config": {
    "cookie": "new_cookie_here"
  }
}
```

**Response**:
```json
{
  "id": "instance-001",
  "name": "Store A (updated)",
  "config": "{}",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### Delete instance

**DELETE** `/instances/:id`

**Response**:
```json
{
  "message": "instance deleted"
}
```

## Message endpoints

### List messages

**GET** `/messages`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)
- `platform_id`: Platform ID (optional)
- `instance_id`: Instance ID (optional)
- `status`: Status (optional)

**Response**:
```json
{
  "data": [
    {
      "id": "msg-001",
      "platform_id": "platform-001",
      "instance_id": "instance-001",
      "conversation_id": "conv-001",
      "sender_id": "user-123",
      "sender_name": "Buyer A",
      "message_type": "text",
      "message_content": "Hello",
      "status": "received",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Get message details

**GET** `/messages/:id`

**Response**:
```json
{
  "id": "msg-001",
  "platform_id": "platform-001",
  "instance_id": "instance-001",
  "conversation_id": "conv-001",
  "sender_id": "user-123",
  "sender_name": "Buyer A",
  "message_type": "text",
  "message_content": "Hello",
  "raw_message": "{}",
  "idempotency_key": "key-001",
  "trace_id": "trace-001",
  "status": "received",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Acknowledge message

**POST** `/messages/:id/ack`

**Response**:
```json
{
  "message": "message acknowledged"
}
```

## Command endpoints

### List commands

**GET** `/commands`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)
- `instance_id`: Instance ID (optional)
- `status`: Status (optional)

**Response**:
```json
{
  "data": [
    {
      "id": "cmd-001",
      "instance_id": "instance-001",
      "command_type": "send_text",
      "payload": "{}",
      "status": "sent",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Create command

**POST** `/commands`

**Request body**:
```json
{
  "instance_id": "instance-001",
  "command_type": "send_text",
  "payload": {
    "target_id": "user-123",
    "content": "Hello"
  }
}
```

**Response**:
```json
{
  "id": "cmd-001",
  "instance_id": "instance-001",
  "command_type": "send_text",
  "status": "created",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Get command details

**GET** `/commands/:id`

**Response**:
```json
{
  "id": "cmd-001",
  "instance_id": "instance-001",
  "command_type": "send_text",
  "payload": "{}",
  "status": "sent",
  "retry_count": 0,
  "max_retries": 3,
  "trace_id": "trace-001",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "sent_at": "2024-01-01T00:00:01Z"
}
```

## Route endpoints

### List routing rules

**GET** `/routes`

**Query parameters**:
- `limit`: Number of results to return (default 20)
- `offset`: Offset (default 0)
- `platform_id`: Platform ID (optional)
- `enabled`: Whether enabled (optional)

**Response**:
```json
{
  "data": [
    {
      "id": "route-001",
      "name": "Taobao message route",
      "platform_id": "platform-001",
      "priority": 10,
      "conditions": "{}",
      "target_type": "app",
      "target_id": "app-001",
      "enabled": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Create routing rule

**POST** `/routes`

**Request body**:
```json
{
  "name": "Taobao message route",
  "platform_id": "platform-001",
  "priority": 10,
  "conditions": {
    "event_type": "message.received"
  },
  "target_type": "app",
  "target_id": "app-001",
  "enabled": true
}
```

**Response**:
```json
{
  "id": "route-001",
  "name": "Taobao message route",
  "platform_id": "platform-001",
  "priority": 10,
  "conditions": "{}",
  "target_type": "app",
  "target_id": "app-001",
  "enabled": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Get routing rule details

**GET** `/routes/:id`

**Response**:
```json
{
  "id": "route-001",
  "name": "Taobao message route",
  "platform_id": "platform-001",
  "priority": 10,
  "conditions": "{}",
  "target_type": "app",
  "target_id": "app-001",
  "enabled": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update routing rule

**PUT** `/routes/:id`

**Request body**:
```json
{
  "name": "Taobao message route (updated)",
  "priority": 20,
  "enabled": false
}
```

**Response**:
```json
{
  "id": "route-001",
  "name": "Taobao message route (updated)",
  "priority": 20,
  "enabled": false,
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### Delete routing rule

**DELETE** `/routes/:id`

**Response**:
```json
{
  "message": "route deleted"
}
```

## WebSocket interface

### Adapter WebSocket

**URL**: `ws://localhost:8080/ws/adapter?instance_id=<instance_id>`

**Message format**:
```json
{
  "type": "message.received",
  "event_id": "evt-001",
  "timestamp": 1704067200000,
  "platform": "taobao",
  "adapter_id": "adapter-001",
  "payload": {
    "conversation_id": "conv-001",
    "sender_id": "user-123",
    "sender_name": "Buyer A",
    "message_type": "text",
    "message_content": "Hello"
  }
}
```

### App WebSocket

**URL**: `ws://localhost:8080/ws/app?app_id=<app_id>`

**Message format**:
```json
{
  "type": "message.received",
  "event_id": "evt-001",
  "timestamp": 1704067200000,
  "platform": "taobao",
  "adapter_id": "adapter-001",
  "payload": {
    "conversation_id": "conv-001",
    "sender_id": "user-123",
    "sender_name": "Buyer A",
    "message_type": "text",
    "message_content": "Hello"
  }
}
```

## Error codes

| Error code | Description |
|--------|------|
| 400 | Invalid request parameters |
| 401 | Unauthenticated |
| 403 | Forbidden |
| 404 | Resource not found |
| 409 | Resource conflict |
| 500 | Internal server error |

## Rate limiting

- Default limit: 100 requests/minute
- Authentication endpoints: 10 requests/minute
- WebSocket: unlimited

## Version history

- v1.0.0 (2024-01-01): Initial version
