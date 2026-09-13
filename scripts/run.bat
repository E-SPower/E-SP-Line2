@echo off
rem E-SP-Line2 launcher (Windows, web/server build)
rem
rem Usage: run.bat
rem
rem NOTE: This file must use CRLF line endings and ASCII-only text.
rem cmd.exe mis-parses LF-only files, especially inside if(...) blocks,
rem which produces bogus "is not recognized as an internal or external
rem command" errors.
setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BIN=e-sp-line2.exe"
if not exist "%BIN%" (
  echo [FAIL] executable not found: %BIN%
  exit /b 1
)

if not exist "data" mkdir data

if exist "config\config.yaml" (
  echo [ OK ] config: %CD%\config\config.yaml
) else (
  echo [WARN] config\config.yaml not found, defaults will be used
)

rem ---------------------------------------------------------------------------
rem Adapter runtime precheck
rem ---------------------------------------------------------------------------
set "PY="
where python >nul 2>nul && set "PY=python"
if not defined PY where python3 >nul 2>nul && set "PY=python3"

if not defined PY (
  echo [FAIL] Python not found - adapters cannot start.
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

  where node >nul 2>nul
  if errorlevel 1 echo [WARN] Node.js not found - adapter signing unavailable
)

echo.
"%BIN%" %*
endlocal
