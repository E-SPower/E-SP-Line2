# 接入器运行时（runtime_type）

`adapter.yaml` 的 `runtime_type` 决定后端**怎么拉起**接入器，不是备注字段。
两种运行时的线协议完全不同：

| `runtime_type` | 拉起方式 | 通信 |
|---|---|---|
| `python`（默认） | `python main.py --instance-id X --backend URL --token <jwt>` | 进程自己回连 `ws://…/ws/adapter?instance_id=X`（桥式） |
| `standalone` | `<interpreter> <runtime.entry> --instance-id X --backend URL --config <file>` | 后端把命令写一行 JSON 到 stdin，进程把一行 JSON 信封写回 stdout |

`node` / `go` 仍是标签：接入器确实用该语言写、但仍按桥式拉起时用它。
只有 `python` 与 `standalone` 有拉起分支，写别的值会在 catalog 扫描时报错。

## 独立式（standalone）

独立式不建立桥回连，没有 WebSocket、没有握手、没有重连；进程本身就是接入器，
由后端按命令驱动。适合一个现成的 CLI（例如 catbus）：它已经有完整的命令词表，
独立式只负责把桥命令翻成命令行、把它的 JSON 信封原样带回。

```yaml
runtime_type: standalone
runtime:
  interpreter: node     # 省略时用 adapter.node_bin，默认 node
  entry: main.mjs       # 必填，必须在接入器目录内
  args: []              # 可选，追加的字面参数
```

### 线协议

stdin 一行一个 JSON 命令，stdout 一行一个 JSON 信封。stdout 只放信封，
日志一律走 stderr（否则会污染命令通道）。

```jsonc
// 后端 -> 进程
{"id":"<调用方给的 id>","command_type":"send_text","payload":{"cid":"…","toid":"…","text":"…"}}
// 进程 -> 后端，id 原样带回
{"id":"…","ok":true,"data":{…},"page":null,"error":null}
```

- 桥命令 `send_text` / `send_image` 由接入器翻译成自己的命令行；没有映射的
  `command_type` 回 `{"ok":false,"error":{"code":"UNSUPPORTED",…}}`。
- `command_type: "cli"` 带 `payload.args`（拼好的 CLI 词表）直接透传，
  用于读取类命令。
- 信封字段照 ESPL v3 / catbus 的 `ok / data / page / error{code,message,hint,detail}`，
  不得自造协议。

### 起动参数

运行器固定追加，接入器按需读：

| 参数 / 环境变量 | 说明 |
|---|---|
| `--instance-id` | 实例 ID |
| `--backend` | 后端基地址（`ESP_BACKEND_URL` 同义） |
| `--token` | 服务 JWT（后端调管理 API 用） |
| `--config <path>` | 实例配置 JSON（`<instance_dir>/config.runtime.json`，0600，**不在**沙箱副本里，所以不影响完整性校验） |
| `--self-check` | 只在实例创建时调用一次，自检必需依赖 |
| `ESP_ADAPTER_RUNTIME=standalone` | 标明当前运行时 |

### 实例生命周期

1. 创建实例时，沙箱从 `adapter.yaml` 所在目录复制（不是 `platform_code` 指向的
   目录），随后跑一次 `--self-check`；
2. 自检非 0 退出 → 实例停在 `error`，日志里是自检输出；自检通过 → `stopped`；
3. 启动后直接 `running`，不再等 WebSocket 握手；
4. 出站指令由网关投递给运行器、写进进程 stdin（`POST /api/v1/instances/:id/command`
   也可以直接下发一条，便于联调）；
5. `SIGTERM` 为正常停止。

### 写一个独立式接入器

- 读 `--config` 拿实例配置；把 `--self-check` 实现成真实检查（必需依赖不存在时
  退出码 1），并在实例日志里用 `时间 | 级别 | [tag] 消息` 单行格式写清楚；
- 启动时打印与容器相同的 ESPL 艺术字横幅（见 `adapters/catbus/main.mjs`）；
- stdout 只写信封，且每行一个、立即 flush；
- 命令失败也用信封回报，别只写 stderr。
