// Package platformspec 保存从 catbus 迁移过来的平台注册表。
//
// catbus（MIT）把命令解析、`platforms` 命令和 docs/capabilities.md 能力矩阵都从同一份
// 平台注册表生成。这里保留同样的单一数据源：加平台、改能力都只动 platformspec，
// 命令树、能力矩阵和 manifest 都从它派生。
//
// 源仓库：https://github.com/cv-cat/catbus （MIT，见 NOTICE-catbus.md）。
// 重新生成见 scripts/gen_platformspec.py。
package platformspec
