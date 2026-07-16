# V3 协议说明文档

## 概述

V3 协议是 E-SP-Line2 的核心通信协议，定义了消息信封、消息链、事件类型和签名验证等标准。该协议参考了 LangBot 的设计，并针对电商场景进行了扩展。

## 协议版本

- **当前版本**: v3
- **协议标识**: `protocol_version: "v3"`

## 核心概念

### 1. 消息信封 (MessageEnvelope)

消息信封是协议的基础结构，用于包装所有类型的消息和事件。

```go
type MessageEnvelope struct {
    ProtocolVersion string      `json:"protocol_version"` // "v3"
    EventID         string      `json:"event_id"`         // UUID
    TraceID         string      `json:"trace_id"`         // 追踪 ID
    Timestamp       int64       `json:"timestamp"`        // Unix 时间戳（毫秒）
    Platform        string      `json:"platform"`         // 平台标识
    AdapterID       string      `json:"adapter_id"`       // 适配器实例 ID
    EventType       EventType   `json:"event_type"`       // 事件类型
    Payload         interface{} `json:"payload"`          // 消息内容
    Signature       string      `json:"signature"`        // HMAC 签名
}
```

**字段说明**:
- `protocol_version`: 协议版本，固定为 "v3"
- `event_id`: 事件唯一标识，使用 UUID 生成
- `trace_id`: 追踪标识，用于全链路追踪
- `timestamp`: 事件发生时间，Unix 时间戳（毫秒）
- `platform`: 平台标识（如 "taobao", "xianyu"）
- `adapter_id`: 适配器实例标识
- `event_type`: 事件类型（见下文）
- `payload`: 事件负载，根据事件类型不同而不同
- `signature`: 消息签名，用于验证消息完整性

### 2. 消息链 (MessageChain)

消息链是消息内容的载体，支持多种消息元素的组合。

```go
type MessageChain struct {
    ID        string           `json:"id"`
    Timestamp int64            `json:"timestamp"`
    Platform  string           `json:"platform"`
    Instance  string           `json:"instance"`
    Sender    SenderInfo       `json:"sender"`
    Content   []MessageElement `json:"content"`
    Raw       json.RawMessage  `json:"raw,omitempty"`
    Hash      string           `json:"hash"`
}
```

**字段说明**:
- `id`: 消息链唯一标识
- `timestamp`: 消息时间戳
- `platform`: 平台标识
- `instance`: 实例标识
- `sender`: 发送者信息
- `content`: 消息元素列表
- `raw`: 原始平台消息（可选）
- `hash`: 消息链哈希（MD5）

### 3. 消息元素 (MessageElement)

消息元素是消息链的基本组成单元。

```go
type MessageElement struct {
    Type    ElementType `json:"type"`
    Content interface{} `json:"content"`
}
```

**支持的元素类型**:

| 类型 | 说明 | 内容结构 |
|------|------|----------|
| `text` | 文本消息 | `{"text": "消息内容"}` |
| `image` | 图片消息 | `{"url": "...", "width": 100, "height": 100}` |
| `audio` | 音频消息 | `{"url": "...", "duration": 60}` |
| `video` | 视频消息 | `{"url": "...", "width": 1920, "height": 1080}` |
| `file` | 文件消息 | `{"url": "...", "name": "file.pdf"}` |
| `product_card` | 商品卡片 | 见下文 |
| `order_info` | 订单信息 | 见下文 |
| `inquiry` | 咨询信息 | 见下文 |
| `location` | 位置信息 | `{"latitude": 39.9, "longitude": 116.4}` |
| `emoji` | 表情消息 | `{"emoji": "😀"}` |
| `mention` | @提及 | `{"user_id": "123", "username": "user"}` |

## 事件类型

### 消息事件

| 事件类型 | 说明 |
|----------|------|
| `message.received` | 收到消息 |
| `message.sent` | 发送消息 |
| `message.delivered` | 消息已送达 |
| `message.acked` | 消息已确认 |
| `message.failed` | 消息发送失败 |

### 适配器事件

| 事件类型 | 说明 |
|----------|------|
| `adapter.connected` | 适配器已连接 |
| `adapter.disconnected` | 适配器已断开 |
| `adapter.started` | 适配器已启动 |
| `adapter.stopped` | 适配器已停止 |
| `adapter.error` | 适配器错误 |

### 命令事件

| 事件类型 | 说明 |
|----------|------|
| `command.created` | 命令已创建 |
| `command.queued` | 命令已入队 |
| `command.sending` | 命令发送中 |
| `command.sent` | 命令已发送 |
| `command.failed` | 命令发送失败 |
| `command.retrying` | 命令重试中 |
| `command.expired` | 命令已过期 |

### 系统事件

| 事件类型 | 说明 |
|----------|------|
| `system.error` | 系统错误 |
| `system.health_check` | 健康检查 |

## 电商扩展

### 商品卡片 (ProductCard)

```go
type ProductCard struct {
    ItemID    string  `json:"item_id"`
    Title     string  `json:"title"`
    Price     float64 `json:"price"`
    ImageURL  string  `json:"image_url"`
    DetailURL string  `json:"detail_url"`
    Platform  string  `json:"platform,omitempty"`
    SKU       string  `json:"sku,omitempty"`
    Stock     int     `json:"stock,omitempty"`
}
```

**示例**:
```json
{
  "type": "product_card",
  "content": {
    "item_id": "123456",
    "title": "iPhone 15 Pro",
    "price": 7999.00,
    "image_url": "https://example.com/iphone.jpg",
    "detail_url": "https://example.com/item/123456",
    "platform": "taobao",
    "sku": "SKU-001",
    "stock": 100
  }
}
```

### 订单信息 (OrderInfo)

```go
type OrderInfo struct {
    OrderID     string  `json:"order_id"`
    Status      string  `json:"status"`
    Amount      float64 `json:"amount"`
    Currency    string  `json:"currency,omitempty"`
    ItemCount   int     `json:"item_count,omitempty"`
    CreatedAt   int64   `json:"created_at,omitempty"`
    PaidAt      int64   `json:"paid_at,omitempty"`
    ShippedAt   int64   `json:"shipped_at,omitempty"`
    DeliveredAt int64   `json:"delivered_at,omitempty"`
}
```

**订单状态**:
- `pending`: 待付款
- `paid`: 已付款
- `shipped`: 已发货
- `delivered`: 已送达
- `cancelled`: 已取消
- `refunded`: 已退款

**示例**:
```json
{
  "type": "order_info",
  "content": {
    "order_id": "ORDER-20240101-001",
    "status": "paid",
    "amount": 7999.00,
    "currency": "CNY",
    "item_count": 1,
    "created_at": 1704067200000,
    "paid_at": 1704067260000
  }
}
```

### 咨询信息 (Inquiry)

```go
type Inquiry struct {
    ProductID string `json:"product_id,omitempty"`
    OrderID   string `json:"order_id,omitempty"`
    Question  string `json:"question"`
    Category  string `json:"category,omitempty"`
}
```

**咨询类别**:
- `price`: 价格咨询
- `shipping`: 物流咨询
- `quality`: 质量咨询
- `return`: 退货咨询
- `other`: 其他咨询

**示例**:
```json
{
  "type": "inquiry",
  "content": {
    "product_id": "123456",
    "question": "这个商品有货吗？",
    "category": "shipping"
  }
}
```

## 签名验证

### 签名算法

支持以下签名算法：

| 算法 | 说明 |
|------|------|
| `HMAC-SHA256` | 推荐，安全性高 |
| `HMAC-MD5` | 兼容旧系统 |
| `MD5` | 简单场景 |

### 签名流程

1. 构造待签名数据（JSON 格式，不含 signature 字段）
2. 使用密钥计算 HMAC
3. 将签名结果转换为十六进制字符串
4. 将签名添加到消息信封

**示例代码**:
```go
signer := v3.NewSigner(v3.AlgorithmHMACSHA256, secret)
err := signer.SignEnvelope(envelope)
```

### 验证流程

1. 从消息信封中提取签名
2. 构造待验证数据（不含 signature 字段）
3. 使用相同密钥计算 HMAC
4. 比较计算结果与提取的签名

**示例代码**:
```go
signer := v3.NewSigner(v3.AlgorithmHMACSHA256, secret)
valid := signer.VerifyEnvelope(envelope)
```

## 消息 ID 生成

消息 ID 使用原始消息字段的 MD5 或 Hash 生成：

```go
func GenerateMessageID(rawMessage json.RawMessage) string {
    hash := md5.Sum(rawMessage)
    return hex.EncodeToString(hash[:])
}
```

**生成规则**:
1. 提取原始消息的关键字段
2. 按固定顺序排列
3. 计算 MD5 或 SHA256
4. 转换为十六进制字符串

## 幂等性

### 幂等键 (IdempotencyKey)

用于防止消息重复处理：

```go
func GenerateIdempotencyKey(platformID, instanceID, messageID string) string {
    return platformID + "-" + instanceID + "-" + messageID
}
```

**使用场景**:
- 消息去重
- 命令重试
- 状态同步

## 追踪标识

### TraceID

用于全链路追踪：

```go
func GenerateTraceID() string {
    bytes := make([]byte, 8)
    rand.Read(bytes)
    return "trace-" + hex.EncodeToString(bytes)
}
```

**使用场景**:
- 请求追踪
- 日志关联
- 性能分析

## 协议示例

### 接收消息示例

```json
{
  "protocol_version": "v3",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "trace_id": "trace-12345678",
  "timestamp": 1704067200000,
  "platform": "taobao",
  "adapter_id": "instance-001",
  "event_type": "message.received",
  "payload": {
    "id": "msg-001",
    "timestamp": 1704067200000,
    "platform": "taobao",
    "instance": "instance-001",
    "sender": {
      "id": "user-123",
      "name": "买家A"
    },
    "content": [
      {
        "type": "text",
        "content": {
          "text": "你好，这个商品还有货吗？"
        }
      }
    ],
    "hash": "a1b2c3d4e5f6..."
  },
  "signature": "hmac-sha256-signature-here"
}
```

### 发送命令示例

```json
{
  "protocol_version": "v3",
  "event_id": "660e8400-e29b-41d4-a716-446655440001",
  "trace_id": "trace-87654321",
  "timestamp": 1704067260000,
  "platform": "taobao",
  "adapter_id": "instance-001",
  "event_type": "command.created",
  "payload": {
    "command_type": "send_text",
    "instance_id": "instance-001",
    "target_id": "user-123",
    "payload": {
      "text": "您好，商品有货的，欢迎下单！"
    },
    "trace_id": "trace-87654321",
    "timestamp": 1704067260000
  },
  "signature": "hmac-sha256-signature-here"
}
```

## 错误处理

### 协议错误码

| 错误码 | 说明 |
|--------|------|
| `INVALID_ENVELOPE` | 无效的消息信封 |
| `INVALID_SIGNATURE` | 签名验证失败 |
| `UNSUPPORTED_TYPE` | 不支持的消息类型 |
| `EXPIRED_REQUEST` | 请求已过期 |

### 错误响应格式

```json
{
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "Signature verification failed",
    "trace_id": "trace-12345678"
  }
}
```

## 最佳实践

### 1. 消息设计

- 使用消息链组合多种消息元素
- 保留原始平台消息用于调试
- 为电商场景使用专用消息类型

### 2. 签名安全

- 使用 HMAC-SHA256 算法
- 定期轮换密钥
- 验证所有入站消息签名

### 3. 幂等处理

- 为所有消息生成幂等键
- 在接收端进行去重检查
- 记录已处理的消息 ID

### 4. 追踪调试

- 为每个请求生成 TraceID
- 在日志中记录 TraceID
- 使用 TraceID 关联相关日志

## 版本演进

### v3.0 (当前)

- 初始版本
- 支持基础消息类型
- 支持电商扩展类型
- 支持签名验证

### 未来计划

- v3.1: 增加更多电商消息类型
- v3.2: 支持消息压缩
- v3.3: 支持消息加密

## 参考资源

- [LangBot 协议文档](https://docs.langbot.app/protocol)
- [E-SP-Line2 架构文档](../E-SP-Line2-about-platfrom.md)
- [API 参考文档](./api-reference.md)
