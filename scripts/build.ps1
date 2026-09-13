# =============================================================================
# E-SP-Line2 Windows 原生构建脚本 (PowerShell)
# -----------------------------------------------------------------------------
# 在 Windows 上直接编译原生 exe（使用 MSVC 或 mingw-w64 作为 CGO 编译器）。
#
# 前置条件:
#   1. Go 1.22+            https://go.dev/dl/
#   2. Node.js 18+         https://nodejs.org/
#   3. CGO 编译器（二选一）:
#        - MSVC:  安装 "Visual Studio Build Tools"，勾选 "使用 C++ 的桌面开发"
#        - mingw: winget install -e --id MSYS2.MSYS2  然后 pacman -S mingw-w64-x86_64-gcc
#   4. 若使用 MSVC，请从 "x64 Native Tools Command Prompt" 运行本脚本，
#      或先执行 vcvars64.bat 初始化环境。
#
# 用法:
#   powershell -ExecutionPolicy Bypass -File scripts\build.ps1
#   powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Arch arm64
#   powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -NoFrontend
#   powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -FrontendOnly
# =============================================================================
[CmdletBinding()]
param(
  [ValidateSet('amd64','arm64')]
  [string]$Arch = 'amd64',
  [switch]$NoFrontend,
  [switch]$FrontendOnly,
  [switch]$NoLint,
  # 桌面窗口版：内嵌前端 + 原生 WebView（Windows 使用系统 WebView2 运行时）
  [switch]$Desktop,
  # 随包附带外部 adapters\ 目录（默认不含，因适配器已内嵌进二进制）
  [switch]$IncludeAdapters,
  [string]$Version = ''
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RootDir   = Split-Path -Parent $ScriptDir
Set-Location $RootDir

function Info($m) { Write-Host "[INFO] $m" -ForegroundColor Cyan }
function Ok($m)   { Write-Host "[ OK ] $m" -ForegroundColor Green }
function Warn($m) { Write-Host "[WARN] $m" -ForegroundColor Yellow }
function Die($m)  { Write-Host "[FAIL] $m" -ForegroundColor Red; exit 1 }

$OutDir = Join-Path $RootDir 'dist'

# ---------- 版本号 -----------------------------------------------------------
if ([string]::IsNullOrWhiteSpace($Version)) {
  try {
    $Version = (git describe --tags --always --dirty 2>$null)
    if ([string]::IsNullOrWhiteSpace($Version)) { $Version = 'dev' }
  } catch { $Version = 'dev' }
}
$BuildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$LdFlags   = "-s -w -X main.version=$Version -X main.buildTime=$BuildTime"

# ---------- 前端 -------------------------------------------------------------
function Build-Frontend {
  Info '构建前端 (web/dist)...'
  if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Die '未找到 node，请安装 Node.js 18+'
  }
  $pkgMgr = if (Get-Command pnpm -ErrorAction SilentlyContinue) { 'pnpm' } else { 'npm' }
  Info "包管理器: $pkgMgr / node $(node -v)"

  Push-Location (Join-Path $RootDir 'web')
  try {
    if (-not (Test-Path 'node_modules')) {
      Info "安装前端依赖 ($pkgMgr install)..."
      & $pkgMgr install
      if ($LASTEXITCODE -ne 0) { Die '前端依赖安装失败' }
    } else {
      Info '已存在 node_modules，跳过依赖安装'
    }
    if (-not $NoLint) {
      Info '前端 lint...'
      & $pkgMgr run lint
      if ($LASTEXITCODE -ne 0) { Die '前端 lint 失败（可用 -NoLint 跳过）' }
    }
    Info '前端构建 (tsc && vite build)...'
    & $pkgMgr run build
    if ($LASTEXITCODE -ne 0) { Die '前端构建失败' }
  } finally {
    Pop-Location
  }

  if (-not (Test-Path (Join-Path $RootDir 'web\dist\index.html'))) {
    Die '未生成 web/dist/index.html'
  }
  Ok '前端构建完成 -> web/dist'

  # 桌面版将 web/dist 同步到 pkg/webui/dist 以通过 //go:embed 内嵌进二进制。
  if ($Desktop) {
    Info '同步前端产物到 pkg/webui/dist（桌面内嵌）...'
    $embedDir = Join-Path $RootDir 'pkg\webui\dist'
    if (Test-Path $embedDir) { Remove-Item -Recurse -Force $embedDir }
    New-Item -ItemType Directory -Force -Path $embedDir | Out-Null
    Copy-Item -Recurse (Join-Path $RootDir 'web\dist\*') $embedDir
    # 保留占位 .gitkeep，确保全新检出（未跑过桌面构建）时 go:embed 仍可编译。
    New-Item -ItemType File -Force -Path (Join-Path $embedDir '.gitkeep') | Out-Null
    if (-not (Test-Path (Join-Path $embedDir 'index.html'))) {
      Die '同步失败：pkg/webui/dist/index.html 不存在'
    }
    Ok '前端已内嵌同步 -> pkg/webui/dist'
  }
}

# ---------- 同步 adapters 到内嵌目录 -----------------------------------------
# 把仓库根的 adapters\ 复制到 internal\adapters\adapters\，供
# //go:embed all:adapters 编译进二进制；运行时释放到 data\adapters\。
# 这样发布产物无需外部 adapters\ 目录。
function Sync-Adapters {
  $src = Join-Path $RootDir 'adapters'
  if (-not (Test-Path $src)) {
    Warn '未找到 adapters\ 目录，跳过内嵌同步'
    return
  }
  Info '同步 adapters 到 internal\adapters\adapters（内嵌）...'

  $embedDir = Join-Path $RootDir 'internal\adapters\adapters'
  New-Item -ItemType Directory -Force -Path $embedDir | Out-Null

  # 清空除占位 README 之外的内容，避免残留旧文件。
  Get-ChildItem -Path $embedDir -Force | Where-Object { $_.Name -ne 'README.md' } |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue

  Copy-Item -Recurse (Join-Path $src '*') $embedDir -Force
  Get-ChildItem -Path $embedDir -Recurse -Directory -Filter '__pycache__' |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
  Get-ChildItem -Path $embedDir -Recurse -Filter '*.pyc' |
    Remove-Item -Force -ErrorAction SilentlyContinue

  $n = (Get-ChildItem -Path $embedDir -Recurse -Filter 'adapter.yaml').Count
  if ($n -le 0) { Die "adapters 同步失败：$embedDir 中未找到 adapter.yaml" }
  Ok "adapters 已内嵌同步（$n 个接入器）-> internal\adapters\adapters"
}

# ---------- 后端 -------------------------------------------------------------
function Build-Backend {
  Info "编译后端 GOOS=windows GOARCH=$Arch CGO=1 ..."
  if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Die '未找到 go，请安装 Go 1.22+'
  }

  $subdir = "windows-$Arch"
  if ($Desktop) { $subdir = "desktop-windows-$Arch" }
  $targetDir = Join-Path $OutDir $subdir
  New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
  $out = Join-Path $targetDir 'e-sp-line2.exe'

  $env:CGO_ENABLED = '1'
  $env:GOOS        = 'windows'
  $env:GOARCH      = $Arch

  # 自动探测 mingw gcc（若未在 MSVC 环境且 PATH 中存在）
  if (-not (Get-Command cl.exe -ErrorAction SilentlyContinue)) {
    $mingw = Get-Command 'gcc' -ErrorAction SilentlyContinue
    if ($mingw -and -not $env:CC) {
      Warn "未检测到 MSVC (cl.exe)，将使用 gcc: $($mingw.Source)"
    }
  }

  if ($Desktop) {
    Info '使用 desktop 构建标签（内嵌前端 + WebView2）...'
    # Windows 桌面版必须使用 GUI 子系统（-H=windowsgui），否则双击时
    # 会先弹出一个控制台黑框，看起来像"纯命令行程序"。
    $desktopLd = "$LdFlags -H=windowsgui"
    & go build -trimpath -tags desktop -ldflags $desktopLd -o $out ./cmd/esp-desktop
  } else {
    & go build -trimpath -ldflags $LdFlags -o $out .
  }
  if ($LASTEXITCODE -ne 0) { Die '后端编译失败' }
  Ok "后端编译完成 -> $out"
  return $out
}

# ---------- 打包 -------------------------------------------------------------
function Make-Package {
  param([string]$BinPath)
  $pkgName = "e-sp-line2-$Version-windows-$Arch"
  $binName = 'e-sp-line2.exe'
  $stage   = Join-Path $OutDir "_stage-windows-$Arch"
  if ($Desktop) {
    $pkgName = "e-sp-line2-$Version-desktop-windows-$Arch"
    $binName = 'e-sp-line2-desktop.exe'
    $stage   = Join-Path $OutDir "_stage-desktop-windows-$Arch"
  }

  Info "打包 $pkgName ..."
  if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
  New-Item -ItemType Directory -Force -Path $stage | Out-Null

  Copy-Item $BinPath (Join-Path $stage $binName)

  New-Item -ItemType Directory -Force -Path (Join-Path $stage 'config') | Out-Null
  foreach ($f in @('config\config.yaml','config\form-options.yaml')) {
    if (Test-Path (Join-Path $RootDir $f)) { Copy-Item (Join-Path $RootDir $f) (Join-Path $stage 'config') }
  }

  # 桌面版已内嵌前端，无需外部 web/dist。
  if (-not $Desktop -and (Test-Path (Join-Path $RootDir 'web\dist'))) {
    New-Item -ItemType Directory -Force -Path (Join-Path $stage 'web') | Out-Null
    Copy-Item -Recurse (Join-Path $RootDir 'web\dist') (Join-Path $stage 'web\dist')
  }
  # Python 接入器已内嵌进二进制，运行时自动释放到 data\adapters\，
  # 因此发布包默认不再附带外部 adapters\ 目录（保持单文件分发）。
  # 如需随包提供一份可编辑的副本，用 -IncludeAdapters 构建。
  if ($IncludeAdapters -and (Test-Path (Join-Path $RootDir 'adapters'))) {
    $dst = Join-Path $stage 'adapters'
    New-Item -ItemType Directory -Force -Path $dst | Out-Null
    Copy-Item -Recurse (Join-Path $RootDir 'adapters\*') $dst -Force
    Get-ChildItem -Path $dst -Recurse -Directory -Filter '__pycache__' |
      Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
    Info '已随包附带外部 adapters\（-IncludeAdapters）'
  }
  foreach ($d in @('migrations','docs')) {
    if (Test-Path (Join-Path $RootDir $d)) { Copy-Item -Recurse (Join-Path $RootDir $d) (Join-Path $stage $d) }
  }
  foreach ($f in @('README.md','LICENSE')) {
    if (Test-Path (Join-Path $RootDir $f)) { Copy-Item (Join-Path $RootDir $f) $stage }
  }
  # 桌面版是 GUI 子系统程序（双击即开窗口，不弹控制台），使用桌面专用启动脚本。
  if ($Desktop -and (Test-Path (Join-Path $RootDir 'scripts\run-desktop.bat'))) {
    Copy-Item (Join-Path $RootDir 'scripts\run-desktop.bat') (Join-Path $stage 'run.bat')
  } else {
    Copy-Item (Join-Path $RootDir 'scripts\run.bat') $stage
  }
  $depsScript = Join-Path $RootDir 'scripts\install-python-deps.ps1'
  if (Test-Path $depsScript) {
    $sdir = Join-Path $stage 'scripts'
    New-Item -ItemType Directory -Force -Path $sdir | Out-Null
    Copy-Item $depsScript $sdir
  }

  $zip = Join-Path $OutDir "$pkgName.zip"
  if (Test-Path $zip) { Remove-Item -Force $zip }
  Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip -Force
  Remove-Item -Recurse -Force $stage
  Ok "生成 $zip"
}

# ---------- 主流程 -----------------------------------------------------------
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }
Info "E-SP-Line2 构建 | 版本=$Version | 平台=windows | arch=$Arch"

if (-not $NoFrontend -or $FrontendOnly) { Build-Frontend }
if ($FrontendOnly) { Ok '仅前端构建完成'; exit 0 }

# adapters 必须在编译前同步：它们会被 //go:embed 进二进制，运行时释放到 data\adapters\。
Sync-Adapters

$bin = Build-Backend
Make-Package -BinPath $bin

Ok "全部构建完成，产物位于: $OutDir"
