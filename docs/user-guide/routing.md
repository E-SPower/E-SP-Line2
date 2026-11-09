# Routing rules

## Overview

Routing rules determine how messages are dispatched from adapter instances to downstream applications. Flexible routing configuration makes complex message dispatch logic possible.

## Routing rule structure

```json
{
  "id": "rule-001",
  "name": "Taobao message routing",
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

## Routing rule fields

### Basic fields

- **id**: unique rule identifier
- **name**: rule name
- **enabled**: whether the rule is enabled

### Matching fields

- **platform_id**: platform ID (optional; empty matches all platforms)
- **instance_id**: instance ID (optional; empty matches all instances)
- **priority**: priority (the higher the number, the higher the priority)

### Condition fields

- **conditions**: match conditions (JSON object)
  - `event_type`: event type
  - `adapter_id`: adapter ID
  - Other custom conditions

### Target fields

- **target_type**: target type
  - `app`: application
  - `webhook`: Webhook
  - `queue`: message queue
- **target_id**: target ID

## Create a routing rule

### Step 1: Go to route management

Open the Routes page.

### Step 2: Create a new rule

Click the "Create rule" button and fill in the rule information.

### Step 3: Configure match conditions

Set the platform, instance, and condition matching rules.

### Step 4: Configure the target

Set the message dispatch target.

### Step 5: Save and enable

Save the rule and enable it.

## Routing examples

### Example 1: Route by platform

Route all Taobao messages to application A:

```json
{
  "name": "Taobao message routing",
  "platform_id": "taobao",
  "priority": 10,
  "target_type": "app",
  "target_id": "app-taobao"
}
```

### Example 2: Route by instance

Route messages from a specific instance to application B:

```json
{
  "name": "Shop A message routing",
  "instance_id": "instance-001",
  "priority": 20,
  "target_type": "app",
  "target_id": "app-shop-a"
}
```

### Example 3: Route by event type

Route only received messages:

```json
{
  "name": "Received message routing",
  "conditions": {
    "event_type": "message.received"
  },
  "target_type": "app",
  "target_id": "app-message-handler"
}
```

### Example 4: Route by multiple conditions

Combine several conditions:

```json
{
  "name": "Taobao text message routing",
  "platform_id": "taobao",
  "conditions": {
    "event_type": "message.received",
    "message_type": "text"
  },
  "target_type": "app",
  "target_id": "app-text-handler"
}
```

## Priority

When several rules match the same message, they are ordered by priority:

1. Rules with a higher priority match first
2. If the priorities are equal, they are ordered by creation time
3. The default priority is 0

### Priority guidelines

- **High priority (100+)**: special-case rules
- **Medium priority (50-99)**: business rules
- **Low priority (0-49)**: default rules

## Routing target types

### App (application)

Messages are sent to a registered application, which receives them over WebSocket or HTTP.

**Use cases**:
- Real-time message processing
- Two-way communication required
- Complex business logic

### Webhook

Messages are sent to the specified URL over HTTP POST.

**Use cases**:
- Simple message forwarding
- Integration with existing systems
- No real-time response required

**Configuration example**:
```json
{
  "target_type": "webhook",
  "target_id": "https://api.example.com/webhook"
}
```

### Queue (message queue)

Messages are sent to a message queue for consumers to process.

**Use cases**:
- Asynchronous processing
- Message persistence
- Peak shaving

## Route testing

### Testing tool

The system provides a route testing tool that simulates messages and shows the routing result.

### Test steps

1. Go to the route testing page
2. Select a platform and instance
3. Build a test message
4. Click the "Test" button
5. View the matched rules and targets

## Routing monitoring

### Statistics

On the Monitoring page you can view:
- Rule match count
- Message dispatch success rate
- Average processing time
- Error statistics

### Real-time logs

View the real-time routing logs, including:
- Message receipt
- Rule matching
- Message dispatch
- Processing result

## Best practices

### Rule design

1. **Single responsibility**: each rule covers one scenario only
2. **Clear naming**: rule names should state their purpose clearly
3. **Sensible priorities**: avoid priority conflicts
4. **Regular review**: review and tune the rules regularly

### Performance tuning

1. **Fewer rules**: merge similar rules
2. **Simpler conditions**: use simple condition matching
3. **Cached results**: enable caching for frequently matched rules

### Failure handling

1. **Monitoring and alerts**: configure routing failure alerts
2. **Fallback strategy**: configure default routing rules
3. **Log analysis**: analyze the routing logs regularly

## Advanced features

### Dynamic routing

Select the target dynamically based on message content:

```json
{
  "name": "Dynamic routing",
  "conditions": {
    "dynamic_target": true
  },
  "target_type": "dynamic"
}
```

### Routing chain

Process messages through several rules in sequence:

```json
{
  "name": "Routing chain",
  "chain": [
    {"rule_id": "rule-001"},
    {"rule_id": "rule-002"}
  ]
}
```

### Condition expressions

Use complex condition expressions:

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

## Troubleshooting

### A rule does not take effect

**Check**:
1. Whether the rule is enabled
2. Whether the match conditions are correct
3. Whether another rule overrides its priority

### Messages are not dispatched

**Check**:
1. Whether the target is online
2. Whether the target configuration is correct
3. Whether the network connection works

### Duplicate dispatch

**Check**:
1. Whether several rules match
2. The rule priority settings
3. Whether deduplication logic is needed
