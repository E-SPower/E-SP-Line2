#!/usr/bin/env bash
# E-SP-Line2 启动脚本（Linux / macOS）
# 用法: ./run.sh
set -euo pipefail

cd "$(dirname "$0")"

C_GRN='\033[32m'; C_YEL='\033[33m'; C_RED='\033[31m'; C_RST='\033[0m'
[ -t 1 ] || { C_GRN=''; C_YEL=''; C_RED=''; C_RST=''; }

BIN="./e-sp-line2"
[ -x "${BIN}" ] || { printf "${C_RED}[FAIL]${C_RST} 未找到可执行文件 %s\n" "${BIN}" >&2; exit 1; }

mkdir -p data

[ -f "config/config.yaml" ] \
  && printf "${C_GRN}[ OK ]${C_RST} 配置: %s\n" "$(pwd)/config/config.yaml" \
  || printf "${C_YEL}[WARN]${C_RST} 未找到 config/config.yaml，将使用默认配置\n" >&2

# ---------------------------------------------------------------------------
# 接入器运行时预检：Python / 依赖 / Node.js
# 缺失时给出明确的安装指引（不自动安装，避免启动过程意外改动系统）。
# ---------------------------------------------------------------------------
PYTHON_BIN="$(command -v python3 || command -v python || true)"
if [ -z "${PYTHON_BIN}" ]; then
  printf "${C_RED}[FAIL]${C_RST} 未找到 Python —— 接入器无法启动。\n" >&2
  printf "       请先安装接入器运行时:  ./scripts/install-python-deps.sh\n" >&2
else
  MISSING=""
  for m in requests websockets loguru pydantic execjs; do
    "${PYTHON_BIN}" -c "import ${m}" >/dev/null 2>&1 || MISSING="${MISSING} ${m}"
  done
  if [ -n "${MISSING}" ]; then
    printf "${C_YEL}[WARN]${C_RST} Python 缺少接入器依赖:%s\n" "${MISSING}" >&2
    printf "       安装:  ./scripts/install-python-deps.sh\n" >&2
    printf "       （后端仍可启动，但接入器会失败；也可在 WebUI 创建实例时自动安装）\n" >&2
  else
    printf "${C_GRN}[ OK ]${C_RST} Python 接入器依赖就绪 (%s)\n" "${PYTHON_BIN}"
  fi

  command -v node >/dev/null 2>&1 \
    && printf "${C_GRN}[ OK ]${C_RST} Node.js %s\n" "$(node -v)" \
    || printf "${C_YEL}[WARN]${C_RST} 未找到 Node.js —— 接入器签名功能不可用（PyExecJS 依赖）\n" >&2
fi

echo
exec "${BIN}" "$@"
