# E-SP-Line2 安装与编译指南

本指南说明如何在 **Linux** 与 **Windows** 上安装依赖、编译并打包 E-SP-Line2。

---

## 1. 架构与编译难点

E-SP-Line2 由三部分组成，编译方式各不相同：

| 组件 | 技术栈 | 产物 | 是否需编译 |
| --- | --- | --- | --- |
| 后端 | Go 1.22 + Gin + GORM | 单文件可执行程序 | 是（见下） |
| 前端 | React + Vite + TypeScript | 纯静态 `web/dist` | 是（Node 构建） |
| Python 接入器 | Python 3 + Node(JS 执行) | 无需编译 | 否 |

### ⚠ 关键：CGO 与 SQLite

后端默认使用 SQLite（[`gorm.io/driver/sqlite`](../go.mod) → 底层 `mattn/go-sqlite3`），
这是一个 **CGO** 库，因此：

- **必须 `CGO_ENABLED=1`**，本机需要有 C 编译器（gcc / clang / MSVC）。
- **跨平台编译 Windows 版本时，Linux 上必须安装 mingw-w64 交叉编译器。**
  （纯 `GOOS=windows` 而不带 `CC` 会因缺少 C 编译器而失败。）

> 若你不需要 SQLite，可改用 PostgreSQL 驱动（纯 Go，无 CGO），
> 即可用 `CGO_ENABLED=0` 做纯 Go 交叉编译。默认配置仍为 SQLite。

### ⚠ 前端未被后端内嵌

后端代码中**没有** `go:embed` / `Static` 路由（已全仓库确认），
`web/dist` 是独立静态产物，生产环境需：

- 用 Nginx / Caddy / 静态服务器托管 `web/dist`；
- 将 `/api`、`/health`、`/ws` 反向代理到后端 `:8080`。

---

## 2. 依赖清单

| 依赖 | 版本 | 用途 |
| --- | --- | --- |
| Go | 1.22+ | 编译后端 |
| Node.js | 18+ | 编译前端 |
| pnpm 或 npm | 任意较新版本 | 前端包管理 |
| C 编译器 | gcc / clang / MSVC | CGO (SQLite) |
| mingw-w64 | 最新 | **Linux 交叉编译 Windows 版本必需** |
| Python | 3.9+ | 运行接入器（非编译期必需） |

---

## 3. Linux 安装

### 3.1 Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y build-essential git curl

# Go（若未安装，示例 1.22；请按需升级）
# 也可直接使用系统包：sudo apt install -y golang-go

# Node.js 18+（推荐 NodeSource）
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# 前端包管理器（可选）
sudo npm install -g pnpm

# 交叉编译 Windows 必需
sudo apt install -y gcc-mingw-w64 zip
```

### 3.2 Arch / Manjaro

```bash
sudo pacman -S --needed base-devel git go nodejs npm zip mingw-w64-gcc
# 可选：sudo pacman -S pnpm
```

### 3.3 Fedora / RHEL

```bash
sudo dnf install -y @development-tools git golang nodejs npm zip mingw64-gcc
```

### 3.4 macOS（仅本机编译，不产出 Windows）

```bash
brew install go node pnpm
# 如需交叉编译 Windows：brew install mingw-w64
```

---

## 4. Windows 安装

### 4.1 安装 Go

从 <https://go.dev/dl/> 下载 `go1.22.x.windows-amd64.msi` 安装，安装后重启终端：

```powershell
go version
```

### 4.2 安装 Node.js

从 <https://nodejs.org/> 下载 LTS 版本安装：

```powershell
node -v
npm -v
# 可选
npm install -g pnpm
```

### 4.3 安装 CGO 编译器（二选一）

**方案 A：MSVC（推荐）**

安装 *Visual Studio Build Tools*，勾选「使用 C++ 的桌面开发」。
构建时从 **x64 Native Tools Command Prompt for VS** 运行，或先执行：

```bat
call "C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvars64.bat"
```

**方案 B：mingw-w64**

```powershell
winget install -e --id MSYS2.MSYS2
# 然后打开 MSYS2 终端执行：
# pacman -S mingw-w64-x86_64-gcc
# 并把 C:\msys64\mingw64\bin 加入 PATH
```

### 4.4 安装 Git（可选，用于版本号）

从 <https://git-scm.com/download/win> 安装。

---

## 5. 编译方式

### 方式 A：一键脚本（推荐）

**Linux（同时产出 Linux + Windows）：**

```bash
cd E-SP-Line2
chmod +x scripts/build.sh
./scripts/build.sh --target all          # Linux amd64 + Windows amd64
./scripts/build.sh --target windows      # 仅 Windows
./scripts/build.sh --target linux        # 仅 Linux
./scripts/build.sh --target windows --arch arm64
./scripts/build.sh --frontend-only       # 只构建前端
./scripts/build.sh --backend-only        # 只构建后端
./scripts/build.sh --no-lint             # 跳过前端 lint
```

**Windows（使用 PowerShell 原生脚本）：**

```powershell
cd E-SP-Line2
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Arch arm64
powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -FrontendOnly
```

产物统一输出到 `dist/`：

```
dist/
├── linux-amd64/e-sp-line2
├── windows-amd64/e-sp-line2.exe
├── e-sp-line2-<ver>-linux-amd64.tar.gz
└── e-sp-line2-<ver>-windows-amd64.zip
```

### 方式 B：Makefile

```bash
make deps                # 下载 Go 依赖
make install-web         # 安装前端依赖
make build-frontend      # 构建前端 -> web/dist
make build-linux         # Linux amd64
make build-linux-arm64   # Linux arm64
make build-windows       # Windows amd64（需 mingw）
make build-all           # 前端 + Linux + Windows
make package             # 完整打包（调用 scripts/build.sh）
```

### 方式 C：手动命令

**Linux 本机：**

```bash
go build -trimpath -ldflags "-s -w" -o bin/e-sp-line2 main.go
```

**Linux → Windows 交叉编译（关键：指定 CGO 与 CC）：**

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc \
  go build -trimpath -ldflags "-s -w" -o bin/e-sp-line2.exe .
```

**Windows 本机（PowerShell）：**

```powershell
$env:CGO_ENABLED="1"; $env:GOOS="windows"; $env:GOARCH="amd64"
go build -trimpath -ldflags "-s -w" -o bin\e-sp-line2.exe .
```

**前端：**

```bash
cd web
pnpm install          # 或 npm install
pnpm run build        # 或 npm run build -> 生成 web/dist
```

---

## 6. 版本号注入

构建脚本会自动注入版本与编译时间（在 [`main.go`](../main.go) 中的 `version` / `buildTime`）：

```bash
-ldflags "-X main.version=$(git describe --tags --always) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

手动指定版本：

```bash
./scripts/build.sh --target all --version 1.0.0
```

---

## 7. 运行部署

### 7.1 后端

```bash
# 确保 config/config.yaml 存在（可复制示例修改）
./e-sp-line2            # Linux
e-sp-line2.exe          # Windows
```

默认监听 `0.0.0.0:8080`，SQLite 数据库自动生成于 `data/e-sp-line2.db`。

### 7.2 前端 + 反向代理

**Nginx（Linux）示例：**

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /opt/e-sp-line2/web/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;   # SPA 路由回退
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    location /health { proxy_pass http://127.0.0.1:8080; }
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

**Windows（IIS 或直接用 Caddy）：**

```
# Caddyfile
:80 {
    root * C:\e-sp-line2\web\dist
    try_files {path} /index.html
    reverse_proxy /api/* 127.0.0.1:8080
    reverse_proxy /ws/*  127.0.0.1:8080
}
```

### 7.3 Python 接入器运行时（重要）

**接入器源码已内嵌进二进制**（运行时自动释放），但 **Python 解释器与 pip 依赖不随包分发**。

| 组件 | 是否随二进制 | 说明 |
| --- | --- | --- |
| 前端 WebUI | ✅ 会（仅桌面版，`go:embed`） | 编译进二进制 |
| Go 后端 | ✅ 会 | 就是二进制本身 |
| **接入器源码** | ✅ **会**（`go:embed`） | 运行时释放到 `data/adapters/` |
| **Python 运行时** | ❌ **不会** | 需目标机安装解释器 |
| **接入器 pip 依赖** | ❌ **不会** | 需目标机安装（或用安装脚本） |
| Node.js（PyExecJS 签名） | ❌ 不会 | 需目标机安装 |

#### 内嵌适配器机制（新增）

```
构建期:  adapters/  ──同步──▶  internal/adapters/adapters/  ──go:embed──▶  二进制
运行期:  二进制  ──释放──▶  data/adapters/<platform>/  ──拷贝──▶  data/instances/<id>/adapter/
```

- **发布包不再包含外部 `adapters/` 目录**（保持单文件分发，已验证包内 adapters 条目数为 0）。
- 首次启动自动释放到 `data/adapters/`，并用内容指纹（`.embedded-stamp`）判断是否需重释；
  指纹未变时**完全不写磁盘**，正常启动零开销。
- **开发覆盖**：若二进制同目录存在真实 `adapters/`（含 `adapter.yaml`），则**优先使用它**，
  便于不改代码直接调试接入器。
- 如需随包附带一份可编辑副本：`INCLUDE_ADAPTERS=1 ./scripts/build.sh ...`
  （PowerShell：`-IncludeAdapters`）。

也就是说：**换一台机器部署时，必须安装 Python 3.9+ 与接入器依赖**（接入器代码本身已内嵌，无需拷贝）。

> 项目内置的 [`dependency_installer.go`](../internal/service/dependency_installer.go) 会在
> WebUI 创建实例时**自动尝试安装**依赖（级联 `python -m pip` → `pip3` → `pip` → `pipx`，
> 含 PEP 668 `--break-system-packages` 重试与逐包降级），但前提是目标机**已有 Python 与 pip**。

#### 方式一：使用安装脚本（推荐）

```bash
# Linux / macOS
./scripts/install-python-deps.sh          # 安装到当前 Python 环境
./scripts/install-python-deps.sh --venv   # 创建隔离的 .venv 并安装
./scripts/install-python-deps.sh --check  # 仅自检，不安装
```

```powershell
# Windows
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Venv
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Check
```

发布包中已内置这两个脚本（`scripts/` 目录）。

#### 方式二：手动安装

```bash
pip install -r adapters/taobao/requirements.txt
pip install -r adapters/xianyu/requirements.txt
```

使用虚拟环境时，在 `config/config.yaml` 指定解释器：

```yaml
adapter:
  python_bin: "/path/to/.venv/bin/python"   # Windows: C:\path\to\.venv\Scripts\python.exe
```

#### 依赖清单与风险

| 包 | 用途 | 备注 |
| --- | --- | --- |
| requests, websockets | HTTP/WS | 必备 |
| loguru | 日志 | 必备 |
| pydantic | 数据模型 | 必备 |
| PyExecJS | 执行签名 JS | **需 Node.js 18+** |
| blackboxprotobuf | Protobuf 解密 | 淘宝 |
| qrcode, backoff | 二维码登录/重试 | 可选 |

> ⚠ **Python 版本注意**：`PyExecJS` 在较新的 Python（如 3.13/3.14）上可能安装或运行异常。
> 安装器已实现「整体失败 → 逐包跳过」的降级策略。若签名功能异常，
> 建议使用 **Python 3.10 ~ 3.12** 虚拟环境并指定 `adapter.python_bin`。

> ⚠ **Node.js 是硬依赖**：接入器的签名计算依赖 PyExecJS 调用 Node，缺失时接入器无法登录平台。

---

## 7.5 桌面窗口版（编译后即窗口，不走浏览器）

需求区分两种运行形态：

| 形态 | 后端 | 前端 | 用户体验 |
| --- | --- | --- | --- |
| **网页版**（默认，不编译直接运行） | `go run main.go` / 默认二进制 | 外部 `web/dist` + 反代 | 浏览器访问 `http://host:port` |
| **桌面窗口版**（编译后） | 带 `desktop` 标签，内嵌前端 | 通过 `go:embed` 打进二进制 | 直接弹出原生窗口，无需浏览器 |

### 实现原理

- 前端产物被 `//go:embed all:dist` 内嵌进二进制（[`pkg/webui/webui_desktop.go`](../pkg/webui/webui_desktop.go)）；
- 桌面入口 [`cmd/esp-desktop/main.go`](../cmd/esp-desktop/main.go) 启动同一个 API 服务（仅绑定 `127.0.0.1`），再打开原生 WebView 指向它；
- WebView 由 [`github.com/webview/webview_go`](https://github.com/webview/webview_go) 提供：
  - **Windows**：使用系统 **WebView2**（Win10/11 通常已内置；若无请安装 [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)）
  - **Linux**：使用 **WebKitGTK**
- 后端通过 `NoRoute` 托管内嵌 SPA（[`internal/server/server.go`](../internal/server/server.go)），深链自动回退 `index.html`；
- **默认（网页版）构建完全不含这些代码**，体积与行为不受影响。

### Linux 桌面版

前置依赖（编译期，CGO）：

```bash
sudo apt install -y libwebkit2gtk-4.1-dev libgtk-3-dev build-essential
```

构建：

```bash
./scripts/build.sh --desktop --target linux     # 或
make build-desktop                              # 或
make run-desktop                                # 开发时直接运行窗口
```

**图形化启动（不弹终端）**：发布包内含 `e-sp-line2.desktop`（`Terminal=false`）。
双击它或在应用菜单中启动，即为纯窗口体验：

```bash
# 复制到用户应用目录后即可在应用菜单中找到
cp e-sp-line2.desktop ~/.local/share/applications/
```

> **Linux 没有"GUI 子系统"概念**：窗口程序就是一个会打开 X11/Wayland 窗口的普通可执行文件。
> 是否出现终端取决于启动方式：
> - 通过 `.desktop`（`Terminal=false`）或 `run.sh`（后台启动）→ **无终端窗口**；
> - 直接在终端执行 `./e-sp-line2` → 保留终端便于看日志。
>
> 桌面版始终把日志镜像写入 `data/desktop.log`，即使无终端也能事后排查。

> **webkit2gtk-4.0 vs 4.1**：`webview_go` 的 cgo 指令写死了 `webkit2gtk-4.0`，
> 而 Ubuntu 24.04+/Arch 只提供 `webkit2gtk-4.1`（ABI 重命名，C API 兼容）。
> 构建脚本会自动在 `scripts/pkgconfig/` 生成 `webkit2gtk-4.0.pc`（复制自 4.1）
> 并注入 `PKG_CONFIG_PATH`，**无需修改任何第三方源码**。

产物：`dist/desktop-linux-<arch>/e-sp-line2`（内嵌前端，单文件即可运行）。

### Windows 桌面版

前置条件：
- **CGO 编译器**（见 §4.3：MSVC 或 mingw-w64）
- **WebView2 Runtime**（Win11 默认内置；Win10 若缺失请安装）

构建（PowerShell）：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Desktop
powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Desktop -Arch arm64
```

产物：`dist\desktop-windows-<arch>\e-sp-line2.exe`（**GUI 子系统**，双击即出窗口，含内嵌前端）。

> **为什么双击不再弹黑框**：桌面版以 `-ldflags "-H=windowsgui"` 链接，
> PE 子系统为 `Windows GUI`（网页版保持 `Windows CUI` 以便看日志）。
> 若需在终端查看输出，本程序会自动附加到父控制台；否则日志写入 `data/desktop.log`。

### 手动编译（等价命令）

```bash
# 1) 先构建前端
make build-frontend
rm -rf pkg/webui/dist && mkdir -p pkg/webui/dist && cp -R web/dist/. pkg/webui/dist/

# 2) Linux 桌面版
PKG_CONFIG_PATH="$PWD/scripts/pkgconfig:$PKG_CONFIG_PATH" \
CGO_ENABLED=1 go build -tags desktop -o bin/e-sp-line2-desktop ./cmd/esp-desktop

# 3) Windows 桌面版（在 Windows 上）
go build -tags desktop -o bin\e-sp-line2-desktop.exe .\cmd\esp-desktop
```

> 注意：**必须带 `-tags desktop`**，否则编译出的是“提示需要 desktop 标签”的占位程序。

### Python 接入器运行时（重要）

**Python 运行时不会被打包进二进制**，与前端不同。原因：

| 组件 | 是否随二进制分发 | 说明 |
| --- | --- | --- |
| 前端 WebUI | ✅ 会（仅桌面版，`go:embed`） | 编译进二进制，窗口内即前端界面 |
| Go 后端 | ✅ 会 | 就是二进制本身 |
| **Python 运行时** | ❌ **不会** | 需目标机自行安装解释器 |
| **接入器 Python 依赖** | ❌ **不会** | pip 包，需目标机安装 |
| 接入器源码 | ✅ 会（`adapters/` 目录） | 随发布包分发 |
| Node.js（PyExecJS 签名） | ❌ 不会 | 需目标机安装 |

也就是说：**换一台机器部署时，必须安装 Python 3.9+ 与接入器依赖**，
否则 WebUI 中启动接入器会失败。

> 项目内置的 [`dependency_installer.go`](../internal/service/dependency_installer.go) 会在
> WebUI 创建实例时**自动尝试安装**依赖（级联 `python -m pip` → `pip3` → `pip` → `pipx`，
> 含 PEP 668 `--break-system-packages` 重试与逐包降级），但前提是目标机**已有 Python 与 pip**。

#### 方式一：一键安装脚本（推荐，**缺什么装什么**）

脚本会**自动检测并安装** Python、pip、Node.js（若宿主机缺失），再安装接入器依赖并自检。

```bash
# Linux / macOS
./scripts/install-python-deps.sh                # 缺什么装什么（自动装 Python/Node）
./scripts/install-python-deps.sh --venv         # 用隔离虚拟环境 .venv 安装
./scripts/install-python-deps.sh --check        # 仅检查，不做任何安装
./scripts/install-python-deps.sh --no-auto      # 只装 pip 依赖，不装系统包
./scripts/install-python-deps.sh -y             # 自动确认，免交互
./scripts/install-python-deps.sh --python /usr/bin/python3
```

```powershell
# Windows
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Venv
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Check
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -NoAuto
powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Yes
```

**自动安装能力矩阵**（无需手动预装任何东西）：

| 宿主机缺失 | Linux 行为 | Windows 行为 |
| --- | --- | --- |
| Python 3.9+ | 调 `apt/dnf/yum/pacman/zypper/apk/brew` 安装 `python3 python3-pip python3-venv` | `winget install Python.Python.3.12`；无 winget 则下载 python.org 官方安装包静默安装 |
| pip | 随 Python 包一并安装；退化尝试 `ensurepip` | `ensurepip --upgrade` |
| Node.js | 调包管理器安装 `nodejs npm` | `winget install OpenJS.NodeJS.LTS`；无 winget 则下载 nodejs.org `.msi` 静默安装 |
| venv 模块 | 自动补装 `python3-venv` 后重试 | 通常内置 |

> 需要管理员权限时脚本会自动调用 `sudo`（Linux）/ 提示以管理员重跑（Windows）。
> 发布包中已内置这两个脚本（`scripts/` 目录）。

#### 方式二：手动安装

```bash
pip install -r adapters/taobao/requirements.txt
pip install -r adapters/xianyu/requirements.txt
```

使用虚拟环境时，在 `config/config.yaml` 指定解释器：

```yaml
adapter:
  python_bin: "/path/to/.venv/bin/python"   # Windows: C:\path\to\.venv\Scripts\python.exe
```

#### 依赖清单与风险

| 包 | 用途 | 备注 |
| --- | --- | --- |
| requests, websockets | HTTP/WS | 必备 |
| loguru | 日志 | 必备 |
| pydantic | 数据模型 | 必备 |
| PyExecJS | 执行签名 JS | **需 Node.js 18+** |
| blackboxprotobuf | Protobuf 解密 | 淘宝 |
| qrcode, backoff | 二维码登录/重试 | 可选 |

> ⚠ **Python 版本注意**：`PyExecJS` 在较新的 Python（如 3.13/3.14）上可能安装或运行异常。
> 安装器已实现「整体失败 → 逐包跳过」的降级策略。若签名功能异常，
> 建议使用 **Python 3.10 ~ 3.12** 虚拟环境并指定 `adapter.python_bin`。

> ⚠ **Node.js 是硬依赖**：接入器的签名计算依赖 PyExecJS 调用 Node，缺失时接入器无法登录平台。

---

## 7.6 交叉编译 Windows 桌面版的两个坑（已自动处理）

本项目的 `scripts/build.sh` 已内置以下修复，此处记录便于手动排查：

**坑 1：`unrecognized command-line option '-m64'`**

`webview_go` 含 C++ 源文件，Go 会调用 `CXX`。若只设 `CC` 而 `CXX` 回退到宿主机 `g++`，即报此错。
脚本已自动设置 `CXX=x86_64-w64-mingw32-g++`。需先安装：

```bash
sudo apt install -y g++-mingw-w64-x86-64
```

**坑 2：`WebView2.h: EventToken.h: No such file or directory`**

`webview_go` 自带的 WebView2 SDK 头**不完整**：`WebView2.h` 引用了 `EventToken.h`，
但该文件未随模块分发（原生 Windows + MSVC 能从 Windows SDK 找到，mingw 交叉编译则失败）。
本项目在 [`scripts/mswebview2-compat/EventToken.h`](../scripts/mswebview2-compat/EventToken.h)
提供 ABI 兼容头，脚本自动通过 `CGO_CXXFLAGS` 注入，**不修改第三方源码**。

**等价手动命令：**

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc \
  CXX=x86_64-w64-mingw32-g++ \
  CGO_CXXFLAGS="-I$PWD/scripts/mswebview2-compat" \
  CGO_CFLAGS="-I$PWD/scripts/mswebview2-compat" \
  go build -trimpath -tags desktop -o bin/e-sp-line2-desktop.exe ./cmd/esp-desktop
```

---

## 7.7 Windows 双击 exe 闪退？已修复

**现象**：在 Windows 上双击 `e-sp-line2.exe`，黑框一闪就消失，看不到任何错误。

**根因**（两个叠加）：
1. Windows 双击启动时，**工作目录不保证是 exe 所在目录**（可能是 `C:\Windows\System32`）；
   而配置发现使用相对路径 `./config/config.yaml`，SQLite 使用相对路径 `data/e-sp-line2.db`。
2. **`data/` 目录不存在时 SQLite 无法创建数据库**，初始化失败 → `Fatalf` → 进程立即退出。
   （Linux 下用 `run.sh` 启动，脚本里有 `mkdir -p data`，所以从未暴露。）

**修复**（[`pkg/bootstrap`](../pkg/bootstrap/bootstrap.go)）：
- 进程启动时自动 `chdir` 到 exe 所在目录（`go run` 场景会自动跳过）；
- 自动创建 `data/`、`data/instances/`、`data/logs/`、`data/deps/`；
- 致命错误改为**弹窗提示**（Windows 用 `MessageBoxW`），不再无声退出；
  非 Windows 平台在交互式终端下会等待回车，管道/CI 下不阻塞。

因此现在双击 exe 若仍有问题，会**弹出对话框说明原因**，而不是闪退。

> 仍然建议用随包的 `run.bat` 启动，它除了预检 Python/Node，还会保留控制台窗口便于看日志。

---

## 7.8 Windows 下载后无法运行？（两个常见拦截）

### 现象 1：`Windows 安全中心 - 无法打开这些文件：你的 Internet 安全设置阻止打开一个或多个文件`

**原因**：从浏览器/网盘下载的 zip 被 Windows 打上 **Mark of the Web** 标记，
解压后每个文件都带"来自 Internet"属性，SmartScreen 会拦截可执行文件。

**解决**：解压后执行随包的解锁脚本（递归清除 Zone.Identifier）：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\unblock-windows.ps1
```

或手动：右键 zip → **属性** → 勾选 **解除锁定** → 确定，**然后**再解压。

### 现象 2：运行 `run.bat` 满屏 `'xxx' 不是内部或外部命令`

**原因**：批处理文件用了 **LF 换行**（Unix 风格）。`cmd.exe` 要求 **CRLF**，
LF-only 的 `.bat` 会被错误解析，尤其在 `if (...)` 块内会产生大量假报错。

**已修复**：构建脚本在打包时自动把所有 `.bat` / `.ps1` / `.txt` 转换为 CRLF
（构建日志会显示 `已将 N 个 Windows 文本文件转换为 CRLF`）。
同时 `run.bat` / `run-desktop.bat` 已改为**纯 ASCII**，避免代码页导致的中文乱码。

---

## 7.9 `instance sandbox is not initialized` 已可自愈

**现象**：WebUI 中启动接入器时报
`instance sandbox is not initialized; please recreate the instance`。

**原因**：接入器运行时使用 `data/instances/<id>/adapter/` 下的**副本**（沙箱），
而不是直接跑源码——这是为了防止运行时篡改源文件。沙箱在创建实例时生成，但下列情况会丢失：

- `data/` 被删除或移动（例如把 exe 拷到别处运行，工作目录变了）；
- 数据库保留但 `data/instances/` 被清理；
- 跨版本升级导致目录布局变化。

旧行为是直接报错并要求**重建实例**——这会丢掉用户已填的配置。

**现在**：启动接入器时若发现沙箱缺失或不完整（缺 `main.py` 或 `manifest.json`），
会自动从适配器源（内嵌的 `data/adapters/` 或外部 `adapters/`）**重建沙箱**并继续启动。
日志会记录：

```
Rebuilt missing instance sandbox  {"instance_id": "...", "adapter_dir": "data/instances/<id>/adapter"}
```

只有当适配器源本身也不存在时才会失败，此时错误信息会说明具体原因与排查方向。

> 相关逻辑：[`python_runner.go`](../internal/service/python_runner.go) 的 `sandboxDir()` / `sandboxUsable()`，
> 覆盖测试见 [`sandbox_heal_test.go`](../internal/service/sandbox_heal_test.go)。

---

## 7.10 创建实例时请求超时（已修复）

**现象**：WebUI 点击创建实例后，浏览器报 `Request timeout`，界面长时间无响应。

**根因**：`POST /api/v1/instances` **同步等待 pip 安装完成**。
pip 需要联网下载并编译 wheel（实测 **31 秒**），远超浏览器/反向代理的默认超时。

代码注释写的是"WebUI polls the init status"（前端轮询进度），说明**设计意图本就是异步**，
但实现是把安装直接跑在请求线程里。

**修复**：`InstallDependencies` 立即返回，实际安装在后台 goroutine 中执行，
进度仍通过 `state.json` / 实例日志暴露给前端轮询。

```
POST /instances  立即 201 返回（status=initializing）
        ↓ 后台 goroutine
    pip install ...  →  完成后 status 变为 stopped
```

同时为该 goroutine 加了 `recover()`，避免安装过程中的 panic 拖垮整个后端进程。

## 7.11 Windows 上 `python` 是 Microsoft Store 存根

**现象**：日志出现
```
Python was not found but can be installed from the Microsoft Store: https://go.microsoft.com/fwlink?linkID=2082640
python -m pip 安装失败，尝试下一方式...
```

**原因**：Windows 自带一个名为 `python.exe` 的 **App Execution Alias**（应用执行别名）。
未安装真实 Python 时，它只是一个"广告存根"，执行后打印上面的提示并非零退出。

**修复**：安装器会先探测 `python -c "print(1)"` 是否真的可执行；
若识别为 Store 存根（输出含 `microsoft store` / `was not found` / `app execution alias`），
直接跳过该安装方式，继续走 `pip3` / `pip` / `pipx`，不再产生误导性报错。

> 彻底解决：从 <https://www.python.org/downloads/> 安装 Python（勾选 *Add python.exe to PATH*），
> 或在「设置 → 应用 → 高级应用设置 → 应用执行别名」中关闭 `python.exe` 别名。

## 7.12 端口被占用时窗口仍打开（已修复）

**现象**：日志出现
```
FATAL: 服务启动失败: listen tcp :8080: bind: Only one usage of each socket address ... permitted
Desktop window opening at http://127.0.0.1:8080
```
即：监听失败弹出致命错误，**但窗口还是打开了**——而且连的是另一个进程的服务。

**根因**：监听失败发生在 goroutine 里，主协程没感知，继续 `waitReady()`。
由于另一个实例正占着 8080，`waitReady` 反而成功了，于是窗口指向了**不属于本进程的服务**。

**修复**（两点）：
1. 启动前先探测端口，被占用则**自动顺延**到下一个可用端口（最多尝试 20 个），并记录日志；
2. 监听失败通过 channel 传回主协程，`waitReadyOrError` 立即中止，不再打开窗口。

这同时解决了"双开桌面版会冲突"的问题——现在每个实例都用自己的端口。

> 覆盖测试见 [`port_test.go`](../cmd/esp-desktop/port_test.go)。

---

## 8. 常见问题

**Q1. `cgo: C compiler "x86_64-w64-mingw32-gcc" not found`**

Windows 交叉编译缺少 mingw。安装：

```bash
# Debian/Ubuntu
sudo apt install -y gcc-mingw-w64
# Arch
sudo pacman -S mingw-w64-gcc
# Fedora
sudo dnf install -y mingw64-gcc
```

**Q2. `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo`**

请设 `CGO_ENABLED=1`（脚本已默认开启）。

**Q2b. 桌面版启动即崩溃 `SIGABRT: signal arrived during cgo execution`？**

这是 WebView/GTK 的**线程约束**：`Destroy()` / `Terminate()` 等操作必须在**主线程**执行。
[`cmd/esp-desktop/main.go`](../cmd/esp-desktop/main.go) 已改用 `w.Dispatch(func(){ w.Terminate() })`
把关闭动作派发回主线程。若你自行修改窗口生命周期代码，请遵守同一约束。

**Q2c. 桌面版窗口能打开但内容空白 / 截图全黑（常见于无头、VM、远程桌面）？**

WebKitGTK 默认走 DMABUF 硬件合成，在无 GPU 合成器的环境会渲染失败。设置以下环境变量强制软件渲染：

```bash
export WEBKIT_DISABLE_DMABUF_RENDERER=1
export WEBKIT_DISABLE_COMPOSITING_MODE=1
export LIBGL_ALWAYS_SOFTWARE=1
export GDK_BACKEND=x11
./e-sp-line2
```

> 真实桌面（有 GPU/合成器）通常无需这些变量。仅当出现空白窗口时再设置。

**Q3. 前端构建报 `esbuild` 原生模块错误**

pnpm 默认拦截 postinstall。本项目已通过 [`web/pnpm-workspace.yaml`](../web/pnpm-workspace.yaml)
放行 esbuild；若仍失败，执行：

```bash
cd web && rm -rf node_modules && pnpm install
# 或改用 npm install
```

**Q4. 想纯 Go 交叉编译（无 CGO）？**

把 `config/config.yaml` 的 `database.driver` 改为 `postgres`（纯 Go 驱动），
然后使用 `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build .` 即可，无需 mingw。

**Q5. 编译出的 Windows exe 在 Windows 直接运行报缺少 DLL？**

mingw 静态链接通常没问题；若报 `libgcc_s_seh-1.dll` 等缺失，
可在构建时加 `-ldflags "-extldflags -static"` 做全静态链接：

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -trimpath -ldflags "-s -w -extldflags -static" -o bin/e-sp-line2.exe .
```

---

## 9. 快速对照表

| 目标 | 命令 |
| --- | --- |
| Linux 本机 exe | `./scripts/build.sh --target linux` |
| Windows exe（Linux 上） | `./scripts/build.sh --target windows` |
| Windows exe（Windows 上） | `powershell -File scripts\build.ps1` |
| 全部平台 | `./scripts/build.sh --target all` |
| **Linux 桌面窗口版** | `./scripts/build.sh --desktop --target linux` |
| **Windows 桌面窗口版** | `powershell -File scripts\build.ps1 -Desktop` |
| 桌面版开发直接运行窗口 | `make run-desktop` |
| 桌面版内嵌前端测试 | `go test -tags desktop ./pkg/webui/` |
| 仅前端 | `make build-frontend` |
| 完整打包 | `make package` |
