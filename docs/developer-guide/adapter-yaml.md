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
| `runtime_type` | string | ✅ | 运行时类型：`python` / `node` / `go` |
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
