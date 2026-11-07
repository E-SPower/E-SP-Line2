# 接入器 adapter.yaml 字段规则

> 接入器是**现实存在**的：每个接入器对应 `adapters/` 下的一个子目录，目录内的
> `adapter.yaml` 定义了该接入器的元数据、实例配置字段、是否在 WebUI 显示等。
> 后端启动时扫描 `adapters/*/adapter.yaml`，自动生成可用接入器列表。

## 目录结构

```
adapters/
├── xianyu/
│   ├── adapter.yaml      # 闲鱼接入器定义
│   ├── main.py           # 接入器入口（Python 进程）
│   └── vendor/           # 平台 SDK（可选）
└── taobao/
    ├── adapter.yaml      # 淘宝接入器定义
    └── main.py
```

## 字段规则

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | 接入器唯一标识（如 `xianyu-adapter`），用于实例关联与路由 |
| `platform_code` | string | ✅ | 处理的消息平台代码，须与 `config/form-options.yaml` 的 `platform_codes` 一致（如 `xianyu`、`taobao`） |
| `name` | string | ✅ | 显示名称（如 `闲鱼接入器`） |
| `version` | string | ✅ | 版本号（如 `1.0.0`） |
| `runtime_type` | string | ✅ | 拉起方式：`python`（桥式，默认）或 `standalone`（独立式，命令驱动）。`node` / `go` 只是语言标签。见 [接入器运行时](./adapter-runtime.md) |
| `runtime` | map | 仅 standalone | 独立式的拉起参数：`interpreter`、`entry`（必填）、`args` |
| `description` | string | 否 | 接入器描述 |
| `hidden` | bool | 否 | 是否在 WebUI 隐藏（`false` 显示，`true` 隐藏但仍可用） |
| `icon` | string | 否 | 显示图标（emoji 或路径） |
| `config_schema` | map | 否 | 实例配置字段定义（创建实例时渲染的表单） |
| `capabilities` | list | 否 | 能力声明（如 `receive_message`、`send_text`） |

### config_schema 字段

`config_schema` 的每个键是一个配置项，值包含：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `label` | string | ✅ | 表单显示名称（如 `闲鱼 Cookie`） |
| `type` | string | ✅ | 输入类型：`text` / `password` / `number` |
| `required` | bool | 否 | 是否必填 |
| `placeholder` | string | 否 | 输入框占位提示 |
| `help` | string | 否 | 帮助说明文字 |
| `default` | any | 否 | 默认值 |

### 保留配置键

下面两个键由**运行器**读取（不是接入器脚本自己读），接入器作者只管在 `config_schema`
里把它们声明出来让用户在表单里填，不要自己解析：

| 键 | 说明 |
|----|------|
| `proxy` | 该实例的出网代理。`http://` / `https://` / `socks5://`，或对象 `{scheme,host,port,username,password}`，或 `none` 强制直连。留空＝继承后端代理。 |
| `no_proxy` | 不走代理的地址，逗号分隔，默认 `localhost,127.0.0.1,::1`。 |

运行器会把这些值转成 `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` / `NO_PROXY`（大小写
两套）注入沙箱进程的环境变量，所以用 `requests`、`aiohttp`、`websockets` 写的接入器
不需要改代码就能走代理。行为细节见 [适配器配置](../user-guide/adapters.md)。

### runtime（独立式）

`runtime_type: standalone` 时必填，告诉后端拉哪个入口：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `entry` | string | ✅ | 入口脚本，必须在接入器目录内（如 `main.mjs`） |
| `interpreter` | string | 否 | 解释器，省略时用 `adapter.node_bin`（默认 `node`） |
| `args` | list | 否 | 追加的字面参数 |

独立式的线协议、起动参数与实例生命周期见 [接入器运行时](./adapter-runtime.md)。

## 完整示例

```yaml
# adapters/xianyu/adapter.yaml
id: xianyu-adapter
platform_code: xianyu
name: 闲鱼接入器
version: 1.0.0
runtime_type: python
description: 支持闲鱼平台消息收发，多开多账号
hidden: false
icon: "🐟"

config_schema:
  cookie:
    label: 闲鱼 Cookie
    type: password
    required: true
    placeholder: 登录闲鱼后从浏览器复制完整 Cookie
    help: 每个实例独立的登录 Cookie
  device_id:
    label: Device ID
    type: text
    required: false
    placeholder: 可选，留空自动生成
  heartbeat_interval:
    label: 心跳间隔（秒）
    type: number
    required: false
    default: 15

capabilities:
  - receive_message
  - send_text
  - send_image
```

## 使用方式

1. 在 `adapters/` 下新建目录，放入 `adapter.yaml` 和接入器代码
2. 重启后端（或调用目录重扫），WebUI「接入器管理」自动显示该接入器
3. 在「实例管理」为该接入器创建实例，表单字段由 `config_schema` 动态渲染
4. 实例启动时，后端根据 `id` 在 catalog 中查得 `platform_code`，定位 `adapters/<platform_code>/main.py` 拉起进程

> 隐藏接入器：将 `hidden: true`，WebUI 不显示但仍可通过 API 使用。
