# E-SP-Line2

[English](README.md) · **简体中文**

平台化接入器管理系统 - Platform Adapter Management System

## 项目简介

E-SP-Line2 是一个基于平台化架构的接入器管理系统，支持多平台（闲鱼、淘宝、千牛等）的消息接入、路由和投递。系统采用四层架构设计，支持异步消息处理、多重可用性保障和多语言国际化。

## 核心特性

- **平台化管理**: 支持动态添加多个电商平台
- **接入器架构**: 统一的接入器契约和生命周期管理
- **异步消息处理**: 基于事件总线的异步消息流
- **多重可用性**: 主备租约、故障隔离和自动恢复
- **多语言支持**: 前端和接入器 Manifest 多语言资源
- **WebSocket 通信**: 实时消息推送和接收

## 技术栈

### 后端
- **语言**: Go 1.22+
- **框架**: Gin
- **数据库**: GORM + SQLite/PostgreSQL
- **缓存**: Redis
- **WebSocket**: Gorilla WebSocket

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI 组件**: Tailwind CSS + shadcn/ui
- **国际化**: i18next
- **路由**: React Router DOM

## 项目结构

```
E-SP-Line2/
├── backend/              # Go 后端服务
│   ├── cmd/             # 命令行工具
│   ├── internal/        # 内部包
│   │   ├── config/     # 配置管理
│   │   ├── handler/    # HTTP 处理器
│   │   ├── middleware/ # 中间件
│   │   ├── models/     # 数据模型
│   │   ├── repository/ # 数据访问层
│   │   ├── server/     # 服务器
│   │   └── service/    # 业务逻辑层
│   ├── pkg/            # 公共包
│   └── config/         # 配置文件
├── web/                # React 前端
│   ├── src/
│   │   ├── components/ # 组件
│   │   ├── pages/      # 页面
│   │   ├── locales/    # 多语言资源
│   │   └── i18n.ts     # 国际化配置
│   └── package.json
└── docs/               # 文档
```

## 快速开始

### 环境要求

- Go 1.22+
- Node.js 20.11+
- pnpm 或 npm
- Redis (可选)

### 后端启动

```bash
cd backend

# 安装依赖
make deps

# 运行服务
make run
```

后端服务将在 `http://localhost:8080` 启动。

### 前端启动

```bash
cd web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

前端开发服务器将在 `http://localhost:3000` 启动。

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

### WebSocket 接口

- `ws://localhost:8080/ws/adapter?instance_id=xxx` - 接入器 WebSocket
- `ws://localhost:8080/ws/app?app_id=xxx` - 应用 WebSocket

## 架构设计

### 四层架构

1. **平台层**: 定义业务平台类型（闲鱼、淘宝、千牛等）
2. **接入器包层**: 具体平台的接入实现，带版本和能力声明
3. **接入器实例层**: 真实账号或店铺的运行配置
4. **运行会话层**: 实例当前在线连接状态

### 消息流

```
平台 WebSocket → 接入器实例 → 事件总线 → 核心路由 → 下游应用
                                              ↓
                                        出站命令队列
                                              ↓
                                        接入器实例 → 平台发送
```

## 开发指南

### 添加新平台

平台是数据。在 `matrix.json` 里加一列，并在 `scripts/gen_platformspec.py` 的 `META` / `CONFIG` 里加一项，然后重新生成：

```bash
python3 scripts/gen_platformspec.py matrix.json backend/internal/platformspec
python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
```

注册表驱动适配器 Manifest、能力矩阵和公开的 `GET /api/v1/platforms/spec`，内置适配器在启动时同步进库。详见[适配器开发指南](docs/developer-guide/adapter-development.md)。

### 开发接入器

遵循[消息字段契约](docs/developer-guide/message-fields.md)与[适配器开发指南](docs/developer-guide/adapter-development.md)，按能力支持度决定要实现的命令。上游项目 `Spider_XHS`、`DouYin_Spider`、`TiktokApis`、`BilibiliApis`、`KuaiShou-Spider`、`WeiboApis`、`XianYuApis`、`TaoBaoApis`、`JdApis`、`XApis` 是参考实现；平台元数据迁移自 [catbus](https://github.com/cv-cat/catbus)（[迁移说明](docs/developer-guide/catbus-migration.md)、[声明](NOTICE-catbus.md)）。

### 文档

- [平台能力矩阵](docs/developer-guide/platform-capabilities.md)：各平台的支持度与限制
- [消息字段](docs/developer-guide/message-fields.md)：每个字段的类型、空值与来源
- [API 参考](docs/developer-guide/api-reference.md)

## 许可证

本项目采用 Apache License 2.0 许可证，附加条款见 [LICENSE-CUSTOM.md](LICENSE-CUSTOM.md)，并以 [MIT 许可](LICENSE-MIT) 另行提供。详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request！请先阅读 [贡献指南](.github/CONTRIBUTING.md) 与[行为准则](.github/CODE_OF_CONDUCT.md)。安全问题请按[安全策略](.github/SECURITY.md)私下报告。

## 联系方式

如有问题或建议，请通过以下方式联系：

- GitHub Issues: https://github.com/E-SPower/E-SP-Line2/issues
