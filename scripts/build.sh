#!/usr/bin/env bash
# =============================================================================
# E-SP-Line2 跨平台构建脚本（Linux / Windows）
# -----------------------------------------------------------------------------
# 用法:
#   ./scripts/build.sh                      # 当前平台（默认）
#   ./scripts/build.sh --target all         # Linux + Windows 全部
#   ./scripts/build.sh --target linux
#   ./scripts/build.sh --target windows
#   ./scripts/build.sh --target windows --arch arm64
#   ./scripts/build.sh --frontend-only      # 只构建前端
#   ./scripts/build.sh --backend-only       # 只构建后端
#   ./scripts/build.sh --no-frontend        # 后端 + 打包，复用已有 web/dist
#   ./scripts/build.sh --no-lint            # 跳过前端 lint
#
# 说明:
#   * 后端使用 gorm.io/driver/sqlite (mattn/go-sqlite3)，**依赖 CGO**，
#     因此 Windows 交叉编译必须安装 mingw-w64 工具链。
#   * 前端为纯静态产物 (web/dist)，后端未内嵌，需独立部署 + 反代 /api /ws。
# =============================================================================
set -euo pipefail

# ---------- 路径 -------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

# ---------- 颜色 -------------------------------------------------------------
if [ -t 1 ]; then
  C_RED='\033[31m'; C_GRN='\033[32m'; C_YEL='\033[33m'; C_CYN='\033[36m'; C_RST='\033[0m'
else
  C_RED=''; C_GRN=''; C_YEL=''; C_CYN=''; C_RST=''
fi
info() { printf "${C_CYN}[INFO]${C_RST} %s\n" "$*"; }
ok()   { printf "${C_GRN}[ OK ]${C_RST} %s\n" "$*"; }
warn() { printf "${C_YEL}[WARN]${C_RST} %s\n" "$*"; }
die()  { printf "${C_RED}[FAIL]${C_RST} %s\n" "$*" >&2; exit 1; }

# ---------- 默认参数 ---------------------------------------------------------
TARGET="host"        # host | linux | windows | all
ARCH=""              # amd64 | arm64   (为空则用默认)
DO_FRONTEND=1
DO_BACKEND=1
DO_LINT=1
DO_PACKAGE=1
DO_DESKTOP=0         # 1 = 桌面窗口版（-tags desktop，内嵌前端 + WebView）
VERSION=""
OUT_DIR="dist"
TAGS=""              # go build -tags

# ---------- 参数解析 ---------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --target)        TARGET="${2:-}"; shift 2 ;;
    --target=*)      TARGET="${1#*=}"; shift ;;
    --arch)          ARCH="${2:-}"; shift 2 ;;
    --arch=*)        ARCH="${1#*=}"; shift ;;
    --frontend-only) DO_BACKEND=0; shift ;;
    --backend-only)  DO_FRONTEND=0; shift ;;
    --no-frontend)   DO_FRONTEND=0; shift ;;
    --no-lint)       DO_LINT=0; shift ;;
    --no-package)    DO_PACKAGE=0; shift ;;
    --desktop)       DO_DESKTOP=1; TAGS="desktop"; shift ;;
    --version)       VERSION="${2:-}"; shift 2 ;;
    --version=*)     VERSION="${1#*=}"; shift ;;
    -h|--help)
      sed -n '2,22p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) die "未知参数: $1（用 --help 查看用法）" ;;
  esac
done

[ -n "${TARGET}" ] || die "--target 不能为空"
case "${TARGET}" in host|linux|windows|all) ;; *) die "非法 --target: ${TARGET}";; esac
if [ -n "${ARCH}" ]; then
  case "${ARCH}" in amd64|arm64) ;; *) die "非法 --arch: ${ARCH}（仅支持 amd64/arm64）";; esac
fi

# ---------- 版本号 -----------------------------------------------------------
if [ -z "${VERSION}" ]; then
  if git -C "${ROOT_DIR}" rev-parse --short HEAD >/dev/null 2>&1; then
    VERSION="$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || git -C "${ROOT_DIR}" rev-parse --short HEAD)"
  else
    VERSION="dev"
  fi
fi
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# ---------- 工具检测 ---------------------------------------------------------
need() { command -v "$1" >/dev/null 2>&1; }

# 选择包管理器：pnpm > npm
PKG_MGR=""
pick_pkg_mgr() {
  if need pnpm; then PKG_MGR="pnpm"
  elif need npm; then PKG_MGR="npm"
  else die "未找到 pnpm 或 npm，请先安装 Node.js 18+"
  fi
}

# 查找 mingw 交叉编译器
find_mingw() {
  local want_arch="$1"   # amd64 | arm64
  local candidates
  if [ "${want_arch}" = "arm64" ]; then
    candidates="aarch64-w64-mingw32-gcc"
  else
    candidates="x86_64-w64-mingw32-gcc"
  fi
  for cc in ${candidates}; do
    if command -v "${cc}" >/dev/null 2>&1; then
      command -v "${cc}"; return 0
    fi
  done
  # 回退：任意 mingw gcc
  command -v x86_64-w64-mingw32-gcc 2>/dev/null && return 0
  command -v aarch64-w64-mingw32-gcc 2>/dev/null && return 0
  return 1
}

# ---------- 前端构建 ---------------------------------------------------------
build_frontend() {
  info "构建前端 (web/dist)..."
  need node || die "未找到 node，请安装 Node.js 18+"
  pick_pkg_mgr
  info "包管理器: ${PKG_MGR} / node $(node -v)"

  pushd web >/dev/null
  if [ ! -d node_modules ]; then
    info "安装前端依赖 (${PKG_MGR} install)..."
    if [ "${PKG_MGR}" = "pnpm" ]; then
      pnpm install
    else
      npm install
    fi
  else
    info "已存在 node_modules，跳过依赖安装（如需强制可删除后重试）"
  fi

  if [ "${DO_LINT}" = "1" ]; then
    info "前端 lint..."
    if [ "${PKG_MGR}" = "pnpm" ]; then pnpm run lint; else npm run lint; fi
  fi

  info "前端构建 (tsc && vite build)..."
  if [ "${PKG_MGR}" = "pnpm" ]; then pnpm run build; else npm run build; fi
  popd >/dev/null

  [ -f web/dist/index.html ] || die "前端构建失败：未生成 web/dist/index.html"
  ok "前端构建完成 -> web/dist"

  # 桌面版将 web/dist 同步到 pkg/webui/dist 以通过 //go:embed 内嵌进二进制。
  if [ "${DO_DESKTOP}" = "1" ]; then
    info "同步前端产物到 pkg/webui/dist（桌面内嵌）..."
    # 保留占位 .gitkeep，确保全新检出（未跑过桌面构建）时 go:embed 仍可编译。
    rm -rf pkg/webui/dist
    mkdir -p pkg/webui/dist
    cp -R web/dist/. pkg/webui/dist/
    touch pkg/webui/dist/.gitkeep
    [ -f pkg/webui/dist/index.html ] || die "同步失败：pkg/webui/dist/index.html 不存在"
    ok "前端已内嵌同步 -> pkg/webui/dist"
  fi
}

# ---------- 同步 adapters 到内嵌目录 -----------------------------------------
# 把仓库根的 adapters/ 复制到 internal/adapters/adapters/，供
# //go:embed all:adapters 编译进二进制；运行时释放到 data/adapters/。
# 这样发布产物无需外部 adapters/ 目录。
sync_adapters() {
  if [ ! -d adapters ]; then
    warn "未找到 adapters/ 目录，跳过内嵌同步"
    return 0
  fi
  info "同步 adapters 到 internal/adapters/adapters（内嵌）..."
  local embed_dir="internal/adapters/adapters"
  # 保留占位 README（即便同步失败也可编译），但清空其余内容避免残留旧文件。
  mkdir -p "${embed_dir}"
  find "${embed_dir}" -mindepth 1 -maxdepth 1 ! -name 'README.md' -exec rm -rf {} + 2>/dev/null || true
  cp -R adapters/. "${embed_dir}/"
  find "${embed_dir}" -name '__pycache__' -type d -prune -exec rm -rf {} + 2>/dev/null || true
  find "${embed_dir}" -name '*.pyc' -delete 2>/dev/null || true

  local n
  n="$(find "${embed_dir}" -name 'adapter.yaml' | wc -l | tr -d ' ')"
  [ "${n}" -gt 0 ] || die "adapters 同步失败：${embed_dir} 中未找到 adapter.yaml"
  ok "adapters 已内嵌同步（${n} 个接入器）-> ${embed_dir}"
}

# ---------- 后端构建（单目标） -----------------------------------------------
# 结果输出路径写入全局变量 LAST_OUT（避免用命令替换吞掉 go build 的退出码）。
LAST_OUT=""
build_backend_one() {
  local goos="$1" goarch="$2"
  local binname="e-sp-line2"
  local ext=""
  [ "${goos}" = "windows" ] && ext=".exe"

  # 桌面版产物独立目录，避免与网页版互相覆盖。
  local subdir="${goos}-${goarch}"
  [ "${DO_DESKTOP}" = "1" ] && subdir="desktop-${goos}-${goarch}"

  local out="${OUT_DIR}/${subdir}/${binname}${ext}"
  mkdir -p "${OUT_DIR}/${subdir}"
  LAST_SUBDIR="${subdir}"

  local hostos hostarch
  hostos="$(go env GOHOSTOS)"; hostarch="$(go env GOHOSTARCH)"
  local native=0
  [ "${goos}" = "${hostos}" ] && [ "${goarch}" = "${hostarch}" ] && native=1

  local cc="" cxx=""
  if [ "${goos}" = "windows" ]; then
    # Windows 目标：CGO 需要 mingw 交叉编译器。
    cc="$(find_mingw "${goarch}")" || die "Windows(${goarch}) 交叉编译需要 mingw-w64，但未找到。请安装：
  Debian/Ubuntu : sudo apt install -y gcc-mingw-w64
  Arch          : sudo pacman -S mingw-w64-gcc
  Fedora        : sudo dnf install -y mingw64-gcc
  macOS         : brew install mingw-w64"
    # webview_go 含 C++ 源文件，Go 会调用 CXX；未设置时回退到宿主机 g++，
    # 从而出现 "unrecognized command-line option '-m64'"。必须一并指定交叉 g++。
    if [ "${DO_DESKTOP}" = "1" ]; then
      local cxx_name
      if [ "${goarch}" = "arm64" ]; then cxx_name="aarch64-w64-mingw32-g++"
      else cxx_name="x86_64-w64-mingw32-g++"; fi
      cxx="$(command -v "${cxx_name}" || true)"
      [ -n "${cxx}" ] || die "Windows 桌面版需要 mingw C++ 编译器 ${cxx_name}。请安装：
  Debian/Ubuntu : sudo apt install -y g++-mingw-w64-x86-64
  Arch          : sudo pacman -S mingw-w64-gcc"

      # webview_go 自带的 WebView2 SDK 头文件不完整：WebView2.h 引用了
      # EventToken.h，但该文件未随模块分发（原生 Windows+MSVC 可从 Windows SDK
      # 找到，mingw 交叉编译则失败）。这里注入本仓库提供的兼容头目录。
      if [ -f "${ROOT_DIR}/scripts/mswebview2-compat/EventToken.h" ]; then
        export CGO_CXXFLAGS="${CGO_CXXFLAGS:-} -I${ROOT_DIR}/scripts/mswebview2-compat"
        export CGO_CFLAGS="${CGO_CFLAGS:-} -I${ROOT_DIR}/scripts/mswebview2-compat"
        info "注入 WebView2 兼容头: scripts/mswebview2-compat (EventToken.h)"
      fi
    fi
  elif [ "${native}" = "0" ]; then
    # 非本机 Linux 目标（如 aarch64 主机编译 amd64），需要对应交叉 gcc。
    case "${goarch}" in
      amd64) cc="$(command -v x86_64-linux-gnu-gcc || true)" ;;
      arm64) cc="$(command -v aarch64-linux-gnu-gcc || true)" ;;
    esac
    [ -n "${cc}" ] || die "在 $(go env GOHOSTARCH) 主机上交叉编译 Linux/${goarch} 需要交叉 C 编译器（CGO 依赖 SQLite）。请安装：
  Debian/Ubuntu : sudo apt install -y gcc-$( [ "${goarch}" = amd64 ] && echo x86-64 || echo aarch64 )-linux-gnu
  Arch          : sudo pacman -S $( [ "${goarch}" = amd64 ] && echo x86_64-linux-gnu || echo aarch64-linux-gnu )-gcc
或改为编译本机架构：./scripts/build.sh --target linux --arch ${hostarch}"
  fi

  # 桌面模式：入口为 cmd/esp-desktop，并带 desktop 构建标签内嵌前端。
  local pkg="."
  local tags_opt=""
  if [ "${DO_DESKTOP}" = "1" ]; then
    pkg="./cmd/esp-desktop"
    tags_opt="-tags ${TAGS}"
    if [ "${goos}" != "windows" ]; then
      # Linux 桌面版依赖 WebKitGTK / GTK 开发库（CGO）。
      if ! pkg-config --exists webkit2gtk-4.1 2>/dev/null && ! pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
        die "桌面版(Linux) 需要 WebKitGTK 开发库，但未检测到。请安装：
  Debian/Ubuntu : sudo apt install -y libwebkit2gtk-4.1-dev libgtk-3-dev build-essential
  Arch          : sudo pacman -S webkit2gtk-4.1 gtk3
  Fedora        : sudo dnf install -y webkit2gtk4.1-devel gtk3-devel"
      fi
      # webview_go 的 cgo 指令要求 webkit2gtk-4.0；Ubuntu 24.04+/Arch 仅提供
      # webkit2gtk-4.1（ABI 重命名，C API 兼容）。这里通过在构建目录内生成
      # 一份 webkit2gtk-4.0.pc（内容来自 4.1），并把它加入 PKG_CONFIG_PATH，
      # 从而在不修改任何第三方源码的前提下完成编译。
      if ! pkg-config --exists webkit2gtk-4.0 2>/dev/null && pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
        shim_dir="${ROOT_DIR}/scripts/pkgconfig"
        mkdir -p "${shim_dir}"
        for base in webkit2gtk javascriptcoregtk; do
          if pkg-config --exists "${base}-4.1" 2>/dev/null; then
            pcdir="$(pkg-config --variable=pcfiledir "${base}-4.1" 2>/dev/null)"
            if [ -n "${pcdir}" ] && [ -f "${pcdir}/${base}-4.1.pc" ]; then
              cp -f "${pcdir}/${base}-4.1.pc" "${shim_dir}/${base}-4.0.pc"
            fi
          fi
        done
        if [ -f "${shim_dir}/webkit2gtk-4.0.pc" ]; then
          export PKG_CONFIG_PATH="${shim_dir}:${PKG_CONFIG_PATH:-}"
          info "使用 .pc 映射: webkit2gtk-4.0 -> 4.1 (${shim_dir})"
        fi
      fi
    fi
  fi

  # Windows 桌面版必须使用 GUI 子系统，否则双击会先弹出控制台黑框。
  # 非桌面版保持 console，方便查看日志与服务输出。
  local ldflags="${LDFLAGS}"
  if [ "${goos}" = "windows" ] && [ "${DO_DESKTOP}" = "1" ]; then
    ldflags="${ldflags} -H=windowsgui"
  fi

  info "编译后端 GOOS=${goos} GOARCH=${goarch} CGO=1 CC=${cc:-系统默认} CXX=${cxx:-默认} pkg=${pkg} tags=${TAGS:-无} -> ${out}"
  # shellcheck disable=SC2086
  CGO_ENABLED=1 GOOS="${goos}" GOARCH="${goarch}" \
    CC="${cc:-${CC:-}}" \
    CXX="${cxx:-${CXX:-}}" \
    go build -trimpath ${tags_opt} -ldflags "${ldflags}" -o "${out}" ${pkg} \
    || die "后端编译失败 (GOOS=${goos} GOARCH=${goarch})"

  [ -f "${out}" ] || die "后端编译未生成产物: ${out}"
  ok "后端编译完成 -> ${out}"
  LAST_OUT="${out}"
}

# ---------- 打包 -------------------------------------------------------------
make_package() {
  local goos="$1" goarch="$2"
  local pkgname="e-sp-line2-${VERSION}-${goos}-${goarch}"
  local binname="e-sp-line2"
  [ "${goos}" = "windows" ] && binname="e-sp-line2.exe"
  # 桌面版与网页版产物名区分，避免混淆。
  if [ "${DO_DESKTOP}" = "1" ]; then
    pkgname="e-sp-line2-${VERSION}-desktop-${goos}-${goarch}"
    binname="e-sp-line2-desktop"
    [ "${goos}" = "windows" ] && binname="e-sp-line2-desktop.exe"
  fi
  local stage="${OUT_DIR}/_stage-${goos}-${goarch}"

  info "打包 ${pkgname}..."
  rm -rf "${stage}"; mkdir -p "${stage}"

  # 可执行文件（桌面版使用独立目录与文件名）
  local built_bin="e-sp-line2"
  [ "${goos}" = "windows" ] && built_bin="e-sp-line2.exe"
  local subdir="${goos}-${goarch}"
  [ "${DO_DESKTOP}" = "1" ] && subdir="desktop-${goos}-${goarch}"
  cp "${OUT_DIR}/${subdir}/${built_bin}" "${stage}/${binname}"

  # 运行期资源
  mkdir -p "${stage}/config"
  [ -f config/config.yaml ]       && cp config/config.yaml       "${stage}/config/" || true
  [ -f config/form-options.yaml ] && cp config/form-options.yaml "${stage}/config/" || true
  [ -f config/config.example.yaml ] && cp config/config.example.yaml "${stage}/config/" || true

  # 前端静态产物（独立部署）。桌面版已内嵌前端，无需外部 web/dist。
  if [ "${DO_DESKTOP}" != "1" ] && [ -d web/dist ]; then
    mkdir -p "${stage}/web"
    cp -R web/dist "${stage}/web/dist"
  fi

  # Python 接入器已内嵌进二进制，运行时自动释放到 data/adapters/，
  # 因此发布包默认不再附带外部 adapters/ 目录（保持单文件分发）。
  # 如需随包提供一份可编辑的副本，用 INCLUDE_ADAPTERS=1 构建。
  if [ "${INCLUDE_ADAPTERS:-0}" = "1" ] && [ -d adapters ]; then
    mkdir -p "${stage}/adapters"
    cp -R adapters/. "${stage}/adapters/"
    find "${stage}/adapters" -name '__pycache__' -type d -prune -exec rm -rf {} + 2>/dev/null || true
    info "已随包附带外部 adapters/（INCLUDE_ADAPTERS=1）"
  fi

  # 数据库迁移
  [ -d migrations ] && cp -R migrations "${stage}/migrations"

  # 文档
  [ -d docs ] && cp -R docs "${stage}/docs"
  [ -f README.md ] && cp README.md "${stage}/"
  [ -f LICENSE ]   && cp LICENSE   "${stage}/"

  # 启动脚本 + Python 依赖安装脚本
  #
  # 桌面版是窗口程序：Windows 下以 GUI 子系统链接（不弹控制台），Linux 下
  # 通过 .desktop 启动器（Terminal=false）启动，因此在两种平台都使用桌面专用
  # 启动脚本，仅做运行时预检后把窗口进程放到后台。
  mkdir -p "${stage}/scripts"
  if [ "${goos}" = "windows" ]; then
    if [ "${DO_DESKTOP}" = "1" ] && [ -f scripts/run-desktop.bat ]; then
      cp scripts/run-desktop.bat "${stage}/run.bat"
    else
      cp scripts/run.bat "${stage}/run.bat"
    fi
    [ -f scripts/install-python-deps.ps1 ] && cp scripts/install-python-deps.ps1 "${stage}/scripts/"
    [ -f scripts/unblock-windows.ps1 ] && cp scripts/unblock-windows.ps1 "${stage}/scripts/"
    # 附一份纯 ASCII 的快速说明，避免编码问题
    cat > "${stage}/WINDOWS-README.txt" <<'EOT'
E-SP-Line2 - Windows quick start
================================

1) Unblock files (required after downloading the zip):
     powershell -ExecutionPolicy Bypass -File scripts\unblock-windows.ps1
   Without this Windows may refuse to run the downloaded exe ("无法打开这些文件").

2) Install the adapter runtime (installs Python and Node.js if missing):
     powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1

3) Start:
     e-sp-line2-desktop.exe   -> desktop window (double-click, no console)
     run.bat                  -> web/server build
   Logs are written to data\desktop.log for the desktop build.
EOT
  else
    if [ "${DO_DESKTOP}" = "1" ]; then
      # 已写入桌面版专用 run.sh（见下），并附带 .desktop 启动器供图形化启动。
      if [ -f scripts/run-desktop.sh ]; then
        cp scripts/run-desktop.sh "${stage}/run.sh"
      else
        cp scripts/run.sh "${stage}/run.sh"
      fi
      if [ -f scripts/esp-desktop.desktop ]; then
        cp scripts/esp-desktop.desktop "${stage}/e-sp-line2.desktop"
      fi
    else
      cp scripts/run.sh "${stage}/run.sh"
    fi
    chmod +x "${stage}/run.sh" "${stage}/${binname}"
    if [ -f scripts/install-python-deps.sh ]; then
      cp scripts/install-python-deps.sh "${stage}/scripts/"
      chmod +x "${stage}/scripts/install-python-deps.sh" 2>/dev/null || true
    fi
  fi

  # Windows 文本脚本必须是 CRLF：LF-only 的 .bat 会被 cmd.exe 误解析
  # （尤其在 if(...) 块内），产生大量 "is not recognized..." 报错。
  # .ps1 同样统一为 CRLF，避免 Windows PowerShell 5.1 上的编码/解析问题。
  if [ "${goos}" = "windows" ]; then
    converted=0
    while IFS= read -r f; do
      if sed -i 's/\r$//; s/$/\r/' "${f}" 2>/dev/null; then
        converted=$((converted + 1))
      fi
    done < <(find "${stage}" \( -name '*.bat' -o -name '*.ps1' -o -name '*.txt' \) 2>/dev/null)
    info "已将 ${converted} 个 Windows 文本文件转换为 CRLF"
  fi

  # 压缩
  mkdir -p "${OUT_DIR}"
  if [ "${goos}" = "windows" ]; then
    if need zip; then
      ( cd "${stage}" && zip -qr "../${pkgname}.zip" . ) && ok "生成 ${OUT_DIR}/${pkgname}.zip"
    else
      tar -czf "${OUT_DIR}/${pkgname}.tar.gz" -C "${stage}" . && ok "生成 ${OUT_DIR}/${pkgname}.tar.gz（未找到 zip）"
    fi
  else
    tar -czf "${OUT_DIR}/${pkgname}.tar.gz" -C "${stage}" . && ok "生成 ${OUT_DIR}/${pkgname}.tar.gz"
  fi

  rm -rf "${stage}"
}

# ---------- 主流程 -----------------------------------------------------------
main() {
  info "E-SP-Line2 构建 | 版本=${VERSION} | 目标=${TARGET} | arch=${ARCH:-默认}"
  info "工作目录: ${ROOT_DIR}"
  mkdir -p "${OUT_DIR}"

  need go || die "未找到 go，请安装 Go 1.22+"

  if [ "${DO_FRONTEND}" = "1" ]; then
    build_frontend
  else
    warn "跳过前端构建"
  fi

  if [ "${DO_BACKEND}" != "1" ]; then
    ok "仅前端构建完成"
    return 0
  fi

  # Adaptes must be synced BEFORE compiling: they are //go:embed-ed into the
  # binary and extracted to data/adapters/ at runtime.
  sync_adapters

  local host_os host_arch
  host_os="$(go env GOHOSTOS)"
  host_arch="$(go env GOHOSTARCH)"

  # 默认架构策略：
  #   host / linux  -> 跟随本机架构（保证 CGO 可用，天然可编译）
  #   windows       -> amd64（最常见的 Windows 目标）
  #   all           -> linux 用本机架构，windows 用 amd64
  # 用户可用 --arch 显式覆盖。
  local linux_arch="${ARCH:-${host_arch}}"
  local win_arch="${ARCH:-amd64}"

  local targets=""
  case "${TARGET}" in
    host)    targets="${host_os}:${ARCH:-${host_arch}}" ;;
    linux)   targets="linux:${linux_arch}" ;;
    windows) targets="windows:${win_arch}" ;;
    all)     targets="linux:${linux_arch} windows:${win_arch}" ;;
  esac

  local built=""
  for t in ${targets}; do
    local tgoos="${t%%:*}" tarch="${t##*:}"
    build_backend_one "${tgoos}" "${tarch}"
    built="${built}${tgoos}:${tarch}:${LAST_OUT}
"
  done

  if [ "${DO_PACKAGE}" = "1" ]; then
    for t in ${targets}; do
      local tgoos="${t%%:*}" tarch="${t##*:}"
      make_package "${tgoos}" "${tarch}"
    done
  fi

  echo
  ok "全部构建完成，产物位于: ${ROOT_DIR}/${OUT_DIR}"
  printf '%s' "${built}" | while IFS= read -r line; do
    [ -n "${line}" ] && echo "    - ${line}"
  done
}

main "$@"
