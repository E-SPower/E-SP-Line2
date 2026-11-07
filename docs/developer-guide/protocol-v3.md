# V3 protocol specification

## Overview

The V3 protocol is the core communication protocol of E-SP-Line2 and defines standards such as message envelopes, message chains, event types, and signature verification. The protocol draws on the design of LangBot and extends it for e-commerce scenarios.

## Protocol version

- **Current version**: v3
- **Protocol identifier**: `protocol_version: "v3"`

## Core concepts

### 1. Message envelope (MessageEnvelope)

The message envelope is the base structure of the protocol and wraps all types of messages and events.

```go
type MessageEnvelope struct {
    ProtocolVersion string      `json:"protocol_version"` // "v3"
    EventID         string      `json:"event_id"`         // UUID
    TraceID         string      `json:"trace_id"`         // Trace ID
    Timestamp       int64       `json:"timestamp"`        // Unix timestamp (milliseconds)
    Platform        string      `json:"platform"`         // Platform identifier
    AdapterID       string      `json:"adapter_id"`       // Adapter instance ID
    EventType       EventType   `json:"event_type"`       // Event type
    Payload         interface{} `json:"payload"`          // Message content
    Signature       string      `json:"signature"`        // HMAC signature
}
```

**Field descriptions**:
- `protocol_version`: Protocol version, always "v3"
- `event_id`: Unique event identifier, generated as a UUID
- `trace_id`: Trace ID used for end-to-end tracing
- `timestamp`: Time the event occurred, as a Unix timestamp (milliseconds)
- `platform`: Platform identifier (such as "taobao", "xianyu")
- `adapter_id`: Adapter instance identifier
- `event_type`: Event type (see below)
- `payload`: Event payload, which varies by event type
- `signature`: Message signature used to verify message integrity

### 2. Message chain (MessageChain)

The message chain carries message content and supports combinations of multiple message elements.

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

**Field descriptions**:
- `id`: Unique message chain identifier
- `timestamp`: Message timestamp
- `platform`: Platform identifier
- `instance`: Instance identifier
- `sender`: Sender info
- `content`: List of message elements
- `raw`: Raw platform message (optional)
- `hash`: Message chain hash (MD5)

### 3. Message element (MessageElement)

A message element is the basic unit of a message chain.

```go
type MessageElement struct {
    Type    ElementType `json:"type"`
    Content interface{} `json:"content"`
}
```

**Supported element types**:

| Type | Description | Content structure |
|------|------|----------|
| `text` | Text message | `{"text": "Message content"}` |
| `image` | Image message | `{"url": "...", "width": 100, "height": 100}` |
| `audio` | Audio message | `{"url": "...", "duration": 60}` |
| `video` | Video message | `{"url": "...", "width": 1920, "height": 1080}` |
| `file` | File message | `{"url": "...", "name": "file.pdf"}` |
| `product_card` | Product card | See below |
| `order_info` | Order info | See below |
| `inquiry` | Inquiry | See below |
| `location` | Location info | `{"latitude": 39.9, "longitude": 116.4}` |
| `emoji` | Emoji message | `{"emoji": "😀"}` |
| `mention` | @mention | `{"user_id": "123", "username": "user"}` |

## Event types

### Message events

| Event type | Description |
|----------|------|
| `message.received` | Message received |
| `message.sent` | Message sent |
| `message.delivered` | Message delivered |
| `message.acked` | Message acknowledged |
| `message.failed` | Message sending failed |

### Adapter events

| Event type | Description |
|----------|------|
| `adapter.connected` | Adapter connected |
| `adapter.disconnected` | Adapter disconnected |
| `adapter.started` | Adapter started |
| `adapter.stopped` | Adapter stopped |
| `adapter.error` | Adapter error |

### Command events

| Event type | Description |
|----------|------|
| `command.created` | Command created |
| `command.queued` | Command queued |
| `command.sending` | Command being sent |
| `command.sent` | Command sent |
| `command.failed` | Command sending failed |
| `command.retrying` | Command retrying |
| `command.expired` | Command expired |

### System events

| Event type | Description |
|----------|------|
| `system.error` | System error |
| `system.health_check` | Health check |

## E-commerce extensions

### Product card (ProductCard)

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

**Example**:
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

### Order info (OrderInfo)

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

**Order statuses**:
- `pending`: Awaiting payment
- `paid`: Paid
- `shipped`: Shipped
- `delivered`: Delivered
- `cancelled`: Cancelled
- `refunded`: Refunded

**Example**:
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

### Inquiry

```go
type Inquiry struct {
    ProductID string `json:"product_id,omitempty"`
    OrderID   string `json:"order_id,omitempty"`
    Question  string `json:"question"`
    Category  string `json:"category,omitempty"`
}
```

**Inquiry categories**:
- `price`: Price inquiry
- `shipping`: Shipping inquiry
- `quality`: Quality inquiry
- `return`: Return inquiry
- `other`: Other inquiry

**Example**:
```json
{
  "type": "inquiry",
  "content": {
    "product_id": "123456",
    "question": "Is this item in stock?",
    "category": "shipping"
  }
}
```

## Signature verification

### Signature algorithms

The following signature algorithms are supported:

| Algorithm | Description |
|------|------|
| `HMAC-SHA256` | Recommended, high security |
| `HMAC-MD5` | Compatible with legacy systems |
| `MD5` | Simple scenarios |

### Signing flow

1. Build the data to be signed (JSON format, without the signature field)
2. Compute the HMAC with the secret key
3. Convert the signature result to a hexadecimal string
4. Add the signature to the message envelope

**Example code**:
```go
signer := v3.NewSigner(v3.AlgorithmHMACSHA256, secret)
err := signer.SignEnvelope(envelope)
```

### Verification flow

1. Extract the signature from the message envelope
2. Build the data to be verified (without the signature field)
3. Compute the HMAC with the same secret key
4. Compare the computed result with the extracted signature

**Example code**:
```go
signer := v3.NewSigner(v3.AlgorithmHMACSHA256, secret)
valid := signer.VerifyEnvelope(envelope)
```

## Message ID generation

Message IDs are generated from the MD5 or hash of the raw message fields:

```go
func GenerateMessageID(rawMessage json.RawMessage) string {
    hash := md5.Sum(rawMessage)
    return hex.EncodeToString(hash[:])
}
```

**Generation rules**:
1. Extract the key fields of the raw message
2. Arrange them in a fixed order
3. Compute the MD5 or SHA256
4. Convert to a hexadecimal string

## Idempotency

### Idempotency key (IdempotencyKey)

Used to prevent duplicate message processing:

```go
func GenerateIdempotencyKey(platformID, instanceID, messageID string) string {
    return platformID + "-" + instanceID + "-" + messageID
}
```

**Use cases**:
- Message deduplication
- Command retries
- State synchronization

## Trace ID

### TraceID

Used for end-to-end tracing:

```go
func GenerateTraceID() string {
    bytes := make([]byte, 8)
    rand.Read(bytes)
    return "trace-" + hex.EncodeToString(bytes)
}
```

**Use cases**:
- Request tracing
- Log correlation
- Performance analysis

## Protocol examples

### Receiving a message example

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
      "name": "Buyer A"
    },
    "content": [
      {
        "type": "text",
        "content": {
          "text": "Hello, is this item still in stock?"
        }
      }
    ],
    "hash": "a1b2c3d4e5f6..."
  },
  "signature": "hmac-sha256-signature-here"
}
```

### Sending a command example

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
      "text": "Hello, the item is in stock. Feel free to place an order!"
    },
    "trace_id": "trace-87654321",
    "timestamp": 1704067260000
  },
  "signature": "hmac-sha256-signature-here"
}
```

## Error handling

### Protocol error codes

| Error code | Description |
|--------|------|
| `INVALID_ENVELOPE` | Invalid message envelope |
| `INVALID_SIGNATURE` | Signature verification failed |
| `UNSUPPORTED_TYPE` | Unsupported message type |
| `EXPIRED_REQUEST` | Request expired |

### Error response format

```json
{
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "Signature verification failed",
    "trace_id": "trace-12345678"
  }
}
```

## Best practices

### 1. Message design

- Use a message chain to combine multiple message elements
- Keep the raw platform message for debugging
- Use dedicated message types for e-commerce scenarios

### 2. Signature security

- Use the HMAC-SHA256 algorithm
- Rotate keys regularly
- Verify the signature of every inbound message

### 3. Idempotency handling

- Generate an idempotency key for every message
- Run deduplication checks on the receiving side
- Record processed message IDs

### 4. Trace debugging

- Generate a TraceID for every request
- Record the TraceID in logs
- Use the TraceID to correlate related logs

## Version evolution

### v3.0 (current)

- Initial version
- Supports base message types
- Supports e-commerce extension types
- Supports signature verification

### Future plans

- v3.1: Add more e-commerce message types
- v3.2: Support message compression
- v3.3: Support message encryption

## References

- [LangBot protocol documentation](https://docs.langbot.app/protocol)
- [E-SP-Line2 architecture overview](../../README.md#architecture)
- [API reference](./api-reference.md)
