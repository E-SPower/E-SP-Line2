# =============================================================================
# E-SP-Line2 接入器运行时安装脚本（Windows / PowerShell）
# -----------------------------------------------------------------------------
# 作用:
#   接入器（淘宝/闲鱼）以 Python 子进程运行，需要 Python 3.9+、pip 与 Node.js。
#   本脚本会：
#     1. 检测 Python / pip / Node.js，**缺失时自动安装**（优先 winget，
#        回退到从 python.org / nodejs.org 下载官方安装包静默安装）；
#     2. 安装各接入器 requirements.txt 依赖；
#     3. 做导入自检。
#
# 用法:
#   powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1
#   powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Venv
#   powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Check
#   powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -NoAuto
#   powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1 -Yes
#
# 注意:
#   * 安装系统组件可能需要管理员权限；若当前不是管理员，脚本会提示以管理员身份重跑。
#   * Python 版本建议 3.10~3.12（PyExecJS 在更高版本可能不稳定）。
# =============================================================================
[CmdletBinding()]
param(
  [switch]$Venv,
  [switch]$Check,
  [switch]$NoAuto,
  [switch]$Yes,
  [string]$Python = ''
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RootDir   = Split-Path -Parent $ScriptDir
Set-Location $RootDir

function Info($m) { Write-Host "[INFO] $m" -ForegroundColor Cyan }
function Ok($m)   { Write-Host "[ OK ] $m" -ForegroundColor Green }
function Warn($m) { Write-Host "[WARN] $m" -ForegroundColor Yellow }
function Die($m)  { Write-Host "[FAIL] $m" -ForegroundColor Red; exit 1 }

$AutoInstall = -not $NoAuto

function Test-Admin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  (New-Object Security.Principal.WindowsPrincipal $id).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Have-Cmd($name) { [bool](Get-Command $name -ErrorAction SilentlyContinue) }

# 刷新 PATH（winget/安装器写注册表后当前会话不一定可见）
function Update-Path {
  $m = [Environment]::GetEnvironmentVariable('Path', 'Machine')
  $u = [Environment]::GetEnvironmentVariable('Path', 'User')
  $env:Path = (@($m, $u) | Where-Object { $_ }) -join ';'
}

# 在常见安装位置搜索 python.exe
function Find-PythonExe {
  $candidates = @()
  foreach ($root in @("$env:LOCALAPPDATA\Programs\Python", "$env:ProgramFiles", "${env:ProgramFiles(x86)}")) {
    if ($root -and (Test-Path $root)) {
      $candidates += Get-ChildItem -Path $root -Filter 'python.exe' -Recurse -ErrorAction SilentlyContinue |
                     Where-Object { $_.FullName -match 'Python\d+' } | Select-Object -ExpandProperty FullName
    }
  }
  foreach ($c in @('python','python3')) {
    $cmd = Get-Command $c -ErrorAction SilentlyContinue
    if ($cmd) { $candidates += $cmd.Source }
  }
  foreach ($p in $candidates) {
    try {
      & $p -c 'import sys; sys.exit(0 if sys.version_info>=(3,9) else 1)' 2>$null
      if ($LASTEXITCODE -eq 0) { return $p }
    } catch { }
  }
  return $null
}

# ---------- 自动安装：Python ----------
function Install-Python {
  Warn '未检测到 Python 3.9+，尝试自动安装 ...'

  if (Have-Cmd winget) {
    Info '使用 winget 安装 Python 3.12 ...'
    $args = @('install','-e','--id','Python.Python.3.12',
              '--accept-source-agreements','--accept-package-agreements','--silent')
    if ($Yes) { $args += '--disable-interactivity' }
    & winget @args
    Update-Path
    $p = Find-PythonExe
    if ($p) { Ok "Python 安装完成: $p"; return $p }
  }

  # 回退：从 python.org 下载官方安装包静默安装
  if (-not (Test-Admin)) {
    Warn '非管理员权限，可能无法安装到系统目录；建议以管理员身份重跑本脚本'
  }
  $ver  = '3.12.8'
  $url  = "https://www.python.org/ftp/python/$ver/python-$ver-amd64.exe"
  $dest = Join-Path $env:TEMP "python-$ver-amd64.exe"
  Info "下载 Python 安装包: $url"
  try {
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
  } catch {
    Die "下载 Python 失败: $($_.Exception.Message)`n请手动安装: https://www.python.org/downloads/"
  }
  Info '静默安装 Python（含 PATH 与 pip）...'
  & $dest /quiet InstallAllUsers=1 PrependPath=1 Include_pip=1 Include_test=0
  Update-Path
  $p = Find-PythonExe
  if ($p) { Ok "Python 安装完成: $p"; return $p }
  Die 'Python 已尝试安装但仍未找到，请手动安装后重试'
}

# ---------- 自动安装：Node.js ----------
function Install-Node {
  if (Have-Cmd node) { Ok "检测到 Node.js: $(node -v)"; return $true }
  if (-not $AutoInstall) { Warn '未找到 Node.js（已指定 -NoAuto，跳过）'; return $false }

  Warn '未检测到 Node.js，尝试自动安装（PyExecJS 签名依赖）...'
  if (Have-Cmd winget) {
    Info '使用 winget 安装 Node.js LTS ...'
    & winget install -e --id OpenJS.NodeJS.LTS `
        --accept-source-agreements --accept-package-agreements --silent
    Update-Path
    if (Have-Cmd node) { Ok "Node.js 安装完成: $(node -v)"; return $true }
  }

  $ver  = 'v20.18.1'
  $url  = "https://nodejs.org/dist/$ver/node-$ver-x64.msi"
  $dest = Join-Path $env:TEMP "node-$ver-x64.msi"
  Info "下载 Node.js 安装包: $url"
  try {
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
  } catch {
    Warn "下载 Node.js 失败: $($_.Exception.Message)"
    Warn '请手动安装: https://nodejs.org/'
    return $false
  }
  Info '静默安装 Node.js ...'
  Start-Process msiexec.exe -ArgumentList "/i `"$dest`" /qn /norestart" -Wait
  Update-Path
  if (Have-Cmd node) { Ok "Node.js 安装完成: $(node -v)"; return $true }
  Warn 'Node.js 已尝试安装但仍未找到，请手动安装'
  return $false
}

# =============================================================================
# 主流程
# =============================================================================
Write-Host '========================================================'
Write-Host ' E-SP-Line2 接入器运行时安装'
Write-Host '========================================================'

# ---------- 定位 / 安装 Python ----------
if (-not [string]::IsNullOrWhiteSpace($Python)) {
  if (-not (Test-Path $Python)) { Die "指定的 Python 不存在: $Python" }
} else {
  $found = Find-PythonExe
  if ($found) {
    $Python = $found
    Ok "检测到 Python: $Python ($(& $Python -c 'import sys;print(\"%d.%d\"%sys.version_info[:2])'))"
  } elseif ($AutoInstall -and -not $Check) {
    $Python = Install-Python
  } else {
    Die @'
未找到 Python 3.9+。请安装：
  winget install -e --id Python.Python.3.12
  或从 https://www.python.org/downloads/ 下载（勾选 "Add python.exe to PATH"）
（去掉 -NoAuto 可让本脚本自动安装）
'@
  }
}

$pyVer = & $Python -c 'import sys;print("%d.%d"%sys.version_info[:2])'
& $Python -c 'import sys; sys.exit(0 if sys.version_info>=(3,9) else 1)'
if ($LASTEXITCODE -ne 0) { Die "需要 Python 3.9+，当前为 $pyVer" }

# ---------- pip ----------
& $Python -m pip --version *> $null
if ($LASTEXITCODE -ne 0) {
  Warn 'pip 不可用，尝试 ensurepip ...'
  & $Python -m ensurepip --upgrade *> $null
  & $Python -m pip --version *> $null
  if ($LASTEXITCODE -ne 0) {
    if ($AutoInstall -and -not $Check) { & $Python -m ensurepip --default-pip *> $null }
    & $Python -m pip --version *> $null
    if ($LASTEXITCODE -ne 0) { Die 'pip 不可用，请手动安装 pip' }
  }
}
Ok 'pip 可用'

# ---------- Node.js ----------
$hasNode = Install-Node

# ---------- Check 模式 ----------
$mods = @('requests','websockets','loguru','pydantic','execjs','blackboxprotobuf','qrcode','backoff')
if ($Check) {
  Write-Host ''
  Info '检查模式：仅做导入自检'
  $failed = $false
  foreach ($m in $mods) {
    & $Python -c "import $m" *> $null
    if ($LASTEXITCODE -eq 0) { Write-Host "   [OK] $m" -ForegroundColor Green }
    else { Write-Host "   [MISSING] $m" -ForegroundColor Red; $failed = $true }
  }
  if ($hasNode) { Write-Host '   [OK] Node.js' -ForegroundColor Green }
  else { Write-Host '   [MISSING] Node.js' -ForegroundColor Red; $failed = $true }
  if ($failed) { Warn '存在缺失，去掉 -Check 可自动安装'; exit 1 }
  Ok '全部依赖就绪'; exit 0
}

# ---------- 虚拟环境 ----------
if ($Venv) {
  $venvDir = Join-Path $RootDir '.venv'
  if (-not (Test-Path $venvDir)) {
    Info '创建虚拟环境 .venv ...'
    & $Python -m venv $venvDir
    if ($LASTEXITCODE -ne 0) { Die '创建 venv 失败' }
  }
  $Python = Join-Path $venvDir 'Scripts\python.exe'
  Ok "使用虚拟环境: $venvDir"
  Warn "请在 config\config.yaml 设置 adapter.python_bin: `"$Python`""
}

# ---------- 收集 requirements ----------
$reqs = Get-ChildItem -Path (Join-Path $RootDir 'adapters') -Recurse -Filter 'requirements.txt' -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\adapters\\[^\\]+\\requirements\.txt$' }
if (-not $reqs) { Die '未找到任何 adapters\*\requirements.txt' }

Write-Host ''
Info "发现 $($reqs.Count) 个接入器依赖清单："
foreach ($r in $reqs) { Write-Host "   - $($r.FullName.Substring($RootDir.Length + 1))" }

# ---------- 安装依赖 ----------
Write-Host ''
foreach ($r in $reqs) {
  Info "安装 $($r.FullName) ..."
  & $Python -m pip install -r $r.FullName
  if ($LASTEXITCODE -ne 0) {
    Warn '整体安装失败，逐包安装（跳过不可用项）...'
    Get-Content $r.FullName | ForEach-Object {
      $line = ($_ -replace '#.*','').Trim()
      if ([string]::IsNullOrWhiteSpace($line)) { return }
      if ($line.StartsWith('-') -or $line.StartsWith('http')) { return }
      & $Python -m pip install $line *> $null
      if ($LASTEXITCODE -eq 0) { Write-Host "   [OK] $line" -ForegroundColor Green }
      else { Write-Host "   [跳过] $line" -ForegroundColor Yellow }
    }
  } else {
    Ok "$($r.Name) 安装完成"
  }
}

# ---------- 自检 ----------
Write-Host ''
Info '导入自检...'
$failed = $false
foreach ($m in $mods) {
  & $Python -c "import $m" *> $null
  if ($LASTEXITCODE -eq 0) { Write-Host "   [OK] $m" -ForegroundColor Green }
  else { Write-Host "   [MISSING] $m" -ForegroundColor Red; $failed = $true }
}
if ($hasNode) { Write-Host "   [OK] Node.js $(node -v)" -ForegroundColor Green }
else { Warn 'Node.js 缺失，接入器签名不可用'; $failed = $true }

Write-Host ''
if ($failed) { Warn '部分组件缺失，接入器可能无法完整运行' }
else { Ok '接入器运行时安装完成，可以启动接入器了' }
