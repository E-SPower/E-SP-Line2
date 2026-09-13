#!/usr/bin/env bash
# E-SP-Line2 桌面窗口版启动脚本（Linux / macOS）
#
# 桌面版是窗口程序：
#   - 从终端运行时，本脚本做完预检后把窗口进程放到后台并立即返回，
#     因此终端不会被窗口占用，可直接关闭；
#   - 从文件管理器/应用菜单启动时，请使用随包的 e-sp-line2.desktop
#     （Terminal=false，不会弹出终端窗口）。
#
# 用法: ./run.sh [--foreground]
set -euo pipefail

cd "$(dirname "$0")"

FOREGROUND=0
[ "${1:-}" = "--foreground" ] && FOREGROUND=1

C_GRN='\033[32m'; C_YEL='\033[33m'; C_RED='\033[31m'; C_RST='\033[0m'
[ -t 1 ] || { C_GRN=''; C_YEL=''; C_RED=''; C_RST=''; }

# 桌面包内的可执行文件名为 e-sp-line2-desktop（与网页版 e-sp-line2 区分）。
# 兼容：若不存在则回退到 e-sp-line2。
BIN="./e-sp-line2-desktop"
[ -x "${BIN}" ] || BIN="./e-sp-line2"
[ -x "${BIN}" ] || { printf "${C_RED}[FAIL]${C_RST} 未找到桌面版可执行文件\n" >&2; exit 1; }

mkdir -p data

[ -f "config/config.yaml" ] \
  && printf "${C_GRN}[ OK ]${C_RST} 配置: %s\n" "$(pwd)/config/config.yaml" \
  || printf "${C_YEL}[WARN]${C_RST} 未找到 config/config.yaml，将使用默认配置\n" >&2

# ---------------------------------------------------------------------------
# 接入器运行时预检
# ---------------------------------------------------------------------------
PYTHON_BIN="$(command -v python3 || command -v python || true)"
if [ -z "${PYTHON_BIN}" ]; then
  printf "${C_YEL}[WARN]${C_RST} 未找到 Python —— 接入器无法启动。\n" >&2
  printf "       安装:  ./scripts/install-python-deps.sh\n" >&2
else
  MISSING=""
  for m in requests websockets loguru pydantic execjs; do
    "${PYTHON_BIN}" -c "import ${m}" >/dev/null 2>&1 || MISSING="${MISSING} ${m}"
  done
  if [ -n "${MISSING}" ]; then
    printf "${C_YEL}[WARN]${C_RST} Python 缺少接入器依赖:%s\n" "${MISSING}" >&2
    printf "       安装:  ./scripts/install-python-deps.sh\n" >&2
  else
    printf "${C_GRN}[ OK ]${C_RST} Python 接入器依赖就绪\n"
  fi
  command -v node >/dev/null 2>&1 \
    && printf "${C_GRN}[ OK ]${C_RST} Node.js %s\n" "$(node -v)" \
    || printf "${C_YEL}[WARN]${C_RST} 未找到 Node.js —— 接入器签名不可用\n" >&2
fi

echo

# ---------------------------------------------------------------------------
# 启动窗口
# ---------------------------------------------------------------------------
if [ "${FOREGROUND}" = "1" ]; then
  printf "${C_GRN}[INFO]${C_RST} 前台启动窗口（关闭窗口即退出）...\n"
  exec "${BIN}" "$@"
fi

printf "${C_GRN}[INFO]${C_RST} 启动桌面窗口（后台运行，日志见 data/desktop.log）...\n"
# 脱离终端后台运行；setsid 使其不随终端关闭而退出。
if command -v setsid >/dev/null 2>&1; then
  setsid "${BIN}" >/dev/null 2>&1 < /dev/null &
else
  nohup "${BIN}" >/dev/null 2>&1 < /dev/null &
fi
sleep 1
printf "${C_GRN}[OK]${C_RST} 窗口进程已启动 (pid %s)\n" "$!"
