# Troubleshooting

## Common problems

### 1. The adapter will not start

#### Symptoms
- The adapter status stays at "Starting"
- The logs show a connection failure

#### Possible causes
1. The Cookie is invalid or malformed
2. Network connectivity problems
3. The platform service is unavailable
4. Wrong configuration parameters

#### Resolution steps

**Check the Cookie**
```bash
# Validate the Cookie format
curl -H "Cookie: your_cookie_here" https://www.taobao.com
```

**Check the network**
```bash
# Test network connectivity
ping www.taobao.com
ping www.goofish.com
```

**Check the logs**
```bash
# View the backend logs
tail -f logs/app.log
```

### 2. Messages are not received

#### Symptoms
- The adapter status is "Running"
- But no messages arrive

#### Possible causes
1. No routing rules are configured
2. The routing rules are not enabled
3. The downstream application is not connected
4. Platform message delay

#### Resolution steps

**Check the routing rules**
1. Go to the Routes page
2. Confirm that the rules are enabled
3. Check whether the match conditions are correct

**Check the downstream application**
1. Confirm that the application is connected
2. Check the application WebSocket status
3. Check the application logs

**Test the message flow**
1. Use the route testing tool
2. Send a test message
3. Check message tracking

### 3. Message sending fails

#### Symptoms
- Sending a message returns an error
- The message status is "Failed"

#### Possible causes
1. The target user does not exist
2. The message format is wrong
3. Platform limits (rate, content, etc.)
4. Insufficient Cookie permissions

#### Resolution steps

**Check the target**
- Confirm that the target user ID is correct
- Confirm that the conversation has been created

**Check the message format**
```json
{
  "type": "text",
  "content": "Message content"
}
```

**Check the error details**
1. Go to the Messages page
2. Find the failed message
3. View the detailed error information

### 4. System performance problems

#### Symptoms
- Message processing delay
- Slow system response
- High memory usage

#### Possible causes
1. Message queue backlog
2. Database performance problems
3. Too many connections
4. Memory leak

#### Resolution steps

**Check the queue status**
```bash
# View the queue length
redis-cli llen message:queue
```

**Optimize the database**
```sql
-- Check slow queries
SHOW PROCESSLIST;

-- Optimize indexes
ANALYZE TABLE inbound_events;
```

**Adjust the configuration**
```yaml
# config.yaml
server:
  max_connections: 1000
  worker_pool_size: 10

queue:
  max_size: 10000
  consumer_count: 5
```

### 5. WebSocket connection drops

#### Symptoms
- The application disconnects frequently
- Messages are lost after reconnecting

#### Possible causes
1. Unstable network
2. Heartbeat timeout
3. Server restart
4. Load balancing problems

#### Resolution steps

**Check the heartbeat configuration**
```json
{
  "heartbeat_interval": 30,
  "heartbeat_timeout": 60
}
```

**Implement reconnect logic**
```javascript
// Frontend reconnect example. The handshake needs the JWT (the browser
// WebSocket API cannot set an Authorization header, so it goes in the query)
// and an allow-listed Origin.
function connect() {
  const token = localStorage.getItem('auth_token');
  const ws = new WebSocket(`ws://localhost:8080/ws/app?app_id=app-1&token=${token}`);
  
  ws.onclose = () => {
    setTimeout(connect, 5000);
  };
}
```

**Check the server logs**
```bash
# View the WebSocket logs
grep "WebSocket" logs/app.log
```

## Log analysis
### 适配器连不上平台（需要代理）

**症状**：实例启动后连不上平台，日志里是超时、`ECONNRESET`、证书或 DNS 错误，
但同一个地址用浏览器能打开。

**原因**：只代理了浏览器，接入器进程本身没有代理可走。

**解决**：在该实例的表单里填「代理（可选）」，或直接改配置：

```json
{ "proxy": "http://127.0.0.1:7890" }
```

启动横幅里的 `proxy` 一行会显示实际生效值（`inherit` / `direct (forced)` / 打码后的
URL）。填了 `none` 是强制直连，注意别和代理软件同时开着还以为在走代理。
详见 [适配器配置](./adapters.md)。

## 日志分析

### Log levels

- **DEBUG**: debug information, detailed technical details
- **INFO**: general information, normal operation flow
- **WARN**: warnings, possible problems
- **ERROR**: error information, problems that need handling
- **FATAL**: fatal errors, the system cannot keep running

### Common log queries

**Find errors**
```bash
grep "ERROR" logs/app.log | tail -50
```

**Find a specific adapter**
```bash
grep "adapter_id: instance-001" logs/app.log
```

**Find a specific time range**
```bash
grep "2024-01-01 10:" logs/app.log
```

### Log format

```
[time] [level] [module] message content key1=value1 key2=value2
```

Example:
```
[2024-01-01 10:30:45] [INFO] [adapter] Adapter started adapter_id=instance-001 platform=taobao
```

## Debugging tools

### 1. Health check endpoint

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 3600
}
```

### 2. System status endpoint

```bash
curl http://localhost:8080/api/v1/system/status
```

Response:
```json
{
  "adapters": {
    "total": 10,
    "running": 8,
    "stopped": 2
  },
  "messages": {
    "received": 1000,
    "sent": 950,
    "failed": 50
  },
  "connections": {
    "websocket": 15,
    "http": 5
  }
}
```

### 3. Message tracking

Use message tracking on the Messages page:
1. Enter a message ID or trace ID
2. View the full message lifecycle
3. Analyze the processing time and status

### 4. Route testing

Use route testing in the management interface:
1. Build a test message
2. Select a platform and instance
3. View the matched rules
4. Verify the routing result

## Performance tuning

### Database tuning

**Index tuning**
```sql
-- Add indexes for common queries
CREATE INDEX idx_inbound_events_platform ON inbound_events(platform_id);
CREATE INDEX idx_inbound_events_status ON inbound_events(status);
CREATE INDEX idx_inbound_events_created ON inbound_events(created_at);
```

**Query tuning**
```sql
-- Use paged queries
SELECT * FROM inbound_events 
WHERE status = 'received' 
ORDER BY created_at DESC 
LIMIT 100 OFFSET 0;
```

### Caching

**Redis configuration**
```yaml
cache:
  enabled: true
  driver: redis
  host: localhost
  port: 6379
  ttl: 3600
```

**Caching strategy**
- Platform configuration: long-term cache
- Routing rules: medium-term cache
- Conversation state: short-term cache

### Connection pool tuning

**Database connection pool**
```yaml
database:
  pool_size: 20
  max_overflow: 10
  pool_timeout: 30
```

**HTTP connection pool**
```yaml
http_client:
  max_connections: 100
  max_connections_per_host: 10
  keep_alive: true
```

## Monitoring and alerts

### Key metrics

1. **Adapter status**
   - Number of running adapters
   - Number of adapters in error
   - Adapter start success rate

2. **Message metrics**
   - Message receive rate
   - Message send success rate
   - Message processing delay

3. **System metrics**
   - CPU usage
   - Memory usage
   - Disk I/O
   - Network traffic

### Alert configuration

```yaml
alerts:
  - name: adapter_error
    condition: adapter.status == 'error'
    severity: critical
    notification: email,slack
    
  - name: message_failure_rate
    condition: message.failure_rate > 0.1
    severity: warning
    notification: email
    
  - name: high_latency
    condition: message.processing_time > 5000
    severity: warning
    notification: slack
```

## Contact support

If the methods above do not solve the problem, provide the following information:

1. **System information**
   - System version
   - Operating system
   - Database type and version

2. **Error information**
   - The complete error logs
   - When the error occurred
   - Steps to reproduce the error

3. **Configuration information**
   - The relevant configuration files (redacted)
   - Adapter configuration
   - Routing rule configuration

4. **Environment information**
   - Network environment
   - Deployment method
   - Status of the relevant services
