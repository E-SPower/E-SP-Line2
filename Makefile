.PHONY: build run test clean deps fmt lint docs migrate dev \
        build-frontend build-frontend-dev install-web \
        build-linux build-linux-arm64 build-windows build-windows-arm64 \
        build-all package release \
        build-desktop run-desktop pc-shim \
        sync-safe sync-adapters sync-webui

# ---------------------------------------------------------------------------
# 通用变量
# ---------------------------------------------------------------------------
BIN_NAME   := e-sp-line2
OUT_DIR    := dist
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

# 本机架构（CGO 依赖本地 C 编译器，交叉编译 Linux 需额外工具链）
HOST_ARCH  := $(shell go env GOHOSTARCH)
LINUX_ARCH ?= $(HOST_ARCH)
WIN_ARCH   ?= amd64

# 跨平台交叉编译使用的 mingw C 编译器（Windows 目标必需，因为 SQLite 依赖 CGO）
CC_WIN_AMD64 ?= x86_64-w64-mingw32-gcc
CC_WIN_ARM64 ?= aarch64-w64-mingw32-gcc

# ---------------------------------------------------------------------------
# 开发
# ---------------------------------------------------------------------------
# 构建项目（本机平台）。先同步 adapters 以便内嵌进二进制。
build: sync-adapters
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN_NAME) main.go

# 运行项目
run:
	go run main.go

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -rf bin/
	rm -rf $(OUT_DIR)/
	rm -rf data/*.db

# 安装依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 生成 API 文档
docs:
	swag init

# 数据库迁移
migrate:
	go run main.go migrate

# 开发模式运行
dev:
	go run main.go --dev

# ---------------------------------------------------------------------------
# 前端
# ---------------------------------------------------------------------------
# 安装前端依赖（优先 pnpm，回退 npm）
install-web:
	@cd web && (command -v pnpm >/dev/null 2>&1 && pnpm install || npm install)

# 构建前端静态产物 -> web/dist
build-frontend:
	@cd web && (command -v pnpm >/dev/null 2>&1 && pnpm run build || npm run build)
	@test -f web/dist/index.html || (echo "前端构建失败: 缺少 web/dist/index.html" >&2; exit 1)

# 前端开发服务器 (http://localhost:3000)
build-frontend-dev:
	cd web && (command -v pnpm >/dev/null 2>&1 && pnpm dev || npm run dev)

# ---------------------------------------------------------------------------
# 跨平台发布构建（默认产出到 dist/<os>-<arch>/）
# ---------------------------------------------------------------------------
# 本机 Linux 架构（默认跟随 HOSTARCH，可用 LINUX_ARCH 覆盖）
build-linux: sync-adapters
	@mkdir -p $(OUT_DIR)/linux-$(LINUX_ARCH)
	CGO_ENABLED=1 GOOS=linux GOARCH=$(LINUX_ARCH) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/linux-$(LINUX_ARCH)/$(BIN_NAME) .

build-linux-arm64:
	@mkdir -p $(OUT_DIR)/linux-arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/linux-arm64/$(BIN_NAME) .

build-windows: sync-adapters
	@mkdir -p $(OUT_DIR)/windows-$(WIN_ARCH)
	CGO_ENABLED=1 GOOS=windows GOARCH=$(WIN_ARCH) CC=$(CC_WIN_AMD64) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/windows-$(WIN_ARCH)/$(BIN_NAME).exe .

build-windows-arm64:
	@mkdir -p $(OUT_DIR)/windows-arm64
	CGO_ENABLED=1 GOOS=windows GOARCH=arm64 CC=$(CC_WIN_ARM64) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/windows-arm64/$(BIN_NAME).exe .

# 一键：Linux + Windows 全部构建（含前端）
build-all: build-frontend build-linux build-windows
	@echo "构建完成，产物位于 $(OUT_DIR)/"

# 发布打包（含前端/配置/迁移/接入器/文档），详见 scripts/build.sh
package:
	bash scripts/build.sh --target all --version $(VERSION)

# release 为 package 的别名
release: package

# ---------------------------------------------------------------------------
# 桌面窗口版（内嵌前端 + 原生 WebView，不依赖浏览器）
# ---------------------------------------------------------------------------
# 依赖: Linux 需 WebKitGTK / GTK 开发库
#   sudo apt install -y libwebkit2gtk-4.1-dev libgtk-3-dev
# webview_go 的 cgo 指令要求 webkit2gtk-4.0；新发行版仅提供 4.1，
# 故通过 pc-shim 生成 .pc 映射并注入 PKG_CONFIG_PATH。
pc-shim:
	@mkdir -p scripts/pkgconfig
	@for base in webkit2gtk javascriptcoregtk; do \
	  if pkg-config --exists $$base-4.1 2>/dev/null; then \
	    d=$$(pkg-config --variable=pcfiledir $$base-4.1); \
	    cp -f $$d/$$base-4.1.pc scripts/pkgconfig/$$base-4.0.pc 2>/dev/null || true; \
	  fi; \
	done

# 将前端产物同步到 pkg/webui/dist（供 desktop 构建 go:embed 内嵌）
sync-webui:
	@rm -rf pkg/webui/dist
	@mkdir -p pkg/webui/dist
	@cp -R web/dist/. pkg/webui/dist/
	@test -f pkg/webui/dist/index.html || (echo "缺少 pkg/webui/dist/index.html" >&2; exit 1)
	@echo "前端已同步 -> pkg/webui/dist"

# 将 adapters/ 同步到 internal/adapters/adapters（供 //go:embed 内嵌，
# 运行时由 internal/adapters 释放到 data/adapters/）
sync-adapters:
	@test -d adapters || (echo "缺少 adapters/ 目录" >&2; exit 1)
	@mkdir -p internal/adapters/adapters
	@find internal/adapters/adapters -mindepth 1 -maxdepth 1 ! -name 'README.md' -exec rm -rf {} + 2>/dev/null || true
	@cp -R adapters/. internal/adapters/adapters/
	@find internal/adapters/adapters -name '__pycache__' -type d -prune -exec rm -rf {} + 2>/dev/null || true
	@find internal/adapters/adapters -name '*.pyc' -delete 2>/dev/null || true
	@n=$$(find internal/adapters/adapters -name adapter.yaml | wc -l | tr -d ' '); \
	  test "$$n" -gt 0 || (echo "同步失败: 未找到 adapter.yaml" >&2; exit 1); \
	  echo "adapters 已内嵌同步（$$n 个接入器）-> internal/adapters/adapters"

# 编译前的全部同步步骤
sync-safe: sync-adapters

# 构建桌面窗口版（含前端构建 + 内嵌同步）
build-desktop: build-frontend sync-webui sync-adapters pc-shim
	@mkdir -p $(OUT_DIR)/desktop-linux-$(LINUX_ARCH)
	PKG_CONFIG_PATH="$(CURDIR)/scripts/pkgconfig:$$PKG_CONFIG_PATH" \
	CGO_ENABLED=1 GOOS=linux GOARCH=$(LINUX_ARCH) \
		go build -trimpath -tags desktop -ldflags "$(LDFLAGS)" \
		-o $(OUT_DIR)/desktop-linux-$(LINUX_ARCH)/$(BIN_NAME) ./cmd/esp-desktop
	@echo "桌面版构建完成 -> $(OUT_DIR)/desktop-linux-$(LINUX_ARCH)/$(BIN_NAME)"

# 本地直接运行桌面窗口版（开发调试）
run-desktop: pc-shim
	@test -f pkg/webui/dist/index.html || (echo "请先执行 make build-frontend 并同步 pkg/webui/dist" >&2; exit 1)
	PKG_CONFIG_PATH="$(CURDIR)/scripts/pkgconfig:$$PKG_CONFIG_PATH" \
		go run -tags desktop ./cmd/esp-desktop
