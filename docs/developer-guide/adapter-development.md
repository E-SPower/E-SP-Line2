# 适配器开发指南

## 概述

本指南介绍如何为 E-SP-Line2 开发新的平台适配器。适配器负责与特定电商平台通信，实现消息的接收和发送。

## 架构概览

```
┌─────────────────┐     HTTP/gRPC     ┌─────────────────┐
│   Go Backend    │ ◄──────────────► │ Python Adapter  │
│                 │                   │                 │
│  Adapter Manager│                   │  Platform SDK   │
└─────────────────┘                   └─────────────────┘
```

## 适配器结构

### 1. 适配器包 (Adapter Package)

适配器包是适配器的元数据和配置定义。

```go
type AdapterInfo struct {
    ID              string            `json:"id"`
    PlatformID      string            `json:"platform_id"`
    Name            string            `json:"name"`
    Version         string            `json:"version"`
    RuntimeType     string            `json:"runtime_type"` // python, node, go
    ProtocolVersion string            `json:"protocol_version"`
    Capabilities    []string          `json:"capabilities"`
    ConfigSchema    map[string]interface{} `json:"config_schema"`
    I18n            map[string]I18nResource `json:"i18n"`
    Operations      OperationPolicy   `json:"operations"`
    Security        SecurityPolicy    `json:"security"`
    Status          string            `json:"status"`
}
```

### 2. 适配器实例 (Adapter Instance)

适配器实例是具体的账号或店铺配置。

```go
type Instance struct {
    ID            string                 `json:"id"`
    AdapterID     string                 `json:"adapter_id"`
    PlatformID    string                 `json:"platform_id"`
    Name          string                 `json:"name"`
    Status        InstanceStatus         `json:"status"`
    Config        map[string]interface{} `json:"config"`
    Credentials   map[string]interface{} `json:"credentials"`
    ConnectedAt   *time.Time             `json:"connected_at,omitempty"`
    LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
    MessageCount  int64                  `json:"message_count"`
    ErrorCount    int64                  `json:"error_count"`
    LastError     string                 `json:"last_error,omitempty"`
}
```

## 开发步骤

### 步骤 1：定义适配器信息

创建适配器信息文件 `adapters/myplatform.go`：

```go
package adapter

import (
    "github.com/e-spl/e-sp-line2/internal/protocol/v3"
)

func GetMyPlatformAdapterInfo() *AdapterInfo {
    return &AdapterInfo{
        ID:              "myplatform-adapter-v1",
        PlatformID:      "myplatform",
        Name:            "MyPlatform Message Adapter",
        Version:         "1.0.0",
        RuntimeType:     "python",
        ProtocolVersion: "v3",
        Capabilities: []string{
            "receive_message",
            "send_text",
            "send_image",
            "upload_media",
            "get_history",
        },
        ConfigSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "api_key": map[string]interface{}{
                    "type":        "string",
                    "description": "API Key",
                },
                "api_secret": map[string]interface{}{
                    "type":        "string",
                    "description": "API Secret",
                },
            },
            "required": []string{"api_key", "api_secret"},
        },
        I18n: map[string]v3.I18nResource{
            "zh-CN": {
                DisplayName:  "我的平台适配器",
                Description:  "支持我的平台消息收发",
                InstallGuide: "1. 获取 API Key\n2. 配置适配器",
            },
            "en-US": {
                DisplayName:  "MyPlatform Adapter",
                Description:  "Supports MyPlatform message sending and receiving",
                InstallGuide: "1. Get API Key\n2. Configure adapter",
            },
        },
        Operations: v3.OperationPolicy{
            HeartbeatInterval: 30,
            ReconnectDelay:    5,
            MaxRetries:        3,
            MaxQueueSize:      1000,
        },
        Security: v3.SecurityPolicy{
            SensitiveFields:  []string{"api_key", "api_secret"},
            EncryptedFields:  []string{"api_key", "api_secret"},
            PermissionScopes: []string{"message:read", "message:write"},
        },
        Status: "active",
    }
}
```

### 步骤 2：注册适配器

在 `adapters.go` 中注册适配器：

```go
func RegisterBuiltinAdapters(registry *Registry) error {
    // 注册现有适配器
    if err := registry.Register(GetTaobaoAdapterInfo()); err != nil {
        return err
    }
    if err := registry.Register(GetXianyuAdapterInfo()); err != nil {
        return err
    }
    
    // 注册新适配器
    if err := registry.Register(GetMyPlatformAdapterInfo()); err != nil {
        return err
    }
    
    return nil
}
```

### 步骤 3：实现 Python 适配器

创建 Python 适配器文件 `adapters/myplatform/adapter.py`：

```python
import asyncio
import json
import logging
from typing import Dict, Any, Optional
from dataclasses import dataclass

import aiohttp
from websockets import connect

logger = logging.getLogger(__name__)

@dataclass
class AdapterConfig:
    api_key: str
    api_secret: str
    webhook_url: str
    heartbeat_interval: int = 30

class MyPlatformAdapter:
    def __init__(self, config: AdapterConfig):
        self.config = config
        self.ws: Optional[any] = None
        self.session: Optional[aiohttp.ClientSession] = None
        self.running = False
        
    async def start(self):
        """Start the adapter"""
        self.running = True
        self.session = aiohttp.ClientSession()
        
        # Connect to platform WebSocket
        await self._connect_websocket()
        
        # Start heartbeat
        asyncio.create_task(self._heartbeat_loop())
        
        logger.info("Adapter started")
        
    async def stop(self):
        """Stop the adapter"""
        self.running = False
        if self.ws:
            await self.ws.close()
        if self.session:
            await self.session.close()
        logger.info("Adapter stopped")
        
    async def _connect_websocket(self):
        """Connect to platform WebSocket"""
        url = "wss://api.myplatform.com/ws"
        headers = {
            "Authorization": f"Bearer {self.config.api_key}"
        }
        
        self.ws = await connect(url, extra_headers=headers)
        logger.info("WebSocket connected")
        
        # Start message listener
        asyncio.create_task(self._message_loop())
        
    async def _message_loop(self):
        """Listen for incoming messages"""
        async for message in self.ws:
            try:
                data = json.loads(message)
                await self._handle_message(data)
            except Exception as e:
                logger.error(f"Message handling error: {e}")
                
    async def _handle_message(self, data: Dict[str, Any]):
        """Handle incoming message"""
        # Convert to V3 protocol format
        envelope = {
            "protocol_version": "v3",
            "event_type": "message.received",
            "platform": "myplatform",
            "payload": {
                "conversation_id": data.get("conversation_id"),
                "sender_id": data.get("sender_id"),
                "sender_name": data.get("sender_name"),
                "message_type": data.get("message_type"),
                "message_content": data.get("content"),
            }
        }
        
        # Send to backend
        await self._send_to_backend(envelope)
        
    async def _send_to_backend(self, envelope: Dict[str, Any]):
        """Send message to backend"""
        url = f"{self.config.webhook_url}/api/v1/messages"
        headers = {
            "Content-Type": "application/json",
            "X-API-Key": self.config.api_key
        }
        
        async with self.session.post(url, json=envelope, headers=headers) as resp:
            if resp.status != 200:
                logger.error(f"Failed to send message: {resp.status}")
                
    async def _heartbeat_loop(self):
        """Send heartbeat periodically"""
        while self.running:
            try:
                if self.ws:
                    await self.ws.ping()
                await asyncio.sleep(self.config.heartbeat_interval)
            except Exception as e:
                logger.error(f"Heartbeat error: {e}")
                await self._reconnect()
                
    async def _reconnect(self):
        """Reconnect to platform"""
        logger.info("Reconnecting...")
        await asyncio.sleep(5)
        await self._connect_websocket()
        
    async def send_message(self, target_id: str, message: Dict[str, Any]):
        """Send message to platform"""
        if not self.ws:
            raise RuntimeError("WebSocket not connected")
            
        payload = {
            "action": "send_message",
            "target_id": target_id,
            "message_type": message.get("type"),
            "content": message.get("content")
        }
        
        await self.ws.send(json.dumps(payload))
        logger.info(f"Message sent to {target_id}")

# Entry point
async def main():
    config = AdapterConfig(
        api_key="your_api_key",
        api_secret="your_api_secret",
        webhook_url="http://localhost:8080"
    )
    
    adapter = MyPlatformAdapter(config)
    
    try:
        await adapter.start()
        # Keep running
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        await adapter.stop()

if __name__ == "__main__":
    asyncio.run(main())
```

### 步骤 4：创建 requirements.txt

创建 `adapters/myplatform/requirements.txt`：

```
aiohttp>=3.9.0
websockets>=12.0
```

### 步骤 5：创建 Dockerfile（可选）

创建 `adapters/myplatform/Dockerfile`：

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "adapter.py"]
```

## 能力声明

### 标准能力

| 能力 | 说明 |
|------|------|
| `receive_message` | 接收消息 |
| `send_text` | 发送文本消息 |
| `send_image` | 发送图片消息 |
| `send_audio` | 发送音频消息 |
| `send_video` | 发送视频消息 |
| `upload_media` | 上传媒体文件 |
| `get_history` | 获取历史消息 |
| `create_conversation` | 创建会话 |
| `refresh_token` | 刷新登录态 |

### 电商特有能力

| 能力 | 说明 |
|------|------|
| `get_product_info` | 获取商品信息 |
| `get_order_info` | 获取订单信息 |
| `publish_product` | 发布商品 |

## 配置 Schema

### 基础配置

```json
{
  "type": "object",
  "properties": {
    "api_key": {
      "type": "string",
      "description": "API Key"
    },
    "api_secret": {
      "type": "string",
      "description": "API Secret"
    },
    "webhook_url": {
      "type": "string",
      "description": "Webhook URL",
      "default": "http://localhost:8080"
    }
  },
  "required": ["api_key", "api_secret"]
}
```

### Cookie 配置（淘宝/闲鱼）

```json
{
  "type": "object",
  "properties": {
    "cookie": {
      "type": "string",
      "description": "Platform login cookie"
    },
    "device_id": {
      "type": "string",
      "description": "Device ID (optional)"
    }
  },
  "required": ["cookie"]
}
```

## 多语言支持

### 配置多语言资源

```go
I18n: map[string]v3.I18nResource{
    "zh-CN": {
        DisplayName:  "我的平台适配器",
        Description:  "支持我的平台消息收发",
        InstallGuide: "1. 登录平台\n2. 获取 API Key\n3. 配置适配器",
        ErrorMessages: map[string]string{
            "auth_failed": "认证失败，请检查 API Key",
            "connection_failed": "连接失败，请检查网络",
        },
    },
    "en-US": {
        DisplayName:  "MyPlatform Adapter",
        Description:  "Supports MyPlatform message sending and receiving",
        InstallGuide: "1. Login to platform\n2. Get API Key\n3. Configure adapter",
        ErrorMessages: map[string]string{
            "auth_failed": "Authentication failed, please check API Key",
            "connection_failed": "Connection failed, please check network",
        },
    },
}
```

## 测试

### 单元测试

```go
func TestMyPlatformAdapter(t *testing.T) {
    info := GetMyPlatformAdapterInfo()
    
    assert.Equal(t, "myplatform-adapter-v1", info.ID)
    assert.Equal(t, "myplatform", info.PlatformID)
    assert.Contains(t, info.Capabilities, "receive_message")
}
```

### 集成测试

```python
import pytest
from adapter import MyPlatformAdapter, AdapterConfig

@pytest.mark.asyncio
async def test_adapter_start():
    config = AdapterConfig(
        api_key="test_key",
        api_secret="test_secret",
        webhook_url="http://localhost:8080"
    )
    
    adapter = MyPlatformAdapter(config)
    await adapter.start()
    
    assert adapter.running
    await adapter.stop()
```

## 最佳实践

### 1. 错误处理

- 实现自动重连机制
- 记录详细的错误日志
- 向用户友好的错误信息

### 2. 性能优化

- 使用连接池
- 实现消息队列
- 批量处理消息

### 3. 安全考虑

- 加密存储敏感信息
- 验证所有输入
- 使用 HTTPS/WSS

### 4. 可观测性

- 添加健康检查端点
- 实现指标收集
- 支持分布式追踪

## 参考资源

- [淘宝适配器实现](../../TaoBaoApis/)
- [闲鱼适配器实现](../../XianYuApis/)
- [V3 协议文档](./protocol-v3.md)
- [API 参考文档](./api-reference.md)
