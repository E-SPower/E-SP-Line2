@echo off
rem E-SP-Line2 desktop launcher (Windows)
rem
rem The desktop package is a GUI-subsystem program (-H=windowsgui):
rem double-clicking the exe opens the window directly, with no console.
rem This script is only needed when you want a runtime dependency precheck.
rem
rem Usage: run-desktop.bat
setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BIN=e-sp-line2-desktop.exe"
if not exist "%BIN%" set "BIN=e-sp-line2.exe"
if not exist "%BIN%" (
  echo [FAIL] desktop executable not found.
  pause
  exit /b 1
)

if not exist "data" mkdir data

rem ---------------------------------------------------------------------------
rem Adapter runtime precheck (does not block startup)
rem ---------------------------------------------------------------------------
set "PY="
where python >nul 2>nul && set "PY=python"
if not defined PY where python3 >nul 2>nul && set "PY=python3"

if not defined PY (
  echo [WARN] Python not found - adapters cannot start.
  echo        Install: powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1
) else (
  set "MISSING="
  for %%M in (requests websockets loguru pydantic execjs) do (
    %PY% -c "import %%M" >nul 2>nul || set "MISSING=!MISSING! %%M"
  )
  if defined MISSING (
    echo [WARN] Python adapter deps missing:!MISSING!
    echo        Install: powershell -ExecutionPolicy Bypass -File scripts\install-python-deps.ps1
  ) else (
    echo [ OK ] Python adapter dependencies ready
  )
)

where node >nul 2>nul
if errorlevel 1 echo [WARN] Node.js not found - adapter signing unavailable

echo.
echo [INFO] Starting E-SP-Line2 desktop window...
rem The GUI program does not own this console; this window can be closed.
start "" "%BIN%"
endlocal
