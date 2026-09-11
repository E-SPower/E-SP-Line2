# 非活动字段注册机制（Inactive Field Registration）

> 面向 LangBot 等下游框架的 **主动查询能力**。消息里「顺带」携带的字段是**活动字段**（被动推送）；
> 而商品简介、图片列表、订单详情、物流轨迹、活动信息等**只有在被需要时才去取**的数据，称为
> **非活动字段**。ESPL v3 通过「工具（Tool）」把非活动字段注册为可调用的接口。

本文覆盖：

- 工具目录（内置 **25 个**非活动字段工具：基础 7 个 + XianYuApis 兼容 12 个 + TaoBaoApis 兼容 6 个）
- 协议帧：`tool_catalog` / `tool_call` / `tool_result`
- HTTP API 调用方式
- 桥（Bridge）如何实现工具
- LangBot 适配器如何调用（`call_tool` 与便捷包装方法）
- XianYuApis 原版兼容：商品发布字段、会话/消息字段、交易卡片、音频消息
- TaoBaoApis 原版兼容：登录 token、商品链接解析、媒体上传、会话/消息（`taobao_*` 前缀工具）
- 活动 / 事件信息接收（`order.*` / `logistics.*` / `item.*` / `trade.card.received` / `activity.received` 等 **20 个**事件）
- 四层权限模型与错误码

---

## 1. 机制总览

```
┌──────────────┐   tool_call    ┌────────────────────┐   tool_call    ┌──────────┐
│   LangBot    │ ─────────────▶ │  E-SP-Line2 网关    │ ─────────────▶ │ 桥/Bridge │
│  (下游框架)   │                │  (ToolRegistry)    │                │ (闲鱼/淘宝)│
│              │ ◀───────────── │   pendingTools     │ ◀───────────── │          │
└──────────────┘   tool_result  └────────────────────┘   tool_result  └──────────┘
        │                                │
        │  HTTP: POST /api/v1/tools/call │
        └────────────────────────────────┘
```

- **注册**：工具定义（`ToolDefinition`）注册到网关的 `ToolRegistry`。内置目录由
  [`DefaultESPL2Tools()`](../internal/protocol/v3/tool.go)（基础电商）、
  [`DefaultXianYuTools()`](../internal/protocol/v3/tool.go)（XianYuApis 原版兼容）与
  [`DefaultTaoBaoTools()`](../internal/protocol/v3/tool.go)（TaoBaoApis 原版兼容，`taobao_` 前缀）提供；
  接入器（Bridge/适配器）可在连接建立后通过 `tool_catalog` 帧动态注册自己的工具。
- **发现**：下游框架通过 HTTP `GET /api/v1/tools` 或 WS `tool_catalog` 帧获取工具清单。
- **调用**：下游框架发 `tool_call`（或 HTTP `POST /api/v1/tools/call`）→ 网关路由到持有该
  `instance_id` 的桥 → 桥执行并回 `tool_result` → 网关按 `call_id` 匹配并回传给调用方。
- **事件**：订单/物流/商品/会话/交易卡片/登录等生命周期事件由网关**主动广播**（无需轮询），见第 7 节。

---

## 2. 工具目录

### 2.1 基础电商工具（`DefaultESPL2Tools`，7 个）

| 工具名 | 语义 ID | 类别 | 说明 | 关键参数 |
| --- | --- | --- | --- | --- |
| `get_product` | `get_product` | query_only | 商品详情：名称、简介、图片、价格、规格、库存 | `item_id`(必填) |
| `get_product_list` | `get_product_list` | query_only | 商品搜索/列表 | `keyword`/`category`/`page`/`page_size` |
| `get_order` | `get_order` | query_only | 订单信息：状态、金额、商品明细、支付信息 | `order_id`(必填) |
| `get_logistics` | `get_logistics` | query_only | 物流/发货信息：物流单号、承运商、轨迹 | `order_id`(必填)、`tracking_no` |
| `ship_order` | `ship_order` | query_action | 对订单发货并回填物流单号（**有副作用**） | `order_id`、`tracking_no`(必填) |
| `update_order` | `update_order` | query_action | 更新订单状态（取消/退款/确认收货，**有副作用**） | `order_id`、`status`(必填) |
| `get_activity` | `get_activity` | query_only | 活动/事件信息（促销、秒杀、平台活动） | `activity_id`/`item_id` |

### 2.2 XianYuApis 原版兼容 —— HTTP 接口工具（9 个）

对应 `goofish_apis.py` 中的 HTTP（mtop / upload）调用。

| 工具名 | 语义 ID | 类别 | 对应原版接口 | 关键参数 |
| --- | --- | --- | --- | --- |
| `publish_item` | `publish_item` | query_action | `mtop.idle.pc.idleitem.publish` | `title`/`description`/`images`/`price_in_cent`(必填) |
| `update_item` | `update_item` | query_action | `mtop.taobao.idle.item.update` | `item_id`(必填)、`title`/`description`/`price_in_cent`/`quantity`/`status` |
| `update_item_price` | `update_item_price` | query_action | 商品改价（item.update 子集） | `item_id`/`price_in_cent`(必填)、`orig_price_in_cent` |
| `get_category_recommend` | `get_category_recommend` | query_only | `mtop.taobao.idle.kgraph.property.recommend` | `title`(必填)、`description` |
| `get_default_address` | `get_default_address` | query_only | `mtop.taobao.idle.local.poi.get` | `gps`/`division_id` |
| `upload_media` | `upload_media` | query_action | `stream-upload.goofish.com/api/upload.api` | `file_path`/`url`、`content_type` |
| `refresh_token` | `refresh_token` | query_action | `mtop.taobao.idlemessage.pc.loginuser.get` | `cookies` |
| `get_token` | `get_token` | query_only | `mtop.taobao.idlemessage.pc.login.token` | `cookies`(必填) |
| `login_qrcode` | `login_qrcode` | query_action | 扫码登录（get / poll） | `action`、`session_id` |

### 2.3 XianYuApis 原版兼容 —— lwp WebSocket 命令工具（3 个）

对应 `goofish_live.py` 中通过 `/r/...` lwp 协议发送的命令。

| 工具名 | 语义 ID | 类别 | 对应原版 lwp 命令 | 关键参数 |
| --- | --- | --- | --- | --- |
| `get_conversation_history` | `get_conversation_history` | query_only | `/r/MessageManager/listUserMessages` | `cid`(必填)、`next_cursor`、`count` |
| `create_conversation` | `create_conversation` | query_action | `/r/SingleChatConversation/create` | `pair_first`/`pair_second`(必填)、`biz_type`、`item_id`、`ctx` |
| `send_message` | `send_message` | query_action | `/r/MessageSend/sendByReceiverScope` | `cid`(必填)、`content_type`、`text`/`image_url`/`audio_url`/`custom`、`uuid`、`ext_json` 等 |

> 除上述 12 个原版兼容工具外，lwp 通道还保留了原版的 `/reg`、`/r/SyncStatus/ackDiff`、
> `!` 心跳等底层命令（见 [`command.go`](../internal/protocol/v3/command.go) 的
> `CommandTypeRegister` / `CommandTypeAckDiff` / `CommandTypeHeartbeat`），它们由桥在连接
> 生命周期内自动发送，无需下游显式调用。

### 2.4 TaoBaoApis 原版兼容工具（`DefaultTaoBaoTools`，6 个）

对应 [`TaoBaoApis/taobao_apis.py`](../../../TaoBaoApis/taobao_apis.py) 的 HTTP（mtop / upload）
接口与 [`TaoBaoApis/taobao_live.py`](../../../TaoBaoApis/taobao_live.py) 的 lwp 命令。
淘宝与闲鱼的**语义 ID 相同**（同样的业务意图可跨平台解析），但工具名统一加 `taobao_` 前缀，
以避免与 XianYuApis 工具在网关的 `ToolRegistry`（按 name 索引）中互相覆盖；同时每个定义都
带 `"platform": "taobao"`，调用方可按平台过滤。

| 工具名 | 语义 ID | 类别 | 对应原版接口/命令 | 关键参数 |
| --- | --- | --- | --- | --- |
| `taobao_get_token` | `get_token` | query_only | `mtop.taobao.login.token.get.h5` | `cookies`(必填) |
| `taobao_get_goods_info` | `get_goods_info` | query_only | 商品链接解析（itemInfo） | `goods_url`(必填) |
| `taobao_upload_media` | `upload_media` | query_action | `stream-upload.taobao.com/api/upload.api` | `file_path`(必填) |
| `taobao_get_conversation_history` | `get_conversation_history` | query_only | `/r/MessageManager/listUserMessages` | `cid`(必填)、`next_cursor`、`count` |
| `taobao_create_conversation` | `create_conversation` | query_action | `/r/SingleChatConversation/create` | `encrypt_uid`(必填) |
| `taobao_send_message` | `send_message` | query_action | `/r/MessageSend/sendByReceiverScope` | `cid`/`toid`(必填)、`content_type`、`text`/`image_url`/`file_id`、`sender_nick` 等 |

> 与闲鱼的差异：淘宝的会话标识域为 `@cntaobao`（闲鱼为 `@goofish`），发送者昵称为
> `cntaobao{_nk_}`；创建会话使用 `encryptUid`（闲鱼使用 `pair_first`/`pair_second`）。
> lwp 基址为 `wss://wss-cntaobao.dingtalk.com/`。

**类别语义**

- `query_only`：幂等查询，无副作用，可自由调用。
- `query_action`：产生状态变更，**必须** `confirm=true` 才执行；接入器还需具备写权限。

工具定义结构（Go 侧 [`ToolDefinition`](../internal/protocol/v3/tool.go)）：

```json
{
  "name": "get_logistics",
  "semantic_id": "get_logistics",
  "category": "query_only",
  "description": "查询物流/发货信息：物流单号、承运商、轨迹。",
  "parameters": [
    { "name": "order_id",    "type": "string",  "required": true,  "description": "订单号" },
    { "name": "tracking_no", "type": "string",  "required": false, "description": "物流单号（可选）" }
  ],
  "returns": "LogisticsInfo{ order_id,tracking_no,carrier,status,traces[] }",
  "enabled": true,
  "require_confirm": false,
  "platform": ""
}
```

参数类型支持：`string` / `number` / `boolean` / `object` / `array`；可选字段可带 `default`、`enum`。

---

## 3. 协议帧

### 3.1 `tool_catalog`（双向）

`tool_catalog` 帧是**双向**的：既可以由连接方（桥 / 接入框架）**上报**自己支持的工具，
也可以由网关在连接建立后**下发**当前已注册的工具目录。

#### 3.1.1 连接方 → 网关（上报）

连接方连接后（或运行时）声明自己支持的工具。E-SP-Line2 网关收到后注册进 `ToolRegistry`。
三条链路都支持该帧：

| 链路 | 入口 | 处理代码 |
| --- | --- | --- |
| 桥（Bridge） | `/ws/adapter?instance_id=xxx` | [`handler/websocket.go`](../../internal/handler/websocket.go) `registerToolsFromFrame` |
| 接入框架（server 模式） | `/ws/adapter-gateway?key=xxx` | [`adaptergateway/client.go`](../../internal/adaptergateway/client.go) `handleMessage` |
| 接入框架（client 模式） | E-SP-Line2 主动外拨到框架 | [`adaptergateway/client_connector.go`](../../internal/adaptergateway/client_connector.go) `handleInbound` |

三者最终都调用同一个解析器
[`Gateway.RegisterToolsFromCatalogFrame()`](../../internal/adaptergateway/tool_router.go)，
因此工具定义结构完全一致：

```json
{
  "type": "tool_catalog",
  "id": "a1b2c3d4",
  "timestamp": 1730000000000,
  "adapter_id": "xianyu-1",
  "tools": [
    {
      "name": "get_logistics",
      "semantic_id": "get_logistics",
      "category": "query_only",
      "description": "查询物流/发货信息",
      "parameters": [ { "name": "order_id", "type": "string", "required": true } ],
      "returns": "LogisticsInfo{...}",
      "enabled": true,
      "require_confirm": false,
      "platform": "xianyu"
    }
  ]
}
```

> 工具列表既可放在顶层 `tools`，也可放在 `payload.tools`（两种写法都兼容）。
> 收到后网关回一个 `ack` 帧（`{"type":"ack","id":<原帧 id>}`）。

> LangBot 适配器 [`espl.py`](../../../esplplatfrom/espl.py) 在 `connected` 握手后**立即**发送
> `tool_catalog`，内容取自 `ESPL_TOOL_DEFINITIONS`（基础 7 + XianYuApis 兼容 12 + TaoBaoApis 兼容 6，共 25 个）。

#### 3.1.2 网关 → 连接方（下发）

网关在发送 `connected` 握手后**立即**把当前注册的完整工具目录推给连接方，接入框架无需额外
HTTP 往返即可发现可调用工具。下发帧结构：

```json
{
  "type": "tool_catalog",
  "id": "<connection_id>",
  "timestamp": 1730000000000,
  "tools": [ { "name": "publish_item", "category": "query_action", "...": "..." } ]
}
```

- server 模式：[`Client.sendConnected()`](../../internal/adaptergateway/client.go) →
  `sendToolCatalog()`
- client 模式：[`outboundClient.sendConnected()`](../../internal/adaptergateway/client_connector.go) →
  `sendToolCatalog()`

接入框架侧（例如 LangBot 的 `espl.py`）收到后应缓存，供 LLM 工具选择使用：

```python
# espl.py：_handle_frame 中
if msg_type == 'tool_catalog':
    self._handle_tool_catalog(frame)   # 合并进 self.gateway_tools
    return

# 之后可查询：
tools = adapter.list_gateway_tools()              # 全部
tools = adapter.list_gateway_tools('query_action')  # 仅有副作用的工具
tool  = adapter.get_gateway_tool('publish_item')    # 单个定义
```

### 3.2 `tool_call`（调用方 → 网关）

```json
{
  "type": "tool_call",
  "id": "call_xxx",
  "timestamp": 1730000000000,
  "payload": {
    "call_id": "call_xxx",
    "tool": "get_logistics",
    "semantic_id": "get_logistics",
    "instance_id": "xianyu-1",
    "conversation_id": "conv_123",
    "arguments": { "order_id": "2024001" },
    "confirm": false,
    "timeout_ms": 15000
  }
}
```

网关将其转换为**桥协议**帧（`type: tool_call`，`command_type: tool_call`）下发到
`/ws/adapter?instance_id=xianyu-1` 的连接：

```json
{
  "type": "tool_call",
  "command_type": "tool_call",
  "id": "call_xxx",
  "call_id": "call_xxx",
  "instance_id": "xianyu-1",
  "timestamp": 1730000000000,
  "payload": {
    "call_id": "call_xxx",
    "tool": "get_logistics",
    "semantic_id": "get_logistics",
    "category": "query_only",
    "instance_id": "xianyu-1",
    "conversation_id": "conv_123",
    "arguments": { "order_id": "2024001" },
    "trace_id": ""
  }
}
```

### 3.3 `tool_result`（桥 → 网关 → 调用方）

桥在 `/ws/adapter` 回帧（`type: tool_result`）：

```json
{
  "type": "tool_result",
  "id": "call_xxx",
  "timestamp": 1730000000123,
  "payload": {
    "call_id": "call_xxx",
    "tool": "get_logistics",
    "semantic_id": "get_logistics",
    "success": true,
    "data": {
      "order_id": "2024001",
      "tracking_no": "SF1234567890",
      "carrier": "顺丰速运",
      "carrier_code": "SF",
      "status": "in_transit",
      "traces": [
        { "time": 1730000000000, "status": "in_transit", "description": "已揽收", "location": "杭州" }
      ]
    },
    "error": "",
    "error_code": "",
    "duration_ms": 123
  }
}
```

网关按 `instance_id|call_id` 匹配挂起的调用，随后把结果回传给发起方（下游框架）：

```json
{
  "type": "tool_result",
  "id": "call_xxx",
  "timestamp": 1730000000200,
  "payload": {
    "call_id": "call_xxx",
    "tool": "get_logistics",
    "semantic_id": "get_logistics",
    "success": true,
    "data": { "...": "同上" },
    "error": "",
    "error_code": "",
    "duration_ms": 125
  }
}
```

---

## 4. HTTP API 调用方式

所有接口都需要 JWT 认证：请求头 `Authorization: Bearer <token>`。

### 4.1 列出工具

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/api/v1/tools"
# 仅列出已启用的工具：
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/api/v1/tools?enabled=true"
```

响应：

```json
{
  "data": [ { "name": "get_product", "semantic_id": "get_product", "category": "query_only", "...": "..." } ],
  "total": 19
}
```

### 4.2 启用 / 禁用工具（权限模型第 1 层）

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"enabled": false}' \
  "http://127.0.0.1:8080/api/v1/tools/ship_order/enabled"
```

### 4.3 调用工具

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{
        "call_id": "call_abc",
        "tool": "get_order",
        "instance_id": "xianyu-1",
        "conversation_id": "conv_123",
        "arguments": { "order_id": "2024001" },
        "timeout_ms": 15000
      }' \
  "http://127.0.0.1:8080/api/v1/tools/call"
```

响应（HTTP 200，业务结果在 `data` 内）：

```json
{
  "data": {
    "call_id": "call_abc",
    "tool": "get_order",
    "semantic_id": "get_order",
    "success": true,
    "data": { "order_id": "2024001", "status": "paid", "amount": 199.0, "items": [] },
    "duration_ms": 88
  }
}
```

有副作用的工具必须带 `confirm`：

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{
        "tool": "ship_order",
        "instance_id": "xianyu-1",
        "arguments": { "order_id": "2024001", "tracking_no": "SF1234567890" },
        "confirm": true
      }' \
  "http://127.0.0.1:8080/api/v1/tools/call"
```

### 4.4 发布商品（XianYuApis 兼容）

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{
        "tool": "publish_item",
        "instance_id": "xianyu-1",
        "confirm": true,
        "arguments": {
          "title": "全新机械键盘 87 键",
          "description": "仅拆封未使用，原盒原装",
          "images": [
            { "url": "https://img.alicdn.com/xxx.jpg", "width": 1200, "height": 900 }
          ],
          "price_in_cent": 19900,
          "orig_price_in_cent": 29900,
          "quantity": 1,
          "item_type_str": "b",
          "simple_item": false,
          "item_cat": { "cat_id": "5001", "cat_name": "键盘", "channel_cat_id": "1", "tb_cat_id": "5001" },
          "item_addr": { "area": "西湖区", "city": "杭州", "prov": "浙江", "division_id": "330106", "poi_id": "p1", "poi_name": "公司" },
          "item_post_fee": { "can_free_shipping": true, "support_freight": true, "only_take_self": false, "post_price_in_cent": 0 },
          "delivery": { "choice": "包邮", "post_price_in_cent": 0, "can_self_pickup": false },
          "user_rights_protocols": [ { "enable": true, "service_code": "svc_7day" } ]
        }
      }' \
  "http://127.0.0.1:8080/api/v1/tools/call"
```

### 4.5 拉取会话历史 / 发送消息（lwp 兼容）

```bash
# 拉取历史（游标分页）
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"tool":"get_conversation_history","instance_id":"xianyu-1",
       "arguments":{"cid":"cid_abc","count":20}}' \
  "http://127.0.0.1:8080/api/v1/tools/call"

# 发送文本消息
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"tool":"send_message","instance_id":"xianyu-1","confirm":true,
       "arguments":{"cid":"cid_abc","content_type":1,"text":"您好，商品还在的",
                    "uuid":"uuid-xxx","ext_json":"{}"}}' \
  "http://127.0.0.1:8080/api/v1/tools/call"
```

> `call_id` 可省略，网关会自动生成。`tool` 与 `semantic_id` 二者至少提供一个。

---

## 5. 桥（Bridge）实现工具

桥在 `/ws/adapter?instance_id=<ID>` 连接上接收 `tool_call` 帧，执行后回 `tool_result`。

Python 伪代码：

```python
async def on_frame(frame):
    if frame.get("type") != "tool_call":
        return
    payload = frame["payload"]
    call_id = payload["call_id"]
    tool = payload["tool"]
    args = payload.get("arguments") or {}

    try:
        if tool == "get_logistics":
            data = await fetch_logistics(args["order_id"])
        elif tool == "ship_order":
            data = await do_ship(args["order_id"], args["tracking_no"])
        elif tool == "publish_item":
            data = await goofish.publish_item(**args)
        elif tool == "send_message":
            data = await goofish.send_message(**args)
        else:
            raise ValueError(f"unsupported tool: {tool}")
        result = {"success": True, "data": data}
    except Exception as e:
        result = {"success": False, "error": str(e), "error_code": "40406"}

    await ws.send(json.dumps({
        "type": "tool_result",
        "id": call_id,
        "timestamp": int(time.time() * 1000),
        "payload": {"call_id": call_id, "tool": tool, **result},
    }))
```

**约定**

- `call_id` 必须原样回传（网关据此匹配挂起调用）。
- 处理超时默认 **15 秒**（可由 `timeout_ms` 覆盖）；超时网关返回 `40403`。
- `success: false` 时请填 `error` 与 `error_code`，便于下游展示。

---

## 6. LangBot 适配器调用（`call_tool`）

LangBot 的 ESPL 适配器 [`espl.py`](../../../esplplatfrom/espl.py) 是 WS **服务端**，E-SP-Line2 以
**client 模式**连入。适配器暴露：

```python
result = await adapter.call_tool(
    tool="get_logistics",
    arguments={"order_id": "2024001"},
    instance_id="xianyu-1",
    conversation_id="conv_123",
    confirm=False,
    timeout=15.0,
)
if result.get("success"):
    tracking_no = result["data"]["tracking_no"]
```

内部流程：

1. `_pick_connection(instance_id)` 选中有该实例的连接（否则返回 `40402`）。
2. 生成 `call_id`，在 `conn.pending_tool_calls[call_id]` 建一个 `asyncio.Future`。
3. 发送 `tool_call` 帧。
4. `await asyncio.wait_for(future, timeout)`；收到 `tool_result` 时由
   `_handle_tool_result` → `resolve_tool_result(call_id, payload)` 完成 Future。
5. 连接断开时 `fail_pending_tool_calls('connection closed')` 统一失败所有挂起调用。

工具清单由 `ESPL_TOOL_DEFINITIONS` 定义，在连接握手后通过 `tool_catalog` 帧上报。

### 6.1 XianYu 便捷包装方法

适配器为每个兼容工具提供了语义化包装（内部均调用 `call_tool`）：

| 方法 | 等价工具 | 签名要点 |
| --- | --- | --- |
| `publish_item(**kwargs)` | `publish_item` | 自动 `confirm=True` |
| `update_item(item_id, **kwargs)` | `update_item` | 自动 `confirm=True` |
| `update_item_price(item_id, price_in_cent, **kwargs)` | `update_item_price` | 自动 `confirm=True` |
| `get_category_recommend(title, description='')` | `get_category_recommend` | 只读 |
| `get_default_address(gps='', division_id='')` | `get_default_address` | 只读 |
| `upload_media(file_path='', url='', content_type='image/jpeg')` | `upload_media` | 自动 `confirm=True` |
| `refresh_token(cookies='')` | `refresh_token` | 自动 `confirm=True` |
| `get_token(cookies)` | `get_token` | 只读 |
| `login_qrcode(action='get', session_id='')` | `login_qrcode` | `action='get'` 时需确认 |
| `get_conversation_history(cid, next_cursor='', count=20)` | `get_conversation_history` | 只读 |
| `create_conversation(pair_first, pair_second, biz_type='', item_id='', ctx='')` | `create_conversation` | 自动 `confirm=True` |
| `send_chat_message(cid, **kwargs)` | `send_message` | 自动 `confirm=True` |

示例：

```python
# 发布商品
await adapter.publish_item(
    title="全新机械键盘 87 键",
    description="仅拆封未使用",
    images=[{"url": "https://img.alicdn.com/xxx.jpg"}],
    price_in_cent=19900,
)

# 拉取会话历史并发送回复
history = await adapter.get_conversation_history("cid_abc", count=20)
await adapter.send_chat_message("cid_abc", content_type=1, text="您好，商品还在的")
```

### 6.2 TaoBao 便捷包装方法

淘宝工具名带 `taobao_` 前缀，因此包装方法也以 `taobao_` 命名（内部同样调用 `call_tool`）：

| 方法 | 等价工具 | 签名要点 |
| --- | --- | --- |
| `taobao_get_token(cookies)` | `taobao_get_token` | 只读 |
| `taobao_get_goods_info(goods_url)` | `taobao_get_goods_info` | 只读 |
| `taobao_upload_media(file_path)` | `taobao_upload_media` | 自动 `confirm=True` |
| `taobao_get_conversation_history(cid, next_cursor='', count=20)` | `taobao_get_conversation_history` | 只读 |
| `taobao_create_conversation(encrypt_uid)` | `taobao_create_conversation` | 自动 `confirm=True` |
| `taobao_send_message(cid, toid, **kwargs)` | `taobao_send_message` | 自动 `confirm=True` |

```python
# 解析商品页 → 建会话 → 发消息
info = await adapter.taobao_get_goods_info("https://item.taobao.com/item.htm?id=...")
conv = await adapter.taobao_create_conversation(info["data"]["encryptUid"])
await adapter.taobao_send_message(
    cid=conv["data"]["cid"], toid=info["data"]["userId"],
    content_type=1, text="您好，请问还在吗？",
)
```

### 6.3 出站消息组件映射

`_emit_outbound` 会把 LangBot `platform_message` 组件转换为 ESPL v3 `MessageChain` 元素：

| LangBot 组件 | ESPL 元素 `type` | 关键字段 |
| --- | --- | --- |
| `Plain` | `text` | `content.text` |
| `Image` | `image` | `content.url` |
| `Voice` | `audio` | `content.audio_url` / `content.url`（`duration_ms` 可选） |
| `File` | `file` | `content.url` / `content.name` |

---

## 7. 活动 / 事件信息接收

除「按需查询」外，网关还会**主动广播**电商生命周期事件（`event_type` 为下列之一）：

### 7.1 订单 / 物流 / 商品 / 活动（基础）

| `event_type` | 含义 | 关键字段 |
| --- | --- | --- |
| `order.created` | 新订单创建 | `order_id`, `product_id` |
| `order.paid` | 订单支付 | `order_id`, `amount` |
| `order.shipped` | 已发货 | `order_id`, `logistics.tracking_no` |
| `order.delivered` | 已签收 | `order_id`, `delivered_at` |
| `order.cancelled` | 订单取消 | `order_id`, `reason` |
| `order.refunded` | 退款 | `order_id`, `amount` |
| `logistics.updated` | 物流轨迹更新 | `order_id`, `logistics.traces[]` |
| `product.updated` | 商品信息变更 | `product_id`, `product.*` |
| `activity.received` | 活动/促销事件 | `activity_id`, `item_id`, `start_at`, `end_at` |

### 7.2 XianYuApis 兼容事件（新增）

| `event_type` | 常量 | 含义 | 关键字段 |
| --- | --- | --- | --- |
| `item.published` | `EventItemPublished` | 商品发布成功 | `item_id`, `status`, `publish_scene` |
| `item.updated` | `EventItemUpdated` | 商品信息更新 | `item_id`, `status`, `updated_at` |
| `item.price_updated` | `EventItemPriceUpdated` | 商品改价 | `item_id`, `price.current_price` |
| `item.off_shelf` | `EventItemOffShelf` | 商品下架 | `item_id`, `status` |
| `conversation.created` | `EventConversationCreated` | 会话创建 | `cid`, `conversation_type` |
| `trade.card.received` | `EventTradeCardReceived` | 收到交易卡片（订单卡片） | `cid`, `card.target_url`, `card.item.*` |
| `media.uploaded` | `EventMediaUploaded` | 媒体上传完成 | `url`, `pix`, `width`, `height` |
| `token.refreshed` | `EventTokenRefreshed` | 登录态刷新 | `cookies`, `user_id` |
| `category.recommended` | `EventCategoryRecommended` | 分类推荐返回 | `cat_id`, `cat_name`, `card_list[]` |
| `address.default` | `EventAddressDefault` | 默认地址返回 | `poi_id`, `poi_name`, `area`, `city`, `prov` |
| `login.qrcode` | `EventLoginQRCode` | 扫码登录状态变更 | `session_id`, `status`, `qr_url` |

事件信封（`MessageEnvelope`）：

```json
{
  "protocol_version": "v3",
  "event_id": "evt_xxx",
  "trace_id": "trace_xxx",
  "timestamp": 1730000000000,
  "platform": "xianyu",
  "adapter_id": "xianyu-1",
  "event_type": "trade.card.received",
  "payload": {
    "instance_id": "xianyu-1",
    "cid": "cid_abc",
    "card": {
      "content_type": 26,
      "target_url": "fleamarket://order_detail?id=2024001&role=seller",
      "item": { "title": "全新机械键盘", "desc": "订单已创建", "role": "seller" },
      "template": { "name": "idlefish_message_trade_chat_card" },
      "extension": { "biz_tag": "order" }
    }
  },
  "signature": "..."
}
```

**接收方式（LangBot 适配器）**：`_handle_frame` 识别 `event_type` 属于 `_ECOMMERCE_EVENTS`
（基础 9 + 兼容 11 = **20 个**）时调用 `_handle_ecommerce_event`，把最新事件存入
`adapter.ecommerce_events[event_type]`，并写日志。插件/监听器可读取：

```python
latest = adapter.latest_ecommerce_event("order.shipped")
# 或读取全部：adapter.latest_ecommerce_event()
tracking_no = latest["payload"]["logistics"]["tracking_no"]

card = adapter.latest_ecommerce_event("trade.card.received")
order_id = card["payload"]["card"]["target_url"]  # fleamarket://order_detail?id=...
```

**Go 侧广播**：业务代码调用
`gateway.BroadcastEvent(envelopeMap)`（内部转发到 `BroadcastInbound`），即可向所有匹配的
接入器客户端（server 模式与 client 模式）推送事件。信封字段与
[`MessageEnvelope`](../internal/protocol/v3/envelope.go) 一致；`event_type` 会透传到帧顶层
（由 `broadcastToAdapterGateway` 从 `payload["event_type"]` 推导，默认 `message.received`）。

**订阅式建议**：需要强一致处理时，优先用 `order.*` / `logistics.updated` / `trade.card.received`
事件驱动状态机，仅在事件缺少细节时再用 `get_order` / `get_logistics` / `get_conversation_history`
工具补查。

---

## 8. XianYuApis 兼容数据结构

所有结构定义于 [`ecommerce.go`](../internal/protocol/v3/ecommerce.go)（Go）与
[`espl.py`](../../../esplplatfrom/espl.py)（Python 侧工具参数）。

### 8.1 商品发布（`ItemPublish` 及其子 DTO）

| 结构 | 关键字段 | 说明 |
| --- | --- | --- |
| `ItemTextDTO` | `desc`, `title`, `title_desc_separate` | 标题/简介 |
| `ItemPriceDTO` | `price_in_cent`, `orig_price_in_cent` | 价格（**分**） |
| `ImageInfoDO` | `url`, `width`, `height` | 图片列表项 |
| `ItemCatDTO` | `cat_id`, `cat_name`, `channel_cat_id`, `tb_cat_id` | 分类 |
| `ItemAddrDTO` | `area`, `city`, `division_id`, `gps`, `poi_id`, `poi_name`, `prov` | 发货地址 |
| `ItemPostFeeDTO` | `can_free_shipping`, `support_freight`, `only_take_self`, `template_id`, `post_price_in_cent` | 运费模板 |
| `ItemLabelExt` | 标签扩展 | 商品标签 |
| `UserRightsProtocol` | `enable`, `service_code` | 用户权益协议 |
| `ItemPublish` | `freebies`, `item_type_str`(默认 `b`), `quantity`, `simple_item`, `image_info_do_list[]`, `item_text_dto`, `item_label_ext_list[]`, `item_price_dto`, `user_rights_protocols[]`, `item_post_fee_dto`, `item_addr_dto`, `default_price`, `item_cat_dto`, `unique_code`, `source_id`, `bizcode`, `publish_scene` | 发布请求体 |
| `ItemPublishResult` | `item_id`, `status`, `publish_scene` | 发布结果 |
| `ItemUpdateResult` | `item_id`, `status`, `updated_at` | 更新结果 |

**价格与运费辅助**

```go
// 价格（单位：分）
price := v3.CreatePrice(19900, 29900, "CNY")           // current_price / original_price / currency

// 运费设置（choice: 包邮 | 按距离计费 | 一口价 | 无需邮寄）
fee := v3.CreateDeliverySettings(v3.DeliveryChoiceFreeShipping, 0, true)
```

`DeliveryChoice` 取值常量：`DeliveryChoiceFreeShipping`(`包邮`)、
`DeliveryChoiceByDistance`(`按距离计费`)、`DeliveryChoiceFixedPrice`(`一口价`)、
`DeliveryChoiceNoPost`(`无需邮寄`)。

`ItemStatus` 取值常量：`ItemStatusOnSale`(`on_sale`)、`ItemStatusOffShelf`(`off_shelf`)、
`ItemStatusSoldOut`(`sold_out`)、`ItemStatusDraft`(`draft`)。

### 8.2 分类推荐 / 默认地址 / 媒体上传

| 结构 | 关键字段 | 说明 |
| --- | --- | --- |
| `CardListItem` | `card_id`, `card_name`, `card_type` | 推荐卡片项 |
| `CategoryPredictResult` | `cat_id`, `cat_name`, `channel_cat_id`, `tb_cat_id` | 分类预测结果 |
| `CategoryRecommend` | `cat_id`, `cat_name`, `channel_cat_id`, `tb_cat_id`, `card_list[]` | 分类推荐（对应 `kgraph.property.recommend`） |
| `DefaultAddress` | `poi_id`, `poi_name`, `area`, `city`, `prov`, `division_id`, `gps` | 默认发货地址（对应 `commonAddresses`） |
| `MediaUpload` | `url`, `pix`, `width`, `height`, `size` | 上传结果（对应 `object.url` / `object.pix`） |
| `TokenResult` | `cookies`, `user_id`, `display_name`, `access_token`, `refreshed_at` | 登录态 / token |
| `QRCodeLogin` | `qr_url`, `session_id`, `status`, `cookies` | 扫码登录状态（`pending`/`scanned`/`confirmed`/`expired`） |

### 8.3 会话与消息（lwp）

| 结构 | 关键字段 | 说明 |
| --- | --- | --- |
| `HistoryMessage` | `message_id`, `sender_id`, `content_type`, `content`, `created_at` | 单条历史消息 |
| `ConversationHistory` | `has_more`, `next_cursor`, `messages[]` | `listUserMessages` 响应 |
| `ConversationCreate` | `cid`, `conversation_type`, `created_at` | `SingleChatConversation.create` 响应 |
| `SendMessageResult` | `message_id`, `uuid`, `cid`, `sent_at` | `sendByReceiverScope` 响应 |

`ContentType` 取值常量（与闲鱼 `contentType` 数值一致）：

| 常量 | 值 | 含义 |
| --- | --- | --- |
| `ContentTypeText` | `"1"` | 文本 |
| `ContentTypeImage` | `"2"` | 图片 |
| `ContentTypeAudio` | `"3"` | 语音 |
| `ContentTypeFile` | `"4"` | 文件 |
| `ContentTypeTradeCard` | `"26"` | 交易卡片（dxCard） |

### 8.4 交易卡片（dxCard，`contentType=26`）

对应闲鱼订单卡片消息 `dxCard`。

| 结构 | 关键字段 | 说明 |
| --- | --- | --- |
| `TradeCardButton` | `text`, `url` | 卡片按钮 |
| `TradeCardItem` | `main`, `target_url`, `ex_content` | 卡片主体 |
| `TradeCardTemplate` | `name` | 模板名，默认 `idlefish_message_trade_chat_card` |
| `TradeCard` | `content_type`(26), `target_url`, `item`, `template`, `extension`, `close_push_receiver`, `close_unread_number`, `detail_notice`, `ext_json` | 交易卡片 |

辅助函数：

```go
// 生成闲鱼订单详情深链：fleamarket://order_detail?id=<orderID>&role=<role>
url := v3.TradeCardURL("2024001", v3.TradeCardRoleSeller)

// 生成最小交易卡片
card := v3.CreateTradeCard("2024001", v3.TradeCardRoleSeller, "全新机械键盘", "订单已创建")
```

`TradeCardRole` 取值常量：`TradeCardRoleSeller`(`seller`)、`TradeCardRoleBuyer`(`buyer`)。

### 8.5 消息链元素

ESPL v3 `MessageChain` 元素类型（`ElementType`，见
[`message_chain.go`](../internal/protocol/v3/message_chain.go)）在原有基础上新增 `trade_card`：

| `type` | 结构 | 关键字段 |
| --- | --- | --- |
| `text` | `TextContent` | `text` |
| `image` | `ImageContent` | `url`, `width`, `height` |
| `audio` | `AudioContent` | `url` / `audio_url`, `duration` / `duration_ms` |
| `trade_card` | `TradeCard` | `target_url`, `item`, `template` |
| `product_card` | `ProductContent` | 商品卡片（标题/价格/图片/链接） |

Go 侧构造方法：`AddText` / `AddImage` / `AddAudio` / `AddProductCard` / `AddTradeCard`；
读取方法：`GetTexts` / `GetImages` / `GetAudioContents` / `GetTradeCards`。

`AddAudio` 会同步 `URL`↔`AudioURL`、`Duration`↔`DurationMS`，兼容 LangBot `Voice` 组件
（`url` + `length`）。

---

## 9. 权限模型（四层）

| 层 | 位置 | 行为 |
| --- | --- | --- |
| 1. 工具开关 | `ToolRegistry.SetEnabled` / `PUT /api/v1/tools/:name/enabled` | 禁用后调用返回 `40408` |
| 2. 路径权限 | 网关/接入器配置 | 决定哪些接入器可接收 `tool_call` |
| 3. 细粒度鉴权 | 适配器（如 `client.go` 的 `ToolRequiresWrite`） | `query_action` 工具需写权限，否则 `40405` |
| 4. 审计日志 | `logger` 输出 | 每次调用记录 tool / instance / call_id / 结果 |

有副作用的工具还需 **`confirm=true`**（`40407`），用于防止误操作。XianYuApis 兼容工具中，
`publish_item` / `update_item` / `update_item_price` / `upload_media` / `refresh_token` /
`login_qrcode` / `create_conversation` / `send_message` 均为 `query_action`，必须显式确认。
淘宝侧对应的 `taobao_upload_media` / `taobao_create_conversation` / `taobao_send_message`
同理（`taobao_get_token` / `taobao_get_goods_info` / `taobao_get_conversation_history` 为只读）。

---

## 10. 错误码

| 错误码 | 常量 | 含义 |
| --- | --- | --- |
| `40401` | `ToolErrNotFound` | 工具不存在 |
| `40402` | `ToolErrBridgeOffline` | 桥离线 / 无可用连接 |
| `40403` | `ToolErrTimeout` | 调用超时 |
| `40404` | `ToolErrInvalidArgs` | 参数缺失或非法 |
| `40405` | `ToolErrPermission` | 权限不足（需写权限） |
| `40406` | `ToolErrUpstream` | 上游执行失败 |
| `40407` | `ToolErrConfirmRequired` | 需要 `confirm=true` |
| `40408` | `ToolErrDisabled` | 工具已被禁用 |

---

## 11. 相关源码

- 工具声明 / 注册表：[`internal/protocol/v3/tool.go`](../internal/protocol/v3/tool.go)
  （`DefaultESPL2Tools` 基础 7 + `DefaultXianYuTools` 兼容 12 + `DefaultTaoBaoTools` 兼容 6）
- 工具路由 / 挂起匹配：[`internal/adaptergateway/tool_router.go`](../internal/adaptergateway/tool_router.go)
- 事件类型：[`internal/protocol/v3/envelope.go`](../internal/protocol/v3/envelope.go)
- 命令类型（含 lwp `reg`/`ackDiff`/`heartbeat`）：[`internal/protocol/v3/command.go`](../internal/protocol/v3/command.go)
- 消息链元素（audio / trade_card）：[`internal/protocol/v3/message_chain.go`](../internal/protocol/v3/message_chain.go)
- 电商字段（ItemPublish / DeliverySettings / TradeCard / ConversationHistory 等）：
  [`internal/protocol/v3/ecommerce.go`](../internal/protocol/v3/ecommerce.go)
- HTTP 接口：[`internal/handler/tool.go`](../internal/handler/tool.go)
- 桥 WS 入口：[`internal/handler/websocket.go`](../internal/handler/websocket.go)
- 接入框架 WS 入口（server 模式）：[`internal/adaptergateway/client.go`](../internal/adaptergateway/client.go)
- 接入框架 WS 出口（client 模式）：[`internal/adaptergateway/client_connector.go`](../internal/adaptergateway/client_connector.go)
- 网关装配（内置目录注册）：[`internal/adaptergateway/gateway.go`](../internal/adaptergateway/gateway.go)
- LangBot 适配器：[`esplplatfrom/espl.py`](../../../esplplatfrom/espl.py)
- 端到端测试（接入框架视角）：[`E-SP-Line2-e2e-tests/esp_tool_catalog_e2e.py`](../../../E-SP-Line2-e2e-tests/esp_tool_catalog_e2e.py)
- 原版参考（兼容对照）：`XianYuApis/goofish_apis.py`、`XianYuApis/goofish_live.py`、
  `XianYuApis/message/types.py`
