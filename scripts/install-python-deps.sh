#!/usr/bin/env bash
# =============================================================================
# E-SP-Line2 接入器运行时安装脚本（Linux / macOS）
# -----------------------------------------------------------------------------
# 作用:
#   接入器（淘宝 / 闲鱼）以 Python 子进程运行，需要 Python 3.9+、pip 与 Node.js。
#   本脚本会：
#     1. 检测 Python / pip / Node.js，**缺失时自动通过系统包管理器安装**；
#     2. 安装各接入器 requirements.txt 依赖（含 PEP 668 处理与逐包降级）；
#     3. 做导入自检。
#
# 用法:
#   ./scripts/install-python-deps.sh                # 缺什么装什么（推荐）
#   ./scripts/install-python-deps.sh --venv         # 用隔离虚拟环境 .venv 安装
#   ./scripts/install-python-deps.sh --check        # 仅检查，不做任何安装
#   ./scripts/install-python-deps.sh --no-auto      # 只装 pip 依赖，不装系统包
#   ./scripts/install-python-deps.sh --yes          # 自动确认（免交互）
#   ./scripts/install-python-deps.sh --python /usr/bin/python3
#
# 支持的包管理器: apt / dnf / yum / pacman / zypper / apk / brew
# 需要管理员权限时会自动调用 sudo（请确保当前用户可 sudo）。
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

C_GRN='\033[32m'; C_YEL='\033[33m'; C_RED='\033[31m'; C_CYN='\033[36m'; C_RST='\033[0m'
[ -t 1 ] || { C_GRN=''; C_YEL=''; C_RED=''; C_CYN=''; C_RST=''; }
info() { printf "${C_CYN}[INFO]${C_RST} %s\n" "$*"; }
ok()   { printf "${C_GRN}[ OK ]${C_RST} %s\n" "$*"; }
warn() { printf "${C_YEL}[WARN]${C_RST} %s\n" "$*"; }
die()  { printf "${C_RED}[FAIL]${C_RST} %s\n" "$*" >&2; exit 1; }

PYTHON=""
USE_VENV=0
CHECK_ONLY=0
AUTO_INSTALL=1     # 默认：缺失系统组件时自动安装
ASSUME_YES=0

while [ $# -gt 0 ]; do
  case "$1" in
    --venv)       USE_VENV=1; shift ;;
    --check)      CHECK_ONLY=1; shift ;;
    --no-auto)    AUTO_INSTALL=0; shift ;;
    --yes|-y)     ASSUME_YES=1; shift ;;
    --python)     PYTHON="${2:-}"; shift 2 ;;
    --python=*)   PYTHON="${1#*=}"; shift ;;
    -h|--help)    sed -n '2,27p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "未知参数: $1" ;;
  esac
done

# =============================================================================
# 包管理器识别与系统包安装
# =============================================================================
PKG_MGR=""
detect_pkg_mgr() {
  if [ -n "${PKG_MGR}" ]; then return 0; fi
  for c in apt-get apt dnf yum pacman zypper apk brew; do
    if command -v "$c" >/dev/null 2>&1; then PKG_MGR="$c"; return 0; fi
  done
  PKG_MGR="unknown"; return 1
}

# 是否需要 sudo（非 root 且非 brew 时）
SUDO=""
init_sudo() {
  if [ "$(id -u)" = "0" ]; then SUDO=""; return; fi
  if [ "${PKG_MGR}" = "brew" ]; then SUDO=""; return; fi
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
    if [ "${ASSUME_YES}" = "0" ]; then
      warn "安装系统包需要管理员权限，可能会提示输入密码"
    fi
  else
    SUDO=""
    warn "未找到 sudo，且当前非 root；若安装失败请以 root 重跑"
  fi
}

# sys_install <包名...>
sys_install() {
  [ $# -gt 0 ] || return 0
  detect_pkg_mgr || { warn "未识别的包管理器，无法自动安装: $*"; return 1; }
  init_sudo

  info "使用 ${PKG_MGR} 安装系统包: $*"
  case "${PKG_MGR}" in
    apt-get|apt)
      # 先尝试更新索引（失败不致命）
      ${SUDO} apt-get update -y >/dev/null 2>&1 || warn "apt-get update 失败，继续尝试安装"
      ${SUDO} apt-get install -y "$@"
      ;;
    dnf)     ${SUDO} dnf install -y "$@" ;;
    yum)     ${SUDO} yum install -y "$@" ;;
    pacman)  ${SUDO} pacman -S --needed --noconfirm "$@" ;;
    zypper)  ${SUDO} zypper --non-interactive install "$@" ;;
    apk)     ${SUDO} apk add --no-cache "$@" ;;
    brew)    brew install "$@" ;;
    *)       warn "不支持的包管理器: ${PKG_MGR}"; return 1 ;;
  esac
}

# 各发行版对应的包名
pkgs_for_python() {
  case "${PKG_MGR}" in
    apt-get|apt) echo "python3 python3-pip python3-venv" ;;
    dnf|yum)     echo "python3 python3-pip" ;;
    pacman)      echo "python python-pip" ;;
    zypper)      echo "python3 python3-pip" ;;
    apk)         echo "python3 py3-pip" ;;
    brew)        echo "python" ;;
    *)           echo "" ;;
  esac
}
pkgs_for_node() {
  case "${PKG_MGR}" in
    apt-get|apt) echo "nodejs npm" ;;
    dnf|yum)     echo "nodejs npm" ;;
    pacman)      echo "nodejs npm" ;;
    zypper)      echo "nodejs npm" ;;
    apk)         echo "nodejs npm" ;;
    brew)        echo "node" ;;
    *)           echo "" ;;
  esac
}

# =============================================================================
# 组件检测（缺失则自动安装）
# =============================================================================
find_python() {
  for c in python3 python; do
    if command -v "$c" >/dev/null 2>&1; then
      # 优先选择版本 >= 3.9 的
      if "$c" -c 'import sys; sys.exit(0 if sys.version_info>=(3,9) else 1)' 2>/dev/null; then
        command -v "$c"; return 0
      fi
    fi
  done
  return 1
}

ensure_python() {
  [ -n "${PYTHON}" ] && [ -x "${PYTHON}" ] && return 0

  if PYTHON="$(find_python)"; then
    ok "检测到 Python: ${PYTHON} ($(${PYTHON} -V 2>&1 | awk '{print $2}'))"
    return 0
  fi

  if [ "${AUTO_INSTALL}" = "0" ]; then
    die "未找到 Python 3.9+，且已指定 --no-auto。请手动安装后重试"
  fi

  warn "未检测到 Python 3.9+，尝试自动安装 ..."
  detect_pkg_mgr || die "无法识别包管理器，请手动安装 Python 3.9+"
  # shellcheck disable=SC2086
  sys_install $(pkgs_for_python) || die "Python 自动安装失败，请手动安装"

  # 刷新可能的 shell 路径
  hash -r 2>/dev/null || true
  if PYTHON="$(find_python)"; then
    ok "Python 安装完成: ${PYTHON} ($(${PYTHON} -V 2>&1 | awk '{print $2}'))"
    return 0
  fi
  die "已尝试安装但仍未找到 Python，请手动安装后重试"
}

ensure_pip() {
  if ${PYTHON} -m pip --version >/dev/null 2>&1; then
    ok "pip 可用（python -m pip）"
    return 0
  fi

  if [ "${AUTO_INSTALL}" = "0" ]; then
    warn "python -m pip 不可用（已指定 --no-auto，跳过安装）"
    return 0
  fi

  warn "pip 不可用，尝试安装 ..."
  # shellcheck disable=SC2086
  sys_install $(pkgs_for_python) || true

  if ${PYTHON} -m pip --version >/dev/null 2>&1; then
    ok "pip 安装完成"
  else
    # 退化尝试 ensurepip
    warn "尝试 ensurepip ..."
    ${PYTHON} -m ensurepip --upgrade >/dev/null 2>&1 || true
    ${PYTHON} -m pip --version >/dev/null 2>&1 && ok "pip 已通过 ensurepip 就绪" \
      || warn "pip 仍不可用，将尝试 pip3/pip 回退"
  fi
}

ensure_node() {
  if command -v node >/dev/null 2>&1; then
    ok "检测到 Node.js: $(node -v)"
    return 0
  fi

  if [ "${AUTO_INSTALL}" = "0" ]; then
    warn "未找到 Node.js（接入器签名需要）。已指定 --no-auto，跳过安装"
    return 1
  fi

  warn "未检测到 Node.js，尝试自动安装（PyExecJS 签名依赖）..."
  detect_pkg_mgr || { warn "无法识别包管理器，请手动安装 Node.js 18+"; return 1; }
  # shellcheck disable=SC2086
  if sys_install $(pkgs_for_node); then
    hash -r 2>/dev/null || true
    if command -v node >/dev/null 2>&1; then
      ok "Node.js 安装完成: $(node -v)"
      return 0
    fi
  fi
  warn "Node.js 自动安装失败，请手动安装：https://nodejs.org/"
  return 1
}

# =============================================================================
# 主流程
# =============================================================================
echo "========================================================"
echo " E-SP-Line2 接入器运行时安装"
echo "========================================================"

if [ "${CHECK_ONLY}" = "1" ]; then
  info "检查模式：不安装任何内容"
  # 显式指定的解释器优先；否则自动探测。
  if [ -n "${PYTHON}" ] && [ -x "${PYTHON}" ]; then
    ok "Python: ${PYTHON} ($(${PYTHON} -V 2>&1 | awk '{print $2}'))"
  elif PYTHON="$(find_python)"; then
    ok "Python: ${PYTHON} ($(${PYTHON} -V 2>&1 | awk '{print $2}'))"
  else
    warn "Python 缺失（去掉 --check 可自动安装）"
    PYTHON=""
  fi
  command -v node >/dev/null 2>&1 && ok "Node.js: $(node -v)" || warn "Node.js 缺失"
  # 检查模式仍需解释器做导入自检；若缺失则直接结束。
  if [ -z "${PYTHON}" ]; then
    warn "Python 不可用，无法进行依赖自检"
    exit 1
  fi
else
  ensure_python
  ensure_pip
  ensure_node || true
fi

# ---------- 解释器与 pip 命令 ----------
PYVER="$(${PYTHON} -c 'import sys;print("%d.%d"%sys.version_info[:2])')"
info "Python: ${PYTHON} (${PYVER})"
${PYTHON} -c 'import sys; sys.exit(0 if sys.version_info>=(3,9) else 1)' \
  || die "需要 Python 3.9+，当前为 ${PYVER}"

PIP_ARGS="__none__"
if ${PYTHON} -m pip --version >/dev/null 2>&1; then
  PIP_ARGS="-m pip install"
else
  for c in pip3 pip; do
    if command -v "$c" >/dev/null 2>&1; then
      warn "python -m pip 不可用，回退使用 ${c}"
      PIP_ARGS="__${c}__install"
      break
    fi
  done
fi
[ "${PIP_ARGS}" != "__none__" ] || die "未找到可用的 pip。请安装 python3-pip 后重试"

# ---------- 虚拟环境 ----------
if [ "${USE_VENV}" = "1" ]; then
  VENV_DIR="${ROOT_DIR}/.venv"
  if [ ! -d "${VENV_DIR}" ]; then
    info "创建虚拟环境 .venv ..."
    if ! ${PYTHON} -m venv "${VENV_DIR}" 2>/dev/null; then
      warn "venv 创建失败，尝试安装 venv 支持（python3-venv）..."
      [ "${AUTO_INSTALL}" = "1" ] && sys_install $(pkgs_for_python) || true
      ${PYTHON} -m venv "${VENV_DIR}" || die "创建 venv 失败"
    fi
  fi
  PYTHON="${VENV_DIR}/bin/python"
  PIP_ARGS="-m pip install"
  ok "使用虚拟环境: ${VENV_DIR}"
  warn "请在 config/config.yaml 设置: adapter.python_bin: \"${VENV_DIR}/bin/python\""
fi

# ---------- PEP 668 ----------
BREAK_FLAG=""
if [ "${USE_VENV}" = "0" ] && ${PYTHON} -c '
import sysconfig, os
p = os.path.join(sysconfig.get_paths()["stdlib"], "EXTERNALLY-MANAGED")
sys.exit(0 if os.path.exists(p) else 1)' 2>/dev/null; then
  BREAK_FLAG="--break-system-packages"
  warn "检测到 PEP 668 外部管理环境，将使用 ${BREAK_FLAG}"
fi

run_pip() {
  if [ "${PIP_ARGS#__}" != "${PIP_ARGS}" ]; then
    local bin; bin="${PIP_ARGS#__}"; bin="${bin%%__*}"
    local sub; sub="${PIP_ARGS##*__}"
    "${bin}" "${sub}" "$@"
  else
    # shellcheck disable=SC2086
    ${PYTHON} ${PIP_ARGS} "$@"
  fi
}

# ---------- 收集 requirements ----------
shopt -s nullglob
REQS=()
for f in adapters/*/requirements.txt; do REQS+=("$f"); done
shopt -u nullglob
[ ${#REQS[@]} -gt 0 ] || die "未找到任何 adapters/*/requirements.txt"

echo
info "发现 ${#REQS[@]} 个接入器依赖清单："
for f in "${REQS[@]}"; do echo "   - $f"; done

# ---------- 检查模式 ----------
MODS=(requests websockets loguru pydantic execjs blackboxprotobuf qrcode backoff)
if [ "${CHECK_ONLY}" = "1" ]; then
  echo
  info "导入自检："
  FAILED=0
  for m in "${MODS[@]}"; do
    if ${PYTHON} -c "import ${m}" >/dev/null 2>&1; then
      printf "   ${C_GRN}[✓]${C_RST} %s\n" "${m}"
    else
      printf "   ${C_RED}[✗]${C_RST} %s\n" "${m}"; FAILED=1
    fi
  done
  command -v node >/dev/null 2>&1 \
    && printf "   ${C_GRN}[✓]${C_RST} Node.js %s\n" "$(node -v)" \
    || { printf "   ${C_RED}[✗]${C_RST} Node.js\n"; FAILED=1; }
  [ "${FAILED}" = "0" ] && ok "全部依赖就绪" || warn "存在缺失，去掉 --check 可自动安装"
  exit ${FAILED}
fi

# ---------- 安装依赖 ----------
echo
for f in "${REQS[@]}"; do
  info "安装 $f ..."
  # shellcheck disable=SC2086
  if run_pip install ${BREAK_FLAG} -r "$f"; then
    ok "$f 安装完成"
  else
    warn "$f 整体安装失败，逐包安装（跳过不可用项）..."
    while IFS= read -r line; do
      line="$(printf '%s' "$line" | sed 's/#.*//; s/^[[:space:]]*//; s/[[:space:]]*$//')"
      [ -n "$line" ] || continue
      case "$line" in -*) continue ;; http*) continue ;; esac
      if run_pip install ${BREAK_FLAG} "$line" >/dev/null 2>&1; then
        printf "   ${C_GRN}[✓]${C_RST} %s\n" "$line"
      else
        printf "   ${C_YEL}[跳过]${C_RST} %s\n" "$line"
      fi
    done < "$f"
  fi
done

# ---------- 自检 ----------
echo
info "导入自检..."
FAILED=0
for m in "${MODS[@]}"; do
  if ${PYTHON} -c "import ${m}" >/dev/null 2>&1; then
    printf "   ${C_GRN}[✓]${C_RST} %s\n" "${m}"
  else
    printf "   ${C_RED}[✗]${C_RST} %s\n" "${m}"; FAILED=1
  fi
done

if command -v node >/dev/null 2>&1; then
  printf "   ${C_GRN}[✓]${C_RST} Node.js %s\n" "$(node -v)"
  ${PYTHON} -c "import execjs;print('      execjs 引擎:', execjs.get().name)" 2>/dev/null || true
else
  warn "Node.js 缺失，接入器签名不可用"
  FAILED=1
fi

echo
if [ "${FAILED}" = "0" ]; then
  ok "接入器运行时安装完成，可以启动接入器了"
else
  warn "部分组件缺失，接入器可能无法完整运行"
fi
