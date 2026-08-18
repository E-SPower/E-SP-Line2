# 架构重构设计：桥 (Bridge) 与 接入器 (Adapter) 概念分离

> 状态：**设计提案（Design Proposal）** · 作者：E-SP-Line2 架构组
>
> 本文档重新定义 E-SP-Line2 中「桥」与「接入器」两个核心概念，明确二者的职责边界，
> 并定义全新的「接入器对外 WebSocket 服务端口 + 令牌」协议，用于接入外部聊天机器人 /
> 客服系统等框架。**本文档仅作设计，不涉及代码改动。**

---

## 1. 背景与动机

### 1.1 现状问题：概念被混用

在现有代码库中，「接入器 (Adapter)」一词同时承载了两个完全不同的职责：

1. **连接电商平台**（淘宝 / 闲鱼）：`adapters/taobao/`、`adapters/xianyu/` 下的 Python
   进程，通过逆向 SDK 连接淘宝/闲鱼 WebSocket，将平台消息转换为 ESPL v3 信封上报后端，
   并接收后端出站指令调用平台 API 发送。这在本质上是**桥 (Bridge)**——桥接两个异构系统
   （电商平台 ↔ E-SP-Line2 核心）。

2. **对外暴露服务端口 + 令牌**，让外部聊天机器人 / 客服系统等框架作为**客户端**接入
   E-SP-Line2，把电商消息作为「消息源」喂给外部框架，并把框架回复发回电商平台。
   这才是「接入器 (Adapter)」的本义——把外部系统「接入」E-SP-Line2。

当前后端只有两个 WebSocket 入口（[`internal/handler/websocket.go`](../internal/handler/websocket.go:1)）：

- `/ws/adapter?instance_id=xxx`：**桥回连**入口（Python 桥进程连回后端）
- `/ws/app?app_id=xxx`：下游应用入口

**没有任何一个入口是「服务端」形态**——即后端监听端口、发放令牌、等待外部框架
作为客户端主动连入。这正是本次设计要补齐的能力。

### 1.2 概念重定义

| 概念 | 英文 | 角色 | 方向 | 现有代码位置 |
|------|------|------|------|--------------|
| **桥** | Bridge | 连接**电商平台**（淘宝/闲鱼）与 E-SP-Line2 核心 | 电商平台 ↔ 核心 | [`adapters/taobao/`](../adapters/taobao/esp_bridge.py:1)、[`adapters/xianyu/`](../adapters/xianyu/esp_bridge.py:1) |
| **接入器** | Adapter | 让**外部机器人框架**接入 E-SP-Line2，消费电商消息并回复 | 外部框架 ↔ 核心 | **不存在，需新建** |

> 一句话总结：**桥把外部平台「桥」进核心，接入器把核心「接入」外部框架。**
> 桥是 E-SP-Line2 主动发起的连接（核心是 Server），接入器是外部框架主动发起的连接
> （核心是 Server，外部框架是 Client）。

### 1.3 目标

- **G1**：明确「桥」与「接入器」的职责边界，建立统一的概念模型。
- **G2**：设计接入器的**对外 WebSocket 服务端口 + 令牌**机制，令牌可创建、吊销、轮换。
- **G3**：定义接入器与外部框架之间的**消息协议**（消息链、事件、ACK、心跳）。
- **G4**：定义接入器如何与现有桥/消息路由/出站指令系统衔接，形成完整闭环。
- **G5**：给出迁移路径（现有 `adapter` 命名 → `bridge` 命名），不影响存量功能。

### 1.4 非目标

- 不实现外部框架的**全部**平台协议（如完整 OneBot / AIOBot 规范），只定义
  E-SP-Line2 自己的接入器协议，并给出与外部框架对接的适配层设计。
- 不替代现有 `/ws/adapter`（桥回连）入口——它继续作为「桥」的通信通道存在。
- 不实现 Web 管理界面（WebUI 令牌管理页）的具体 UI，只定义数据模型与 API。

---

## 2. 整体架构

### 2.1 目标架构总览

```
┌───────────────────────────  E-SP-Line2 核心 (Go)  ───────────────────────────┐
│                                                                              │
│  ┌─────────────┐   ESPL v3 WS    ┌──────────────┐   路由/分发   ┌──────────┐ │
│  │ 电商平台     │ ◄─────────────► │   桥 (Bridge)  │ ──────────► │ 消息核心  │ │
│  │ 淘宝/闲鱼 WS │    (桥回连)     │  Python 进程   │  inbound    │  Hub/路由 │ │
│  └─────────────┘                 └──────────────┘              └────┬─────┘ │
│                                                                     │       │
│  ┌──────────────────────────────────────────────────────────────┐  │       │
│  │  接入器网关 (Adapter Gateway) — 对外 WS 服务端口 + 令牌        │ ◄┘       │
│  │  • 监听 /ws/adapter-gateway  (服务端)                        │          │
│  │  • 令牌认证 (Token Auth)                                     │          │
│  │  • 消息链转换 (ESPL v3 ↔ 接入器协议)                          │          │
│  │  • 会话管理 / 心跳 / 重连                                     │          │
│  └───────────────────────────────┬──────────────────────────────┘          │
│                                  │ 接入器协议 (WS Client)                    │
└──────────────────────────────────┼──────────────────────────────────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              ▼                    ▼                    ▼
        ┌────────────┐      ┌────────────┐      ┌────────────┐
        │  机器人框架  │      │  客服系统   │      │  其他框架   │
        │  (WS Client)│      │  (WS Client)│     │  (WS Client)│
        └────────────┘      └────────────┘      └────────────┘
```

**核心闭环（以闲鱼买家咨询为例）：**

```
买家在闲鱼发消息
  → 桥 (闲鱼) 收到，转 ESPL v3 信封上报后端
  → 消息核心入库、路由
  → 接入器网关将消息转换为接入器协议，推送给已连入的外部框架
  → 外部框架调用大模型生成回复
  → 回复经接入器协议回传接入器网关
  → 核心将回复转换为出站指令，路由给对应桥实例
  → 桥调用闲鱼 API 把回复发给买家
```

### 2.2 角色模型（Role Model）

| 角色 | 组件 | 主动方 | 连接形态 | 协议 |
|------|------|--------|----------|------|
| 桥客户端 | Python 桥进程（淘宝/闲鱼） | 桥（主动连后端） | WS Client → `/ws/adapter` | ESPL v3 |
| 桥服务端 | 后端桥接入点 | 后端（被动接收） | WS Server | ESPL v3 |
| 接入器客户端 | 外部框架 | 框架（主动连后端） | WS Client → 接入器网关 | 接入器协议（本文 §4） |
| 接入器服务端 | 接入器网关 | 后端（被动接收） | WS Server + 令牌 | 接入器协议（本文 §4） |

---

## 3. 桥 (Bridge) 概念澄清与重命名

### 3.1 桥的职责

桥（现有 `adapters/taobao/`、`adapters/xianyu/`）的职责保持不变：

1. 连接电商平台 WebSocket，订阅消息。
2. 启动时从后端拉取实例配置（Cookie / DeviceID）。
3. 将平台消息转换为 ESPL v3 信封，通过 `/ws/adapter?instance_id=xxx` 上报后端。
4. 接收后端下发的出站指令（`send_text` / `send_image` 等），调用平台 API 发送。
5. 支持多开：每个实例独立进程/线程，独立 Cookie。

### 3.2 重命名建议

为消除概念歧义，建议将现有代码中「桥」相关的命名从 `adapter` 改为 `bridge`：

| 现状 | 建议 | 说明 |
|------|------|------|
| `adapters/` 目录 | `bridges/` 目录 | 桥代码目录 |
| `adapter.yaml` | `bridge.yaml` | 桥的元数据文件 |
| `AdapterPackage` 模型 | `BridgePackage` 模型 | 见 §6 数据模型 |
| `AdapterInstance` 模型 | `BridgeInstance` 模型 | 见 §6 数据模型 |
| `AdapterService` | `BridgeService` | 见 §6 服务层 |
| `/api/v1/adapters` | `/api/v1/bridges` | 桥管理 API |
| `/ws/adapter` | `/ws/bridge` | 桥回连入口 |
| `platform_code` | `platform_code`（不变） | 桥关联的电商平台代码 |

> **兼容策略**：重命名期间保留旧路由/旧模型为别名（如 `/api/v1/adapters` 重定向到
> `/api/v1/bridges`），数据库表名通过 GORM `TableName()` 兼容，避免破坏存量数据与
> 已部署的桥进程。迁移完成后删除别名。

### 3.3 桥与接入器的关系

```
桥 (Bridge)                    接入器 (Adapter)
─────────                      ──────────────
连接电商平台                   连接机器人框架
消息源：平台                   消息源：核心（来自桥）
消息汇：核心                   消息汇：机器人框架
主动连后端 (WS Client)         被动接框架 (WS Server)
每实例 = 一个账号              每令牌 = 一个框架接入
```

二者通过核心的消息路由系统衔接：桥上报的 inbound 消息 → 核心路由 → 接入器推送；
接入器回传的回复 → 核心出站指令 → 桥发送到平台。

---

## 4. 接入器协议设计（对外 WS 服务端口 + 令牌）

### 4.1 连接端点

```
ws://<esp-host>:<esp-port>/ws/adapter-gateway?token=<ACCESS_TOKEN>
```

- **Token 必须通过查询参数传递**（WebSocket 握手阶段没有标准 Header 机制，兼容所有客户端）。
- 也可支持 `Sec-WebSocket-Protocol` 子协议头携带令牌（进阶，可选实现）。
- 握手成功后，服务器下发 `connected` 事件，包含 `adapter_id`、`gateway_version`。

### 4.2 认证与令牌

#### 4.2.1 令牌模型

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 令牌唯一 ID（UUID） |
| `name` | string | 令牌名称（如「客服接入器-1」） |
| `token` | string | 令牌明文（**仅创建时返回一次**） |
| `token_hash` | string | 令牌 SHA-256 哈希（存储用） |
| `platform` | string | 关联平台（`taobao` / `xianyu` / 空=全部） |
| `scope` | string | 权限范围：`read` / `write` / `read+write` |
| `status` | string | `active` / `revoked` |
| `expires_at` | *time | 过期时间（空=永不过期） |
| `last_used_at` | *time | 最后使用时间 |
| `created_at` | time | 创建时间 |
| `created_by` | string | 创建者用户 ID |

#### 4.2.2 令牌生命周期

- **创建**：管理员通过 API `POST /api/v1/adapter-tokens` 创建，响应返回明文 token（仅一次）。
- **认证**：接入器握手时携带 token，网关用 SHA-256 哈希比对 `token_hash`，校验状态为
  `active` 且未过期。
- **吊销**：`DELETE /api/v1/adapter-tokens/:id` 将状态置为 `revoked`，已连入的连接会被
  网关强制断开（下发 `error` 事件后关闭）。
- **轮换**：创建新令牌 → 客户端重连 → 吊销旧令牌。

#### 4.2.3 安全要求

- 令牌明文只显示一次，数据库只存哈希（参考密码学最佳实践）。
- 传输层必须启用 TLS（生产环境 `wss://`）。
- 可选：令牌绑定来源 IP 白名单。

### 4.3 消息帧格式（JSON over WS Text）

所有消息均为 JSON 对象，统一外层结构：

```jsonc
{
  "type": "message",           // 消息类型，见 4.4
  "id": "evt_01H...",          // 事件 ID（客户端生成，用于 ACK 关联）
  "timestamp": 1780000000000,  // Unix 毫秒时间戳
  "payload": { ... }           // 随 type 变化的负载
}
```

### 4.4 消息类型定义

#### 4.4.1 服务端 → 客户端

| type | 方向 | 说明 |
|------|------|------|
| `connected` | S→C | 握手成功，返回 `adapter_id`、`gateway_version`、`session_id` |
| `message` | S→C | 推送一条入站消息（来自电商平台） |
| `event` | S→C | 推送非消息事件（订单状态、商品咨询等，可选） |
| `ack` | S→C | 对客户端消息的确认（回显 `id`） |
| `pong` | S→C | 心跳响应 |
| `error` | S→C | 错误通知（含错误码） |

#### 4.4.2 客户端 → 服务端

| type | 方向 | 说明 |
|------|------|------|
| `message` | C→S | 发送一条出站消息（机器人回复） |
| `ping` | C→S | 心跳 |
| `ack` | C→S | 对服务端消息的确认 |
| `subscribe` | C→S | 订阅指定平台/实例的消息（可选） |

### 4.5 入站消息格式（S→C `message`）

```jsonc
{
  "type": "message",
  "id": "evt_01H...",
  "timestamp": 1780000000000,
  "payload": {
    "direction": "inbound",        // 入站
    "platform": "xianyu",          // 电商平台
    "bridge_id": "inst_xxx",       // 来源桥实例 ID
    "conversation_id": "conv_xxx", // 会话 ID（买家）
    "sender": { "id": "u_001", "name": "买家A", "platform": "xianyu" },
    "message_chain": [
      { "type": "text", "text": "这个商品有货吗？" },
      { "type": "image", "url": "https://..." }
    ],
    "raw": { }                     // 原始 ESPL v3 payload（可选透传）
  }
}
```

**字段说明**：
- `message_chain` 元素类型对齐 ESPL v3 的 [`MessageElement`](../internal/protocol/v3/message_chain.go:31)：
  `text` / `image` / `audio` / `video` / `file` / `product_card` / `order_info` / `inquiry` /
  `location` / `emoji` / `mention`。
- `sender.id` 是桥上报的买家 ID，`conversation_id` 是桥上报的会话 ID。

### 4.6 出站消息格式（C→S `message`）

```jsonc
{
  "type": "message",
  "id": "evt_02H...",
  "timestamp": 1780000000000,
  "payload": {
    "direction": "outbound",       // 出站
    "platform": "xianyu",          // 目标电商平台
    "conversation_id": "conv_xxx", // 回话会话（对应入站）
    "reply_to": "evt_01H...",      // 回复的入站消息 ID（可选）
    "message_chain": [
      { "type": "text", "text": "有货的，亲~" }
    ]
  }
}
```

**核心处理流程**（后端）：
1. 网关校验令牌的 `write` 权限。
2. 根据 `platform` + `conversation_id` 解析目标桥实例（通过消息核心的路由规则或
   conversation→instance 映射）。
3. 将 `message_chain` 转换为 ESPL v3 出站指令（`send_text` / `send_image` 等），
   经消息核心下发到桥实例。
4. 返回 `ack`（含指令 ID，便于追踪发送结果）。

### 4.7 心跳与保活

- 客户端每 **30 秒**发送 `{"type":"ping"}`；服务端回复 `{"type":"pong"}`。
- 服务端每 60 秒未收到任何帧（含 ping）则判定超时，断开连接。
- 客户端断线后应指数退避重连（1s、2s、4s…上限 30s），重连使用同一 token。

### 4.8 错误码

| code | 说明 |
|------|------|
| `40101` | 无效或已吊销的令牌 |
| `40102` | 令牌已过期 |
| `40103` | 令牌权限不足（无 write/read） |
| `40001` | 消息格式错误 |
| `40401` | 目标平台/实例不存在 |
| `42901` | 发送频率超限 |
| `50001` | 内部错误 |

---

## 5. 与外部框架的对接

### 5.1 对接模型

E-SP-Line2 的接入器协议是**核心自有协议**，外部框架不能直接消费，因此需要
一层**适配器（连接器）**——在外部框架侧运行一个小型客户端进程，将接入器协议
转换为框架原生 API。

```
┌────────────┐   接入器协议(WS)   ┌──────────────────┐   框架原生 API   ┌──────────┐
│ E-SP-Line2 │ ◄───────────────► │ 连接器 (Connector)│ ◄──────────────► │ 外部框架  │
│ 接入器网关  │                   │  Python/JS 小进程 │                  │ (机器人等)│
└────────────┘                   └──────────────────┘                  └──────────┘
```

### 5.2 对接要点

外部框架通常提供两种接入方式：

1. **HTTP Bot Adapter**：外部系统 POST 消息到框架的 `/bots/<bot_uuid>`，框架通过回调
   URL 回传回复。**接入器连接器可选用此方式**：连接器收到 E-SP-Line2 推送的入站消息后，
   POST 到框架（带 HMAC 签名），并暴露回调 URL 接收回复，再回传 E-SP-Line2 接入器网关。

2. **WebSocket 调试/嵌入通道**：`ws://<framework>/api/v1/pipelines/<uuid>/ws/connect`，
   适合实时聊天 UI。**连接器也可选用此方式**，但会话标识受限于框架的 `connection_id`，
   不如 HTTP Bot 的 `session_id` 灵活。

> **推荐**：首选 **HTTP Bot Adapter**，原因：
> - 会话标识由外部（E-SP-Line2 的 `conversation_id`）指定，天然支持多买家隔离。
> - 支持 N→1 消息聚合与 1→M 多段回复，贴合客服场景。
> - 无需维护长连接，签名机制成熟（HMAC-SHA256）。

### 5.3 消息链映射表（ESPL v3 ↔ 外部框架）

| ESPL v3 元素 | 框架段（如 OneBot） |
|--------------|---------------------|
| `text` | `text` |
| `image` | `image` |
| `audio` | `record` |
| `file` | `file` |
| `mention` | `at` |
| `quote`（预留） | `reply` |
| `product_card` / `order_info` / `inquiry` | 自定义段（或降级为文本卡片） |

---

## 6. 数据模型与服务层设计

### 6.1 新增：接入器令牌模型

```go
// AdapterToken represents an access token for the adapter gateway.
type AdapterToken struct {
    ID          string     `json:"id" gorm:"primaryKey"`
    Name        string     `json:"name" gorm:"not null"`
    TokenHash   string     `json:"-" gorm:"not null;uniqueIndex"` // SHA-256 hash only
    Platform    string     `json:"platform"`                      // taobao / xianyu / "" (all)
    Scope       string     `json:"scope" gorm:"default:'read+write'"`
    Status      string     `json:"status" gorm:"default:'active'"` // active, revoked
    ExpiresAt   *time.Time `json:"expires_at"`
    LastUsedAt  *time.Time `json:"last_used_at"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    CreatedBy   string     `json:"created_by"`
}
```

### 6.2 新增：接入器网关组件

```
internal/adaptergateway/          # 新增包（或 internal/adapter/gateway.go）
├── gateway.go                    # WS 服务端：升级、令牌认证、连接管理
├── token.go                      # 令牌创建/吊销/校验（hash 比对）
├── protocol.go                   # 接入器协议帧编解码
├── hub.go                        # 连接注册表（类似 internal/message/hub.go）
└── converter.go                  # ESPL v3 ↔ 接入器协议 转换
```

### 6.3 与现有消息核心的衔接

| 场景 | 接入器网关行为 |
|------|---------------|
| 桥上报 inbound 消息 | 消息核心入库后，将消息投递给**匹配该平台**的所有已连入接入器（fan-out） |
| 接入器回传出站消息 | 网关调用消息核心的出站指令接口，路由到目标桥实例 |
| 接入器断线重连 | 网关按 token 恢复连接，消息核心对未送达消息做队列缓冲（可选） |
| 桥实例下线 | 消息核心暂停向该平台接入器推送，恢复后继续 |

### 6.4 API 设计

```
# 接入器令牌管理
POST   /api/v1/adapter-tokens          # 创建令牌（返回明文，仅一次）
GET    /api/v1/adapter-tokens          # 令牌列表（不含明文）
GET    /api/v1/adapter-tokens/:id      # 令牌详情
DELETE /api/v1/adapter-tokens/:id      # 吊销令牌

# 接入器连接状态（新增）
GET    /api/v1/adapter-connections     # 当前接入器连接列表（token 名、平台、在线状态）

# WebSocket（对外服务端）
GET    /ws/adapter-gateway?token=xxx   # 接入器网关入口
```

---

## 7. 迁移路径（分阶段实施）

### 阶段 0：概念冻结（本文档）
- 明确「桥」与「接入器」定义，冻结本文档为架构基线。

### 阶段 1：桥重命名（纯重命名，不改行为）
- `adapters/` → `bridges/`，`adapter.yaml` → `bridge.yaml`。
- 后端模型/服务/路由重命名，保留旧别名与数据库表名兼容。
- `/ws/adapter` → `/ws/bridge`（保留旧路径兼容）。
- 更新文档与 WebUI 文案。

### 阶段 2：接入器令牌管理（后端）
- 新增 `AdapterToken` 模型、repository、service、handler。
- 新增令牌 CRUD API 与权限控制（仅 admin）。

### 阶段 3：接入器网关（后端）
- 实现 `internal/adaptergateway/` WS 服务端。
- 实现令牌认证、连接管理、心跳、协议编解码。
- 接入消息核心：inbound fan-out + outbound 指令下发。

### 阶段 4：外部框架连接器（外部）
- 实现 HTTP Bot 连接器（推荐路径）。
- 实现 OneBot 等协议连接器（可选）。
- 端到端联调：闲鱼消息 → 桥 → 核心 → 接入器 → 外部框架 → 大模型 → 回复 → 桥 → 闲鱼。

### 阶段 5：WebUI 与管理
- 接入器令牌管理页。
- 接入器连接状态监控页。

---

## 8. 影响面分析

### 8.1 需要改动的现有文件

| 文件 | 改动 |
|------|------|
| [`internal/models/models.go`](../internal/models/models.go:1) | 新增 `AdapterToken`；`AdapterPackage`/`AdapterInstance` 考虑更名（或加别名） |
| [`internal/service/`](../internal/service/) | `AdapterService` → `BridgeService`（或新增别名）；新增 token service |
| [`internal/handler/`](../internal/handler/) | 新增 token handler；`/ws/adapter` 路由保留兼容 |
| [`internal/server/server.go`](../internal/server/server.go:65) | 新增 `/ws/adapter-gateway` 路由与 token API |
| [`internal/message/`](../internal/message/) | Dispatcher 增加 `adapter-gateway` 连接类型与 fan-out 逻辑 |
| [`internal/config/config.go`](../internal/config/config.go:10) | 新增 `adapter_gateway` 配置块（端口、心跳、超时） |
| [`web/src/pages/`](../web/src/pages/) | 新增令牌管理页（可选） |

### 8.2 不改动的部分
- 桥的 Python 进程（`adapters/taobao/main.py` 等）逻辑保持不变，仅目录/命名调整。
- ESPL v3 协议本身不变（接入器协议是其子集扩展）。
- 现有 `/ws/app` 下游应用入口保持不变。

### 8.3 风险
- **重命名风险**：大量引用 `adapter` 的地方需同步更新，建议先做全量搜索
  （`internal/`、`web/`、`docs/`），用编译器 + 测试保障。
- **兼容风险**：已部署桥进程使用旧 `/ws/adapter` 路径，需保留旧路由至少一个版本周期。
- **安全风险**：接入器网关是公网暴露面，令牌管理与 TLS 必须严格。

---

## 9. 开放问题（Open Questions）

1. 接入器令牌是否需要支持**多平台多令牌绑定一个框架**（一个框架同时接入淘宝+闲鱼）？
   → 设计上支持：令牌 `platform` 留空表示全部，网关按平台 fan-out。
2. 接入器协议是否需要支持**流式回复**（大模型边生成边回传）？
   → 建议 v1 先支持整段回复，v2 增加 `stream` 分片（对齐框架回调的 `stream` 字段）。
3. 消息核心对**未送达的接入器消息**是否需要持久化队列？
   → 建议 v1 先做内存缓冲 + 断线丢弃（at-most-once），v2 再做持久化（at-least-once）。
4. 接入器网关是否要支持 **HTTP 长轮询/SSE 备选**？
   → 部分框架（如某些 SaaS）不便维护 WS，可后续增加 HTTP 兼容入口。
5. 「桥」重命名是否要连同数据库表名一起迁移？
   → 建议 GORM 用 `TableName()` 保留旧表名，仅改 Go 结构体名，降低迁移成本。

---

## 10. 参考

- [E-SP-Line2 V3 协议文档](developer-guide/protocol-v3.md)
- [E-SP-Line2 接入器 YAML 规则](developer-guide/adapter-yaml.md)
- [E-SP-Line2 接入器网关文档](user-guide/adapter-gateway.md)
