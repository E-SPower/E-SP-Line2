# 猫巴士独立式接入器

`runtime_type: standalone` 的接入器：进程由后端拉起，命令走 stdin、结果走 stdout，
不回连 WebSocket。命令词表与 JSON 信封都来自 [catbus](https://github.com/cv-cat/catbus)，
本目录只做「桥命令 / 命令词 ↔ catbus 命令行」的翻译，不复制它的逻辑。

- 依赖：Node 22+ 与 `catbus`（`npm i -g catbus-cli`）或一个直接可跑的 catbus 入口。
- 登录：`catbus xianyu auth login`（登录态按 `catbus_home` 隔离）。
- 自检：`node main.mjs --self-check --config <file>`，实例创建时由后端调用。
- 线协议：见 `docs/developer-guide/adapter-runtime.md`。
- 日志：与 xianyu/taobao 相同的 ESPL 横幅与 `时间 | 级别 | [tag] 消息` 单行格式，全部走 stderr。

当前映射的桥命令：`send_text`（→ `catbus xianyu msg send <text> --to <toid>`）、
`send_image`（→ `msg send --image <url> --to <toid>`）；读取类命令走
`command_type: cli` + `payload.args` 透传。没映射的 `command_type` 回 `UNSUPPORTED`，
不会静默丢弃。
