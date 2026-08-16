# 淘宝接入器使用指南

> 本文档介绍如何在 E-SP-Line2 中使用「淘宝消息接入器」，从 WebUI 完成配置、启动与停止，支持多开多配置。

## 简介

淘宝接入器将 [TaobaoApis](https://github.com/cv-cat/TaobaoApis) 淘宝逆向 SDK 完整内置到 E-SP-Line2，通过 ESPL v3 中间层与后端 WebSocket 通信。每个**实例**对应一个淘宝账号，可同时运行多个实例（多开）。

```
淘宝 WebSocket ──► 淘宝接入器(Python) ──► ESPL v3 中间层 ──► E-SP-Line2 后端 ──► WebUI
                    adapters/taobao/          esp_bridge.py        消息入库/路由
```

## 前置要求

- E-SP-Line2 后端已运行（`bin/e-sp-line2`）
- Python 3.9+ 和 Node.js 18+（签名 JS）
- 已安装接入器 Python 依赖：

```bash
cd adapters/taobao
pip install -r requirements.txt   # 或使用 venv: python3 -m venv venv && source venv/bin/activate
```

## 配置流程（WebUI）

1. **平台管理**：创建平台，选择「淘宝」（平台代码 `taobao`）
2. **接入器管理**：创建接入器，所属平台选「淘宝」，运行时选「Python」
3. **实例管理**：创建实例，所属接入器选「淘宝消息适配器」，填写：
   - **淘宝 Cookie**：登录 [taobao.com](https://www.taobao.com) 后，从浏览器开发者工具复制完整 Cookie
   - **Device ID**：可选，留空自动生成
   - **心跳间隔 / 重连延迟**：可选参数

## 启动与停止（WebUI）

在「实例管理」页面：
- 点击 **▶ 启动**：后端自动在 `adapters/taobao/` 目录拉起 Python 接入器进程，实例状态变为 `运行中`
- 点击 **⏹ 停止**：后端自动杀掉该进程，实例状态变为 `已停止`
- 进程异常崩溃会自动重启（可在 `config.yaml` 的 `adapter.auto_restart` 关闭）
- 删除运行中的实例或接入器，后端会自动先停止其进程，避免孤儿进程

后端启动 Python 进程的路径在 `config/config.yaml` 配置：

```yaml
adapter:
  python_bin: "python3"        # 建议指向 venv 中的 python，如 /path/to/venv/bin/python
  adapters_dir: "adapters"     # 接入器根目录
  auto_restart: true           # 崩溃自动重启
```

## 多开多配置

每个实例独立一份 Cookie/device_id 配置，支持同时运行多个淘宝账号：

- 在「实例管理」为每个淘宝账号创建独立实例，分别填写各自的 Cookie
- 分别点击「启动」，后端为每个实例拉起独立进程
- 各实例消息独立入库，可在「消息管理」查看

## 消息与指令

- **收到消息**：淘宝买家私信 → 接入器转换为 ESPL v3 payload → 上报后端入库，可在「消息管理」查看
- **发送指令**：通过后端创建 `send_text` / `send_image` 出站指令，接入器收到后调用淘宝 API 发送

## 常见问题

- **实例启动报「没有配置 Cookie」**：请在实例管理编辑该实例，填写淘宝 Cookie
- **进程启动后立即退出**：检查 Python 依赖是否安装（`requirements.txt`）、Cookie 是否有效
- **Node.js 缺失**：签名 JS 需要 Node.js 18+，请先安装

> ⚠️ 逆向接口仅供学习研究，请遵守淘宝平台规则与当地法律法规。
