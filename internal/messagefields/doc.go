// Package messagefields 定义入站消息、出站命令和归一化对象的字段契约。
//
// 迁移自 catbus（MIT）的 src/core/schemas.ts 与 AGENTS.md 6.2「归一化类型」：
// 同一种对象在所有平台上结构相同，字段总是存在，取不到时为 null 或 []，计数是整数或 null，
// 时间是带时区的 ISO 8601，ID 一律是字符串（64 位整数 ID 不丢精度），所有对象都带 id 和 url。
//
// 文档正本是 docs/developer-guide/message-fields.md，测试会检查这里的定义与文档一致。
package messagefields
