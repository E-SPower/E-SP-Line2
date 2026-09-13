# =============================================================================
# E-SP-Line2 Windows 文件解锁脚本
# -----------------------------------------------------------------------------
# 问题:
#   从浏览器/网盘下载的 zip 会被 Windows 打上 "Mark of the Web" 标记，
#   解压后每个文件都带着"来自 Internet"属性。后果：
#     - 双击 exe 时弹出"Windows 安全中心：无法打开这些文件"；
#     - 运行 .bat 时被 SmartScreen / 附件管理器阻止。
#
# 本脚本递归清除当前目录下所有文件的 Internet 区域标记 (Zone.Identifier)。
#
# 用法（在解压后的发布目录中执行）:
#   powershell -ExecutionPolicy Bypass -File scripts\unblock-windows.ps1
#
# 也可以对整个发布目录执行:
#   powershell -ExecutionPolicy Bypass -File scripts\unblock-windows.ps1 -Root "C:\path\to\e-sp-line2"
# =============================================================================
[CmdletBinding()]
param(
  # 要解锁的根目录，默认为脚本所在目录的上一级（发布包根目录）
  [string]$Root = ''
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($Root)) {
  $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
  $Root = Split-Path -Parent $scriptDir
}

if (-not (Test-Path $Root)) {
  Write-Host "[FAIL] directory not found: $Root" -ForegroundColor Red
  exit 1
}

Write-Host "========================================================"
Write-Host " E-SP-Line2 file unblock"
Write-Host "========================================================"
Write-Host "Target: $Root"

$count = 0
$failed = 0

# Unblock-File removes the Zone.Identifier alternate data stream.
Get-ChildItem -Path $Root -Recurse -File -Force -ErrorAction SilentlyContinue | ForEach-Object {
  try {
    Unblock-File -LiteralPath $_.FullName -ErrorAction Stop
    $count++
  } catch {
    $failed++
  }
}

# Belt and braces: explicitly delete any leftover Zone.Identifier streams,
# since Unblock-File can silently skip some file types.
Get-ChildItem -Path $Root -Recurse -File -Force -ErrorAction SilentlyContinue | ForEach-Object {
  try {
    Remove-Item -LiteralPath $_.FullName -Stream 'Zone.Identifier' -ErrorAction Stop
  } catch {
    # No such stream is the common (fine) case.
  }
}

Write-Host "[ OK ] unblocked: $count files" -ForegroundColor Green
if ($failed -gt 0) {
  Write-Host "[WARN] failed: $failed files (may be locked or permission denied)" -ForegroundColor Yellow
}

Write-Host ''
Write-Host 'Next steps:' -ForegroundColor Cyan
Write-Host '  1) Run the adapter runtime installer (installs Python/Node if missing):'
Write-Host '       powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1'
Write-Host '  2) Start the app:'
Write-Host '       run.bat                        (web/server build)'
Write-Host '       e-sp-line2-desktop.exe         (desktop window build, double-click)'
