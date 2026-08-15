# 闲鱼 (XianYu) 接入器 for E-SP-Line2

> 将 [XianYuApis](https://github.com/cv-cat/XianYuApis) 闲鱼平台逆向 SDK 完整内置到 E-SP-Line2，
> 通过中间层转换为 ESPL v3 协议与后端通信，支持多开多配置，配置在 WebUI 完成。

## 架构

```
┌──────────────────────────────┐      WebSocket       ┌──────────────────────┐
│   E-SP-Line2 后端 (Go)        │ ◄──────────────────► │  Python 接入器         │
│   /ws/adapter?instance_id=xx │                      │  adapters/xianyu/     │
│                              │                      │  ├─ main.py           │
│  WebUI 实例管理(配置 Cookie)   │                      │  ├─ esp_bridge.py     │
└──────────────────────────────┘                      │  └─ vendor/ (XianYuApis)│
                                                      └──────────────────────┘
```

- **main.py**：接入器入口，`--instance-id` 指定实例(支持逗号分隔多开)
- **esp_bridge.py**：ESPL v3 中间层，从后端拉取实例配置，通过 WebSocket 连接后端，转换消息协议
- **vendor/**：XianYuApis 原始 SDK(goofish_apis / goofish_live / utils / message / static)，保留完整逆向算法

## 多开多配置

每个**实例**在 WebUI「实例管理」中独立配置：
- `cookie`：该闲鱼账号的登录 Cookie(每个实例不同)
- `device_id`：设备 ID(可选，留空自动生成)
- `heartbeat_interval` / `reconnect_delay`：心跳与重连参数

一个接入器(xianyu-adapter-v1)可创建 N 个实例，每个实例对应一个闲鱼账号，**可同时运行**。

## 配置流程(WebUI)

1. **平台管理**：创建平台，选择「闲鱼」(平台代码 `xianyu`)
2. **接入器管理**：创建接入器，所属平台选「闲鱼」，运行时选「Python」
3. **实例管理**：创建实例，所属接入器选「闲鱼消息适配器」，填写 Cookie 等配置
4. **启动接入器**：

```bash
# 安装依赖(需 Node.js 运行签名 JS)
pip install -r adapters/xianyu/requirements.txt
npm install -g jsdom  # 若 execjs 需要

# 单实例
python adapters/xianyu/main.py --instance-id <INSTANCE_ID> --backend http://localhost:8080

# 多实例(多开)
python adapters/xianyu/main.py --instance-id <ID1>,<ID2>,<ID3> --backend http://localhost:8080

# 带后端 JWT token
python adapters/xianyu/main.py --instance-id <ID> --backend http://localhost:8080 --token <JWT>
```

## 依赖

- Python 3.9+
- Node.js 18+(sign 签名 JS)
- 见 `requirements.txt`

> ⚠️ 逆向接口仅供学习研究，请遵守闲鱼平台规则与当地法律法规。
