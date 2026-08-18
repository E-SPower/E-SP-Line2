# 协议体系总览（Protocol Overview）

> 本文档定义 E-SP-Line2 的**完整协议体系**，明确各协议的角色、归属与状态。
> 这是理解其他协议文档的入口，请先阅读本文。

## 1. 协议全景

E-SP-Line2 存在 **三套协议**，分别服务于不同的通信场景：

| 协议 | 版本 | 传输 | 角色 | 状态 |
|------|------|------|------|------|
| **ESPL v3** | v3 | WebSocket | 接入器（Adapter）与外部系统（机器人框架/客服系统）通信 | ✅ 现行 |
| **Bridge 协议** | — | WebSocket | 桥（Bridge）进程与后端核心通信 | ✅ 现行 |
| **老版接入器 HTTP** | v1 / v2 | HTTP | 老版接入器与后端核心的旧式 HTTP 通信 | ⛔ 已弃用 |

```
┌─────────────────────────────────────────────────────────────────────┐
│                        E-SP-Line2 核心 (Go)                          │
│                                                                     │
│  ┌──────────────┐   Bridge 协议 (WS)   ┌─────────────────────────┐  │
│  │  桥 (Bridge)  │ ◄──────────────────► │  消息核心 / 路由 / 入库   │  │
│  │  Python 进程  │   /ws/adapter        └───────────┬─────────────┘  │
│  │  淘宝/闲鱼    │                                  │                │
│  └──────────────┘                                  │                │
│                                                    ▼                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  接入器网关 (Adapter Gateway) — ESPL v3 协议 (WS)              │  │
│  │  • 服务端模式：外部系统携带 key 连入 /ws                        │  │
│  │  • 客户端模式：核心携带 key 主动连接外部系统                     │  │
│  └───────────────────────────────┬───────────────────────────────┘  │
│                                  │ ESPL v3 (WS)                     │
└──────────────────────────────────┼──────────────────────────────────┘
                                   ▼
                    ┌──────────────────────────┐
                    │  外部系统（机器人/客服等）  │
                    └──────────────────────────┘
```

## 2. 协议职责划分

### 2.1 ESPL v3（接入器协议）— WebSocket

- **用途**：接入器（Adapter）与外部系统（机器人框架、客服系统等）之间的消息通信。
- **角色**：外部系统作为客户端接入 E-SP-Line2，消费电商消息并回复。
- **传输**：WebSocket（`/ws`、`/ws/adapter-gateway` 等）。
- **详细规范**：[ESPL v3 协议](./protocol-v3.md)
- **接入器网关文档**：[WebSocket 接入器](../user-guide/adapter-gateway.md)

### 2.2 Bridge 协议（桥协议）— WebSocket

- **用途**：桥（Bridge）进程（淘宝/闲鱼 Python 进程）与后端核心之间的通信。
- **角色**：桥连接电商平台，将平台消息上报核心，并接收核心下发的出站指令。
- **传输**：WebSocket（`/ws/adapter?instance_id=xxx`）。
- **详细规范**：[桥（Bridge）协议](./bridge-protocol.md)
- **参考实现**：[`internal/adapter/bridge.go`](../../internal/adapter/bridge.go)（`BridgeClient` / `BridgeManager`）

### 2.3 老版接入器 HTTP（v1 / v2）— 已弃用

- **用途**：早期版本中接入器通过 HTTP 与后端通信（注册、注销、状态查询、消息发送）。
- **状态**：⛔ **已弃用**。接入器已迁移到 WebSocket ESPL v3 协议，桥进程已迁移到
  WebSocket Bridge 协议。
- **说明**：请勿在新代码中使用老版 HTTP v1/v2 协议。

## 3. 协议选择指南

| 场景 | 使用协议 |
|------|----------|
| 外部机器人/客服系统接入 E-SP-Line2 | **ESPL v3**（WebSocket） |
| 开发新的桥（连接新电商平台） | **Bridge 协议**（WebSocket） |
| 通过 HTTP 调用管理 API | REST API（`/api/v1/*`，管理用途，非消息协议） |
| 老版接入器 HTTP 通信 | ❌ 已弃用，请迁移到 ESPL v3 / Bridge 协议 |

## 4. 相关文档

- [ESPL v3 协议规范](./protocol-v3.md)
- [桥（Bridge）协议规范](./bridge-protocol.md)
- [WebSocket 接入器（Adapter Gateway）](../user-guide/adapter-gateway.md)
- [REST API 参考](./api-reference.md)
- [桥 vs 接入器架构](../architecture-bridge-vs-adapter.md)
