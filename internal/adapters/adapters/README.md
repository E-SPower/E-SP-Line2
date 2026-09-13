# 内嵌适配器目录（构建输入）

此目录是 `//go:embed all:adapters` 的构建输入，**不要手动编辑**。

构建脚本（`scripts/build.sh` / `scripts/build.ps1`）会在编译前把仓库根目录的
`adapters/` 同步到这里，然后随二进制一起编译进去。运行时由
`internal/adapters.EnsureExtracted` 释放到 `data/adapters/`。

- 占位文件仅用于保证在未同步时 `go build ./...` 依然可以编译通过。
- 当这里只有本 README 时，`internal/adapters.Available()` 返回 false，
  程序会退回到使用外部 `adapters/` 目录（开发模式）。
