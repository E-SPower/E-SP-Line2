# 故障排除

## 常见问题

### 1. 适配器无法启动

#### 症状
- 适配器状态停留在 "Starting"
- 日志显示连接失败

#### 可能原因
1. Cookie 失效或格式错误
2. 网络连接问题
3. 平台服务不可用
4. 配置参数错误

#### 解决步骤

**检查 Cookie**
```bash
# 验证 Cookie 格式
curl -H "Cookie: your_cookie_here" https://www.taobao.com
```

**检查网络**
```bash
# 测试网络连接
ping www.taobao.com
ping www.goofish.com
```

**查看日志**
```bash
# 查看后端日志
tail -f backend/logs/app.log
```

### 2. 消息接收不到

#### 症状
- 适配器状态为 "Running"
- 但没有收到任何消息

#### 可能原因
1. 路由规则未配置
2. 路由规则未启用
3. 下游应用未连接
4. 平台消息延迟

#### 解决步骤

**检查路由规则**
1. 进入"路由规则"页面
2. 确认规则已启用
3. 检查匹配条件是否正确

**检查下游应用**
1. 确认应用已连接
2. 检查应用 WebSocket 状态
3. 查看应用日志

**测试消息流**
1. 使用"路由测试"工具
2. 发送测试消息
3. 查看消息追踪

### 3. 消息发送失败

#### 症状
- 发送消息时返回错误
- 消息状态为 "Failed"

#### 可能原因
1. 目标用户不存在
2. 消息格式错误
3. 平台限制（频率、内容等）
4. Cookie 权限不足

#### 解决步骤

**检查目标**
- 确认目标用户 ID 正确
- 确认会话已创建

**检查消息格式**
```json
{
  "type": "text",
  "content": "消息内容"
}
```

**查看错误详情**
1. 进入"消息追踪"页面
2. 找到失败的消息
3. 查看详细错误信息

### 4. 系统性能问题

#### 症状
- 消息处理延迟
- 系统响应缓慢
- 内存使用过高

#### 可能原因
1. 消息队列积压
2. 数据库性能问题
3. 连接数过多
4. 内存泄漏

#### 解决步骤

**检查队列状态**
```bash
# 查看队列长度
redis-cli llen message:queue
```

**优化数据库**
```sql
-- 检查慢查询
SHOW PROCESSLIST;

-- 优化索引
ANALYZE TABLE inbound_events;
```

**调整配置**
```yaml
# config.yaml
server:
  max_connections: 1000
  worker_pool_size: 10

queue:
  max_size: 10000
  consumer_count: 5
```

### 5. WebSocket 连接断开

#### 症状
- 应用频繁断开连接
- 重连后丢失消息

#### 可能原因
1. 网络不稳定
2. 心跳超时
3. 服务器重启
4. 负载均衡问题

#### 解决步骤

**检查心跳配置**
```json
{
  "heartbeat_interval": 30,
  "heartbeat_timeout": 60
}
```

**实现重连逻辑**
```javascript
// 前端重连示例
function connect() {
  const ws = new WebSocket('ws://localhost:8080/ws/app');
  
  ws.onclose = () => {
    setTimeout(connect, 5000);
  };
}
```

**检查服务器日志**
```bash
# 查看 WebSocket 日志
grep "WebSocket" backend/logs/app.log
```

## 日志分析

### 日志级别

- **DEBUG**：调试信息，详细的技术细节
- **INFO**：一般信息，正常操作流程
- **WARN**：警告信息，可能的问题
- **ERROR**：错误信息，需要处理的问题
- **FATAL**：致命错误，系统无法继续运行

### 常用日志查询

**查找错误**
```bash
grep "ERROR" backend/logs/app.log | tail -50
```

**查找特定适配器**
```bash
grep "adapter_id: instance-001" backend/logs/app.log
```

**查找特定时间段**
```bash
grep "2024-01-01 10:" backend/logs/app.log
```

### 日志格式

```
[时间] [级别] [模块] 消息内容 key1=value1 key2=value2
```

示例：
```
[2024-01-01 10:30:45] [INFO] [adapter] Adapter started adapter_id=instance-001 platform=taobao
```

## 调试工具

### 1. 健康检查端点

```bash
curl http://localhost:8080/health
```

响应：
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 3600
}
```

### 2. 系统状态端点

```bash
curl http://localhost:8080/api/v1/system/status
```

响应：
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

### 3. 消息追踪

在管理界面使用"消息追踪"功能：
1. 输入消息 ID 或 trace ID
2. 查看消息完整生命周期
3. 分析处理时间和状态

### 4. 路由测试

在管理界面使用"路由测试"功能：
1. 构造测试消息
2. 选择平台和实例
3. 查看匹配的规则
4. 验证路由结果

## 性能优化

### 数据库优化

**索引优化**
```sql
-- 添加常用查询索引
CREATE INDEX idx_inbound_events_platform ON inbound_events(platform_id);
CREATE INDEX idx_inbound_events_status ON inbound_events(status);
CREATE INDEX idx_inbound_events_created ON inbound_events(created_at);
```

**查询优化**
```sql
-- 使用分页查询
SELECT * FROM inbound_events 
WHERE status = 'received' 
ORDER BY created_at DESC 
LIMIT 100 OFFSET 0;
```

### 缓存优化

**Redis 配置**
```yaml
cache:
  enabled: true
  driver: redis
  host: localhost
  port: 6379
  ttl: 3600
```

**缓存策略**
- 平台配置：长期缓存
- 路由规则：中期缓存
- 会话状态：短期缓存

### 连接池优化

**数据库连接池**
```yaml
database:
  pool_size: 20
  max_overflow: 10
  pool_timeout: 30
```

**HTTP 连接池**
```yaml
http_client:
  max_connections: 100
  max_connections_per_host: 10
  keep_alive: true
```

## 监控告警

### 关键指标

1. **适配器状态**
   - 运行中适配器数量
   - 错误适配器数量
   - 适配器启动成功率

2. **消息指标**
   - 消息接收速率
   - 消息发送成功率
   - 消息处理延迟

3. **系统指标**
   - CPU 使用率
   - 内存使用率
   - 磁盘 I/O
   - 网络流量

### 告警配置

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

## 联系支持

如果以上方法无法解决问题，请提供以下信息：

1. **系统信息**
   - 系统版本
   - 操作系统
   - 数据库类型和版本

2. **错误信息**
   - 完整的错误日志
   - 错误发生时间
   - 错误复现步骤

3. **配置信息**
   - 相关配置文件（脱敏）
   - 适配器配置
   - 路由规则配置

4. **环境信息**
   - 网络环境
   - 部署方式
   - 相关服务状态
