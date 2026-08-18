# 桥（Bridge）协议规范

> 桥（Bridge）是连接电商平台（淘宝/闲鱼）与 E-SP-Line2 核心的中间层进程。
> 本文档定义桥进程与后端核心之间的 **WebSocket 通信协议**。
>
> 注意：这是**独立于 ESPL v3 接入器协议**的另一套协议。ESPL v3 用于外部系统
> 接入核心；Bridge 协议用于桥进程与核心通信。

## 1. 概述

- **传输**：WebSocket
- **连接端点**：`ws://<esp-host>:<esp-port>/ws/adapter?instance_id=<INSTANCE_ID>`
- **方向**：桥进程主动连接后端核心（桥是 Client，核心是 Server）
- **消息格式**：JSON 文本帧

## 2. 连接流程

```
桥进程                         后端核心
  │  1. 拉取实例配置 (HTTP GET /api/v1/instances/:id)   │
  │ ───────────────────────────────────────────────────► │
  │  2. 建立 WebSocket 连接                              │
  │ ───────────────────────────────────────────────────► │
  │  3. 收到 connected 握手                              │
  │ ◄─────────────────────────────────────────────────── │
  │  4. 上报入站消息 / 接收出站指令（双向）                │
  │ ◄───────────────────────────────────────────────────►│
```

### 2.1 拉取实例配置

桥启动时通过 HTTP API 拉取实例配置（Cookie / DeviceID 等）：

```
GET /api/v1/instances/<instance_id>
Authorization: Bearer <token>
```

> 这是**管理 API**（REST），不是消息协议。桥仅用它获取启动配置。

### 2.2 建立 WebSocket 连接

```
ws://<esp-host>:<esp-port>/ws/adapter?instance_id=<INSTANCE_ID>
```

### 2.3 握手

连接建立后，后端发送 `connected` 消息：

```json
{
  "type": "connected",
  "timestamp": 1723891200000,
  "message": "Adapter WebSocket connected"
}
```

## 3. 消息类型

### 3.1 入站消息（桥 → 核心）

桥收到电商平台消息后，上报给核心。支持两种格式：

**格式 A：扁平 payload**

```json
{
  "platform_id": "xianyu",
  "conversation_id": "123456",
  "sender_id": "user789",
  "sender_name": "张三",
  "message_type": "text",
  "message_content": "你好",
  "idempotency_key": "xianyu-inst-msg-001",
  "raw": { "完整平台原始消息" },
  "message_chain": [
    { "type": "text", "content": { "text": "你好" } }
  ]
}
```

**格式 B：完整 ESPL v3 信封**（`protocol_version: "v3"`）

```json
{
  "protocol_version": "v3",
  "event_id": "evt-uuid",
  "trace_id": "trace-xxx",
  "timestamp": 1723891200000,
  "platform": "xianyu",
  "adapter_id": "inst-uuid",
  "event_type": "message.received",
  "payload": {
    "platform_id": "xianyu",
    "conversation_id": "123456",
    "sender_id": "user789",
    "sender_name": "张三",
    "message_type": "text",
    "message_content": "你好",
    "message_chain": [
      { "type": "text", "content": { "text": "你好" } }
    ],
    "raw": { "完整平台原始消息" }
  }
}
```

**核心响应 ack：**

```json
{
  "type": "ack",
  "timestamp": 1723891200000,
  "event_id": "evt-uuid"
}
```

### 3.2 出站指令（核心 → 桥）

核心向桥下发发送指令（如外部系统回复消息时）：

```json
{
  "type": "send_text",
  "command_type": "send_text",
  "payload": {
    "cid": "conversation-id",
    "toid": "target-user-id",
    "text": "您好，商品有货的，欢迎下单！"
  }
}
```

**支持的指令类型：**

| 指令 | 说明 | payload 字段 |
|------|------|--------------|
| `send_text` | 发送文本消息 | `cid`（会话）、`toid`（目标）、`text`（内容） |
| `send_image` | 发送图片消息 | `cid`、`toid`、`image_url` 等 |

> 桥进程收到指令后调用电商平台 API 发送，发送结果通过日志记录。

## 4. 心跳与重连

- **心跳**：桥进程与核心之间通过 WebSocket 协议层的心跳（Ping/Pong）维持连接。
- **重连**：桥进程断线后按 `reconnect_delay`（默认 5 秒）自动重连，指数退避。
- **多开**：每个实例一个独立桥进程，独立 Cookie 配置，独立 WebSocket 连接。

## 5. 与 ESPL v3 的区别

| 维度 | Bridge 协议 | ESPL v3 |
|------|-------------|---------|
| 通信双方 | 桥进程 ↔ 核心 | 外部系统 ↔ 核心 |
| 连接方向 | 桥主动连核心 | 外部系统主动连核心（或核心主动连外部） |
| 端点 | `/ws/adapter?instance_id=xxx` | `/ws`、`/ws/adapter-gateway` 等 |
| 消息格式 | 扁平 payload / v3 信封 | v3 信封（消息链） |
| 用途 | 电商平台消息收发 | 外部框架接入 |

## 6. 参考实现

- 桥进程：[`adapters/taobao/esp_bridge.py`](../../adapters/taobao/esp_bridge.py)、[`adapters/xianyu/esp_bridge.py`](../../adapters/xianyu/esp_bridge.py)
- 后端入口：[`internal/handler/websocket.go`](../../internal/handler/websocket.go)（`AdapterWebSocket`）
