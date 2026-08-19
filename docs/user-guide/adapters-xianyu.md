# 闲鱼接入器使用指南

> 本文档介绍如何在 E-SP-Line2 中使用「闲鱼消息接入器」，从 WebUI 完成配置、启动与停止，支持多开多配置。

## 简介

闲鱼接入器将 [XianYuApis](https://github.com/cv-cat/XianYuApis) 闲鱼逆向 SDK 完整内置到 E-SP-Line2，通过 ESPL v3 中间层与后端 WebSocket 通信。每个**实例**对应一个闲鱼账号，可同时运行多个实例（多开）。

```
闲鱼 WebSocket ──► 闲鱼接入器(Python) ──► ESPL v3 中间层 ──► E-SP-Line2 后端 ──► WebUI
                    adapters/xianyu/          esp_bridge.py        消息入库/路由
```

## 前置要求

- E-SP-Line2 后端已运行（`bin/e-sp-line2`）
- Python 3.9+ 和 Node.js 18+（签名 JS）
- 已安装接入器 Python 依赖：

```bash
cd adapters/xianyu
pip install -r requirements.txt   # 或使用 venv: python3 -m venv venv && source venv/bin/activate
```

## 配置流程（WebUI）

1. **平台管理**：创建平台，选择「闲鱼」（平台代码 `xianyu`）
2. **接入器管理**：创建接入器，所属平台选「闲鱼」，运行时选「Python」
3. **实例管理**：创建实例，所属接入器选「闲鱼消息适配器」，填写：
   - **闲鱼 Cookie**：登录 [goofish.com](https://www.goofish.com) 后，从浏览器开发者工具复制完整 Cookie
   - **Device ID**：可选，留空自动生成
   - **心跳间隔 / 重连延迟**：可选参数

## 获取闲鱼 Cookie（重要）

> ⚠️ **Cookie 必须是登录后的完整状态**，否则实例会报「Cookie 无效或已过期」。
> 匿名 Cookie（只有 `cna`/`sca`/`aui` 等）无法使用，必须包含登录态字段。

### 需要填写的 Cookie 字段

在 WebUI「闲鱼 Cookie」中填写的字符串，**必须包含以下登录态字段**（`Name=Value` 对，用 `; ` 连接）：

| 字段 | 作用 | 是否必需 |
|------|------|---------|
| `unb` | 用户 ID（数字） | ✅ 必需 |
| `cookie2` | 登录令牌 | ✅ 必需 |
| `tracknick` | 用户名（URL 编码） | ✅ 必需 |
| `sgcookie` | 会话 Cookie | ✅ 必需 |
| `_m_h5_tk` | mtop 签名令牌 | ⚠️ 可选（缺失时自动获取） |
| `_m_h5_tk_enc` | mtop 加密令牌 | ⚠️ 可选（缺失时自动获取） |
| `cna` | 设备标识 | ⚠️ 可选 |
| `tfstk` | 风控令牌 | ⚠️ 可选 |
| `csg` | 风控标识 | ⚠️ 可选 |
| `t` / `xlly_s` / `mtop_partitioned_detect` 等 | 辅助字段 | 可选 |

**判断标准**：只要 Cookie 里同时有 `unb` + `cookie2` + `tracknick` + `sgcookie`，就是有效的登录 Cookie。

### 方式一：扫码登录（推荐，自动获取完整 Cookie）

项目内置了扫码登录工具，用闲鱼 APP 扫码即可自动获取完整登录 Cookie：

```bash
cd adapters/xianyu
python3 qrcode_login.py
```

运行后：
1. 终端显示二维码（用闲鱼 APP 左上角「扫一扫」扫码）
2. 手机确认登录
3. 脚本自动输出**完整登录 Cookie**（包含 `unb`/`cookie2`/`tracknick`/`sgcookie`），并保存到 `adapters/xianyu/cookie.txt`
4. 将输出的 Cookie 复制到 WebUI「实例管理 → 编辑实例 → 闲鱼 Cookie」

> 需要安装 `qrcode` 库才能在终端显示二维码（可选）：
> ```bash
> pip install qrcode
> ```

### 方式二：浏览器手动复制

1. 用浏览器登录 [goofish.com](https://www.goofish.com)
2. 按 `F12` 打开开发者工具 → `Application`（应用）→ `Cookies` → `https://www.goofish.com`
3. 复制**所有** Cookie（`Name=Value` 对，用 `; ` 连接）
4. **确认包含 `unb`、`cookie2`、`tracknick`、`sgcookie` 四个登录字段**，否则无效
5. 将完整 Cookie 填入 WebUI

### 常见错误

| 错误信息 | 原因 | 解决 |
|---------|------|------|
| `会话中缺少 _m_h5_tk 令牌` | Cookie 缺少 mtop 签名令牌 | 已自动修复：适配器会自动获取 `_m_h5_tk` |
| `FAIL_SYS_SESSION_EXPIRED::Session过期` | Cookie 没有登录态（缺少 `unb`/`cookie2`） | 使用扫码登录或复制完整登录 Cookie |
| `获取token失败：Cookie 无效或已过期` | 登录态失效 | 重新扫码登录获取新 Cookie |

## 启动与停止（WebUI）

在「实例管理」页面：
- 点击 **▶ 启动**：后端自动在 `adapters/xianyu/` 目录拉起 Python 接入器进程，实例状态变为 `运行中`
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

每个实例独立一份 Cookie/device_id 配置，支持同时运行多个闲鱼账号：

- 在「实例管理」为每个闲鱼账号创建独立实例，分别填写各自的 Cookie
- 分别点击「启动」，后端为每个实例拉起独立进程
- 各实例消息独立入库，可在「消息管理」查看

## 消息与指令

- **收到消息**：闲鱼买家私信 → 接入器转换为 ESPL v3 payload → 上报后端入库，可在「消息管理」查看
- **发送指令**：通过后端创建 `send_text` / `send_image` 出站指令，接入器收到后调用闲鱼 API 发送

## 常见问题

- **实例启动报「没有配置 Cookie」**：请在实例管理编辑该实例，填写闲鱼 Cookie
- **进程启动后立即退出**：检查 Python 依赖是否安装（`requirements.txt`）、Cookie 是否有效
- **Node.js 缺失**：签名 JS 需要 Node.js 18+，请先安装

> ⚠️ 逆向接口仅供学习研究，请遵守闲鱼平台规则与当地法律法规。
