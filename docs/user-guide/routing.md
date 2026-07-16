# 路由规则

## 概述

路由规则决定消息如何从适配器实例分发到下游应用。通过灵活的路由配置，可以实现复杂的消息分发逻辑。

## 路由规则结构

```json
{
  "id": "rule-001",
  "name": "淘宝消息路由",
  "platform_id": "taobao",
  "instance_id": "",
  "priority": 10,
  "conditions": {
    "event_type": "message.received"
  },
  "target_type": "app",
  "target_id": "app-001",
  "enabled": true
}
```

## 路由规则字段

### 基础字段

- **id**：规则唯一标识
- **name**：规则名称
- **enabled**：是否启用

### 匹配字段

- **platform_id**：平台 ID（可选，为空匹配所有平台）
- **instance_id**：实例 ID（可选，为空匹配所有实例）
- **priority**：优先级（数字越大优先级越高）

### 条件字段

- **conditions**：匹配条件（JSON 对象）
  - `event_type`：事件类型
  - `adapter_id`：适配器 ID
  - 其他自定义条件

### 目标字段

- **target_type**：目标类型
  - `app`：应用程序
  - `webhook`：Webhook
  - `queue`：消息队列
- **target_id**：目标 ID

## 创建路由规则

### 步骤 1：进入路由管理

在管理界面点击"路由规则"菜单。

### 步骤 2：创建新规则

点击"创建规则"按钮，填写规则信息。

### 步骤 3：配置匹配条件

设置平台、实例和条件匹配规则。

### 步骤 4：配置目标

设置消息分发目标。

### 步骤 5：保存并启用

保存规则并启用。

## 路由示例

### 示例 1：按平台路由

将所有淘宝消息路由到应用 A：

```json
{
  "name": "淘宝消息路由",
  "platform_id": "taobao",
  "priority": 10,
  "target_type": "app",
  "target_id": "app-taobao"
}
```

### 示例 2：按实例路由

将特定实例的消息路由到应用 B：

```json
{
  "name": "店铺A消息路由",
  "instance_id": "instance-001",
  "priority": 20,
  "target_type": "app",
  "target_id": "app-shop-a"
}
```

### 示例 3：按事件类型路由

只路由接收到的消息：

```json
{
  "name": "接收消息路由",
  "conditions": {
    "event_type": "message.received"
  },
  "target_type": "app",
  "target_id": "app-message-handler"
}
```

### 示例 4：多条件路由

组合多个条件：

```json
{
  "name": "淘宝文本消息路由",
  "platform_id": "taobao",
  "conditions": {
    "event_type": "message.received",
    "message_type": "text"
  },
  "target_type": "app",
  "target_id": "app-text-handler"
}
```

## 优先级

当多个规则匹配同一条消息时，按优先级排序：

1. 优先级高的规则先匹配
2. 如果优先级相同，按创建时间排序
3. 默认优先级为 0

### 优先级建议

- **高优先级（100+）**：特殊处理规则
- **中优先级（50-99）**：业务规则
- **低优先级（0-49）**：默认规则

## 路由目标类型

### App（应用程序）

消息发送到注册的应用程序，通过 WebSocket 或 HTTP 接收。

**适用场景**：
- 实时消息处理
- 需要双向通信
- 复杂业务逻辑

### Webhook

消息通过 HTTP POST 发送到指定的 URL。

**适用场景**：
- 简单消息转发
- 与现有系统集成
- 不需要实时响应

**配置示例**：
```json
{
  "target_type": "webhook",
  "target_id": "https://api.example.com/webhook"
}
```

### Queue（消息队列）

消息发送到消息队列，供消费者处理。

**适用场景**：
- 异步处理
- 消息持久化
- 削峰填谷

## 路由测试

### 测试工具

系统提供路由测试工具，可以模拟消息并查看路由结果。

### 测试步骤

1. 进入"路由测试"页面
2. 选择平台和实例
3. 构造测试消息
4. 点击"测试"按钮
5. 查看匹配的规则和目标

## 路由监控

### 统计信息

在"路由监控"页面可以查看：
- 规则匹配次数
- 消息分发成功率
- 平均处理时间
- 错误统计

### 实时日志

查看实时的路由日志，包括：
- 消息接收
- 规则匹配
- 消息分发
- 处理结果

## 最佳实践

### 规则设计

1. **单一职责**：每个规则只负责一种场景
2. **明确命名**：规则名称要清晰表达用途
3. **合理优先级**：避免优先级冲突
4. **定期审查**：定期检查和优化规则

### 性能优化

1. **减少规则数量**：合并相似的规则
2. **优化条件**：使用简单的条件匹配
3. **缓存结果**：对频繁匹配的规则启用缓存

### 故障处理

1. **监控告警**：配置路由失败告警
2. **降级策略**：配置默认路由规则
3. **日志分析**：定期分析路由日志

## 高级功能

### 动态路由

根据消息内容动态选择目标：

```json
{
  "name": "动态路由",
  "conditions": {
    "dynamic_target": true
  },
  "target_type": "dynamic"
}
```

### 路由链

多个规则串联处理：

```json
{
  "name": "路由链",
  "chain": [
    {"rule_id": "rule-001"},
    {"rule_id": "rule-002"}
  ]
}
```

### 条件表达式

使用复杂的条件表达式：

```json
{
  "conditions": {
    "$or": [
      {"event_type": "message.received"},
      {"event_type": "message.sent"}
    ],
    "$and": [
      {"platform_id": "taobao"},
      {"priority": {"$gt": 5}}
    ]
  }
}
```

## 故障排除

### 规则不生效

**检查**：
1. 规则是否启用
2. 匹配条件是否正确
3. 优先级是否被其他规则覆盖

### 消息未分发

**检查**：
1. 目标是否在线
2. 目标配置是否正确
3. 网络连接是否正常

### 重复分发

**检查**：
1. 是否有多个规则匹配
2. 规则优先级设置
3. 是否需要去重逻辑
