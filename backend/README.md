# E-SP-Line2 Backend

Go 后端服务，提供平台化接入器管理、消息路由和 WebSocket 通信功能。

## 功能特性

- **平台管理**: 支持多平台接入（闲鱼、淘宝等）
- **接入器管理**: 动态管理接入器包和实例
- **消息路由**: 基于规则的消息路由系统
- **WebSocket 通信**: 实时消息推送和接收
- **认证授权**: JWT 认证和 RBAC 权限控制
- **数据持久化**: 支持 SQLite 和 PostgreSQL

## 技术栈

- **框架**: Gin
- **数据库**: GORM + SQLite/PostgreSQL
- **缓存**: Redis
- **WebSocket**: Gorilla WebSocket
- **认证**: JWT
- **日志**: Zap

## 项目结构

```
backend/
├── cmd/                    # 命令行工具
├── internal/               # 内部包
│   ├── config/            # 配置管理
│   ├── handler/           # HTTP 处理器
│   ├── middleware/        # 中间件
│   ├── models/            # 数据模型
│   ├── repository/        # 数据访问层
│   ├── server/            # 服务器
│   └── service/           # 业务逻辑层
├── pkg/                   # 公共包
│   └── logger/            # 日志工具
├── config/                # 配置文件
├── data/                  # 数据目录
├── main.go               # 入口文件
├── go.mod                # Go 模块定义
└── Makefile              # 构建脚本
```

## 快速开始

### 环境要求

- Go 1.22+
- Redis (可选)
- PostgreSQL (可选，默认使用 SQLite)

### 安装依赖

```bash
make deps
```

### 配置

复制并编辑配置文件：

```bash
cp config/config.yaml.example config/config.yaml
```

### 运行

```bash
# 开发模式
make run

# 构建
make build

# 运行测试
make test
```

## API 文档

### 认证接口

- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `GET /api/v1/auth/me` - 获取当前用户信息

### 平台接口

- `GET /api/v1/platforms` - 获取平台列表
- `GET /api/v1/platforms/:id` - 获取平台详情

### 接入器接口

- `GET /api/v1/adapters` - 获取接入器列表
- `POST /api/v1/adapters` - 创建接入器
- `GET /api/v1/adapters/:id` - 获取接入器详情
- `PUT /api/v1/adapters/:id` - 更新接入器
- `DELETE /api/v1/adapters/:id` - 删除接入器
- `POST /api/v1/adapters/:id/start` - 启动接入器
- `POST /api/v1/adapters/:id/stop` - 停止接入器

### 实例接口

- `GET /api/v1/instances` - 获取实例列表
- `POST /api/v1/instances` - 创建实例
- `GET /api/v1/instances/:id` - 获取实例详情
- `PUT /api/v1/instances/:id` - 更新实例
- `DELETE /api/v1/instances/:id` - 删除实例

### 消息接口

- `GET /api/v1/messages` - 获取消息列表
- `GET /api/v1/messages/:id` - 获取消息详情
- `POST /api/v1/messages/:id/ack` - 确认消息

### 命令接口

- `GET /api/v1/commands` - 获取命令列表
- `POST /api/v1/commands` - 创建命令
- `GET /api/v1/commands/:id` - 获取命令详情

### 路由接口

- `GET /api/v1/routes` - 获取路由规则列表
- `POST /api/v1/routes` - 创建路由规则
- `GET /api/v1/routes/:id` - 获取路由规则详情
- `PUT /api/v1/routes/:id` - 更新路由规则
- `DELETE /api/v1/routes/:id` - 删除路由规则

### WebSocket 接口

- `ws://localhost:8080/ws/adapter?instance_id=xxx` - 接入器 WebSocket
- `ws://localhost:8080/ws/app?app_id=xxx` - 应用 WebSocket

## 开发指南

### 添加新的 API 端点

1. 在 `internal/handler/` 中创建处理器函数
2. 在 `internal/server/server.go` 中注册路由
3. 在 `internal/service/` 中实现业务逻辑
4. 在 `internal/repository/` 中实现数据访问

### 数据库迁移

使用 GORM 的 AutoMigrate 功能自动迁移数据库模型。

### 日志

使用 Zap 结构化日志，支持多种日志级别。

## 许可证

Apache License 2.0
