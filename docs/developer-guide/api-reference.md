# API 参考文档

## 概述

E-SP-Line2 提供 RESTful API 和 WebSocket 接口，用于管理平台、适配器、消息和路由。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: Bearer Token
- **响应格式**: JSON

## 认证接口

### 用户注册

**POST** `/auth/register`

**请求体**:
```json
{
  "username": "admin",
  "password": "password123",
  "email": "admin@example.com"
}
```

**响应**:
```json
{
  "id": "user-001",
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 用户登录

**POST** `/auth/login`

**请求体**:
```json
{
  "username": "admin",
  "password": "password123"
}
```

**响应**:
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

### 获取当前用户

**GET** `/auth/me`

**Headers**:
```
Authorization: Bearer <token>
```

**响应**:
```json
{
  "id": "user-001",
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin"
}
```

## 平台接口

### 获取平台列表

**GET** `/platforms`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）

**响应**:
```json
{
  "data": [
    {
      "id": "platform-001",
      "name": "淘宝",
      "code": "taobao",
      "description": "淘宝电商平台",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 2
}
```

### 获取平台详情

**GET** `/platforms/:id`

**响应**:
```json
{
  "id": "platform-001",
  "name": "淘宝",
  "code": "taobao",
  "description": "淘宝电商平台",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 创建平台

**POST** `/platforms`

**请求体**:
```json
{
  "name": "淘宝",
  "code": "taobao",
  "description": "淘宝电商平台"
}
```

**响应**:
```json
{
  "id": "platform-001",
  "name": "淘宝",
  "code": "taobao",
  "description": "淘宝电商平台",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 更新平台

**PUT** `/platforms/:id`

**请求体**:
```json
{
  "name": "淘宝",
  "description": "淘宝电商平台（已更新）",
  "status": "active"
}
```

**响应**:
```json
{
  "id": "platform-001",
  "name": "淘宝",
  "description": "淘宝电商平台（已更新）",
  "status": "active",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### 删除平台

**DELETE** `/platforms/:id`

**响应**:
```json
{
  "message": "platform deleted"
}
```

## 适配器接口

### 获取适配器列表

**GET** `/adapters`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）

**响应**:
```json
{
  "data": [
    {
      "id": "adapter-001",
      "platform_id": "platform-001",
      "name": "淘宝消息适配器",
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

### 创建适配器

**POST** `/adapters`

**请求体**:
```json
{
  "platform_id": "platform-001",
  "name": "淘宝消息适配器",
  "version": "1.0.0",
  "runtime_type": "python",
  "protocol_version": "v3",
  "manifest": "{}"
}
```

**响应**:
```json
{
  "id": "adapter-001",
  "platform_id": "platform-001",
  "name": "淘宝消息适配器",
  "version": "1.0.0",
  "runtime_type": "python",
  "protocol_version": "v3",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 获取适配器详情

**GET** `/adapters/:id`

**响应**:
```json
{
  "id": "adapter-001",
  "platform_id": "platform-001",
  "name": "淘宝消息适配器",
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

### 更新适配器

**PUT** `/adapters/:id`

**请求体**:
```json
{
  "name": "淘宝消息适配器（已更新）",
  "version": "1.0.1",
  "status": "active",
  "manifest": "{}"
}
```

**响应**:
```json
{
  "id": "adapter-001",
  "name": "淘宝消息适配器（已更新）",
  "version": "1.0.1",
  "status": "active",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### 删除适配器

**DELETE** `/adapters/:id`

**响应**:
```json
{
  "message": "adapter deleted"
}
```

### 启动适配器

**POST** `/adapters/:id/start`

**响应**:
```json
{
  "message": "adapter started",
  "instance_id": "instance-001"
}
```

### 停止适配器

**POST** `/adapters/:id/stop`

**响应**:
```json
{
  "message": "adapter stopped"
}
```

## 实例接口

### 获取实例列表

**GET** `/instances`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）
- `platform_id`: 平台 ID（可选）
- `status`: 状态（可选）

**响应**:
```json
{
  "data": [
    {
      "id": "instance-001",
      "adapter_id": "adapter-001",
      "platform_id": "platform-001",
      "name": "店铺A",
      "status": "running",
      "config": "{}",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 创建实例

**POST** `/instances`

**请求体**:
```json
{
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "店铺A",
  "config": {
    "cookie": "your_cookie_here"
  }
}
```

**响应**:
```json
{
  "id": "instance-001",
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "店铺A",
  "status": "stopped",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 获取实例详情

**GET** `/instances/:id`

**响应**:
```json
{
  "id": "instance-001",
  "adapter_id": "adapter-001",
  "platform_id": "platform-001",
  "name": "店铺A",
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

### 更新实例

**PUT** `/instances/:id`

**请求体**:
```json
{
  "name": "店铺A（已更新）",
  "config": {
    "cookie": "new_cookie_here"
  }
}
```

**响应**:
```json
{
  "id": "instance-001",
  "name": "店铺A（已更新）",
  "config": "{}",
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### 删除实例

**DELETE** `/instances/:id`

**响应**:
```json
{
  "message": "instance deleted"
}
```

## 消息接口

### 获取消息列表

**GET** `/messages`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）
- `platform_id`: 平台 ID（可选）
- `instance_id`: 实例 ID（可选）
- `status`: 状态（可选）

**响应**:
```json
{
  "data": [
    {
      "id": "msg-001",
      "platform_id": "platform-001",
      "instance_id": "instance-001",
      "conversation_id": "conv-001",
      "sender_id": "user-123",
      "sender_name": "买家A",
      "message_type": "text",
      "message_content": "你好",
      "status": "received",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 获取消息详情

**GET** `/messages/:id`

**响应**:
```json
{
  "id": "msg-001",
  "platform_id": "platform-001",
  "instance_id": "instance-001",
  "conversation_id": "conv-001",
  "sender_id": "user-123",
  "sender_name": "买家A",
  "message_type": "text",
  "message_content": "你好",
  "raw_message": "{}",
  "idempotency_key": "key-001",
  "trace_id": "trace-001",
  "status": "received",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 确认消息

**POST** `/messages/:id/ack`

**响应**:
```json
{
  "message": "message acknowledged"
}
```

## 命令接口

### 获取命令列表

**GET** `/commands`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）
- `instance_id`: 实例 ID（可选）
- `status`: 状态（可选）

**响应**:
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

### 创建命令

**POST** `/commands`

**请求体**:
```json
{
  "instance_id": "instance-001",
  "command_type": "send_text",
  "payload": {
    "target_id": "user-123",
    "content": "你好"
  }
}
```

**响应**:
```json
{
  "id": "cmd-001",
  "instance_id": "instance-001",
  "command_type": "send_text",
  "status": "created",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 获取命令详情

**GET** `/commands/:id`

**响应**:
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

## 路由接口

### 获取路由规则列表

**GET** `/routes`

**查询参数**:
- `limit`: 返回数量（默认 20）
- `offset`: 偏移量（默认 0）
- `platform_id`: 平台 ID（可选）
- `enabled`: 是否启用（可选）

**响应**:
```json
{
  "data": [
    {
      "id": "route-001",
      "name": "淘宝消息路由",
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

### 创建路由规则

**POST** `/routes`

**请求体**:
```json
{
  "name": "淘宝消息路由",
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

**响应**:
```json
{
  "id": "route-001",
  "name": "淘宝消息路由",
  "platform_id": "platform-001",
  "priority": 10,
  "conditions": "{}",
  "target_type": "app",
  "target_id": "app-001",
  "enabled": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 获取路由规则详情

**GET** `/routes/:id`

**响应**:
```json
{
  "id": "route-001",
  "name": "淘宝消息路由",
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

### 更新路由规则

**PUT** `/routes/:id`

**请求体**:
```json
{
  "name": "淘宝消息路由（已更新）",
  "priority": 20,
  "enabled": false
}
```

**响应**:
```json
{
  "id": "route-001",
  "name": "淘宝消息路由（已更新）",
  "priority": 20,
  "enabled": false,
  "updated_at": "2024-01-01T01:00:00Z"
}
```

### 删除路由规则

**DELETE** `/routes/:id`

**响应**:
```json
{
  "message": "route deleted"
}
```

## WebSocket 接口

### 适配器 WebSocket

**URL**: `ws://localhost:8080/ws/adapter?instance_id=<instance_id>`

**消息格式**:
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
    "sender_name": "买家A",
    "message_type": "text",
    "message_content": "你好"
  }
}
```

### 应用 WebSocket

**URL**: `ws://localhost:8080/ws/app?app_id=<app_id>`

**消息格式**:
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
    "sender_name": "买家A",
    "message_type": "text",
    "message_content": "你好"
  }
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 500 | 服务器内部错误 |

## 速率限制

- 默认限制：100 请求/分钟
- 认证接口：10 请求/分钟
- WebSocket：无限制

## 版本历史

- v1.0.0 (2024-01-01): 初始版本
