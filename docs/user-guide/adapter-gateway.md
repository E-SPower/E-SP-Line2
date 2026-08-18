# WebSocket 接入器（Adapter Gateway）

> 接入器（Adapter）是 E-SP-Line2 的可配置 WebSocket 端点，支持**双向**接入：
> - **服务端模式（Server）**：本系统监听 WS 端口，外部系统（机器人框架/客服系统等）携带 key 连接
> - **客户端模式（Client）**：本系统主动连接外部 WS 地址，携带 key 鉴权
>
> 与「桥（Bridge）」的区别：桥连接电商平台与核心；接入器连接核心与外部系统。
>
> **协议**：接入器使用 **ESPL v3 协议**（WebSocket）。桥进程与核心通信使用
> **Bridge 协议**（另一套 WebSocket 协议）。老版接入器 HTTP v1/v2 协议已弃用。
> 详见 [协议体系总览](../developer-guide/protocol-overview.md)。

## 1. 概念

| 概念 | 英文 | 角色 | 方向 | 现有代码 |
|------|------|------|------|----------|
| **桥** | Bridge | 连接电商平台（淘宝/闲鱼）与核心 | 电商平台 ↔ 核心 | [`adapters/taobao/`](../../adapters/taobao/main.py)、[`adapters/xianyu/`](../../adapters/xianyu/main.py) |
| **接入器** | Adapter | 可配置的 WebSocket 端点（双模式） | 外部系统 ↔ 核心 | [`internal/adaptergateway/`](../../internal/adaptergateway/gateway.go) |

一句话总结：**桥把电商平台接入核心，接入器让外部系统与核心双向通信。**

## 2. 双模式架构

### 2.1 服务端模式（Server）

**核心监听**，外部系统主动连接：

```
外部系统 (Client) --携带key--> E-SP-Line2 (Server)
```

**连接端点（默认监听路径为 `/ws`）：**
```
ws://<esp-host>:<esp-port>/ws?key=<KEY>
```

或使用自定义监听路径（在接入器配置中指定 `listen_path`）：
```
ws://<esp-host>:<esp-port>/custom/path?key=<KEY>
```

兼容旧路径 `/ws/adapter-gateway?key=<KEY>` 与 `/ws/adapter-gateway/<adapter-id>?key=<KEY>`。

### 2.2 客户端模式（Client）

**核心主动连接**外部系统，携带 key 鉴权：

```
E-SP-Line2 (Client) --携带key--> 外部系统 (Server)
```

**目标地址**在接入器配置中指定 `ws_url`，例如：
```
ws://external-bot:9000/ws?key=<KEY>
```

## 3. 接入器管理

### 3.1 Web UI

访问 **WebSocket 接入器** 页面（`/adapter-gateway`）进行可视化管理：

- **卡片式展示**：每个接入器一张卡片，显示模式、端点、平台、在线状态
- **创建/编辑**：支持双模式切换、自定义 key、配置 WS 地址
- **连接记录**：查看当前和历史连接

### 3.2 API

#### 创建接入器

```http
POST /api/v1/adapter-gateways
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "name": "客服接入器-1",
  "mode": "server",                    // server / client
  "listen_path": "/ws/custom",         // server 模式：监听路径（可选）
  "ws_url": "ws://bot:9000/ws",        // client 模式：目标地址（必填）
  "key": "my-secret-key",              // 访问密钥（可随时修改）
  "platform": "xianyu",                // taobao / xianyu / ""（全部）
  "scope": "read+write",               // read / write / read+write
  "enabled": true
}
```

**响应：**
```json
{
  "id": "uuid",
  "name": "客服接入器-1",
  "mode": "server",
  "listen_path": "/ws/custom",
  "key": "my-secret-key",
  "platform": "xianyu",
  "scope": "read+write",
  "status": "active",
  "enabled": true,
  "created_at": "2026-08-17T10:00:00Z"
}
```

#### 更新接入器

```http
PUT /api/v1/adapter-gateways/<id>
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "key": "new-secret-key",   // 支持修改 key
  "enabled": false           // 禁用接入器
}
```

#### 列出接入器

```http
GET /api/v1/adapter-gateways?limit=20&offset=0
Authorization: Bearer <JWT>
```

#### 查看连接记录

```http
GET /api/v1/adapter-gateways/<id>/connections?limit=20&offset=0
Authorization: Bearer <JWT>
```

## 4. ESPL v3 协议

### 4.1 握手

**连接建立后**，服务端发送：

```json
{
  "type": "connected",
  "id": "conn-uuid",
  "timestamp": 1723891200000,
  "adapter_id": "adapter-uuid",
  "gateway_version": "v3",
  "session_id": "conn-uuid",
  "adapter_name": "客服接入器-1",
  "platform": "xianyu"
}
```

### 4.2 心跳

**客户端发送 ping：**
```json
{
  "type": "ping",
  "timestamp": 1723891200000
}
```

**服务端响应 pong：**
```json
{
  "type": "pong",
  "timestamp": 1723891200000
}
```

### 4.3 入站消息（核心 → 外部系统）

当电商平台有新消息时，核心通过接入器广播给匹配平台的外部系统：

```json
{
  "protocol_version": "v3",
  "event_id": "evt-uuid",
  "trace_id": "trace-xxx",
  "timestamp": 1723891200000,
  "platform": "xianyu",
  "event_type": "message.received",
  "payload": {
    "platform_id": "xianyu",
    "conversation_id": "123456",
    "sender_id": "user789",
    "sender_name": "张三",
    "message_type": "text",
    "message_content": "你好",
    "message_chain": [
      { "type": "text", "text": "你好" }
    ],
    "raw": { /* 完整的平台原始消息 */ }
  }
}
```

**关键字段：**
- `message_chain`：消息链（V3 标准格式）
- `raw`：完整的原始平台消息（不丢失任何信息）

### 4.4 出站消息（外部系统 → 核心）

外部系统回复消息：

```json
{
  "type": "message",
  "id": "msg-uuid",
  "timestamp": 1723891200000,
  "payload": {
    "instance_id": "inst-uuid",
    "command_type": "send_text",
    "conversation_id": "123456",
    "target_id": "user789",
    "message_chain": [
      { "type": "text", "text": "你好，有什么可以帮您？" }
    ]
  }
}
```

**核心响应 ack：**
```json
{
  "type": "ack",
  "id": "msg-uuid",
  "timestamp": 1723891200000
}
```

### 4.5 错误

```json
{
  "type": "error",
  "code": "40103",
  "message": "adapter has no write permission",
  "timestamp": 1723891200000
}
```

**错误码：**
- `40101`：未授权（key 无效或未提供）
- `40103`：权限不足（scope 限制）
- `40001`：请求格式错误

## 5. 权限控制

### 5.1 平台过滤

`platform` 字段限制接入器接收的消息：
- `"taobao"`：只接收淘宝消息
- `"xianyu"`：只接收闲鱼消息
- `""`：接收所有平台消息

### 5.2 Scope 权限

- `"read"`：只读（只接收入站消息，不能回复）
- `"write"`：只写（只能发送出站消息，不接收入站）
- `"read+write"`：读写（完整权限）

## 6. 客户端示例

### 6.1 Python 示例（连接 Server 模式）

```python
import asyncio
import websockets
import json

async def connect_adapter():
    uri = "ws://localhost:8080/ws/adapter-gateway/abc123?key=my-secret-key"
    
    async with websockets.connect(uri) as ws:
        # 等待 connected 事件
        msg = await ws.recv()
        print("Connected:", msg)
        
        # 监听入站消息
        async def receive():
            async for message in ws:
                data = json.loads(message)
                if data.get("event_type") == "message.received":
                    payload = data["payload"]
                    print(f"New message: {payload['message_content']}")
                    
                    # 回复
                    reply = {
                        "type": "message",
                        "id": "reply-123",
                        "timestamp": int(time.time() * 1000),
                        "payload": {
                            "instance_id": "inst-uuid",
                            "command_type": "send_text",
                            "conversation_id": payload["conversation_id"],
                            "target_id": payload["sender_id"],
                            "message_chain": [
                                {"type": "text", "text": "收到！"}
                            ]
                        }
                    }
                    await ws.send(json.dumps(reply))
        
        # 心跳
        async def heartbeat():
            while True:
                await asyncio.sleep(30)
                await ws.send(json.dumps({
                    "type": "ping",
                    "timestamp": int(time.time() * 1000)
                }))
        
        await asyncio.gather(receive(), heartbeat())

asyncio.run(connect_adapter())
```

### 6.2 Client 模式（核心主动连接外部系统）

**外部系统**需要实现 WebSocket 服务端，接收核心的连接请求。握手时核心会携带 `key` 参数，外部系统需验证。

**外部系统伪代码：**
```python
async def handle_esp_connection(websocket, path):
    # 验证 key
    key = parse_query_param(path, "key")
    if key != "expected-key":
        await websocket.close(1008, "Unauthorized")
        return
    
    # 发送 connected
    await websocket.send(json.dumps({
        "type": "connected",
        "timestamp": int(time.time() * 1000)
    }))
    
    # 处理入站消息
    async for message in websocket:
        data = json.loads(message)
        # ... 处理消息
```

## 7. 故障排查

### 7.1 连接失败

- 检查 `key` 是否正确
- 检查接入器状态是否为 `active` 且 `enabled`
- 检查网络连通性
- 查看后端日志：`grep "Adapter gateway auth failed" esp.log`

### 7.2 收不到消息

- 检查 `platform` 过滤
- 检查 `scope` 权限（需要包含 `read`）
- 确认桥已连接且正常工作

### 7.3 无法回复

- 检查 `scope` 权限（需要包含 `write`）
- 确认 `instance_id` 正确
- 检查 `command_type` 是否支持

## 8. 最佳实践

1. **使用强随机 key**：至少 32 字符，包含字母、数字、符号
2. **定期轮换 key**：通过 API 或 Web UI 更新 key
3. **最小权限原则**：只读接入器使用 `scope: read`
4. **平台隔离**：为不同平台创建独立接入器
5. **监控连接状态**：定期检查连接记录，及时发现异常
6. **保存 raw 消息**：完整的原始消息有助于故障排查和数据分析

## 9. 迁移指南

如果你正在从旧的 token 机制迁移到新的 adapter 实体：

1. **数据库迁移**：运行 `migrations/002_adapter_gateway.sql`
2. **更新客户端**：修改连接 URL 为 `/ws/adapter-gateway/<id>?key=<key>`
3. **UI 升级**：旧的 token 管理页面已被 adapter-gateway 页面替代
4. **API 更新**：`/adapter-tokens` → `/adapter-gateways`

## 10. 参考

- [V3 协议规范](../developer-guide/protocol-v3.md)
- [桥（Bridge）vs 接入器（Adapter）](../architecture-bridge-vs-adapter.md)
- [API 参考](../developer-guide/api-reference.md)
