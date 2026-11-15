@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

set "VER="
for /f "tokens=2 delims=:" %%i in ('findstr /c:"productVersion" wails.json') do set "VER=%%i"
if not defined VER (
  echo [build] cannot read productVersion from wails.json
  exit /b 1
)
set "VER=%VER:"=%"
set "VER=%VER: =%"
set "VER=%VER:,=%"

echo [build] version: %VER%
wails build -ldflags "-X zhi-ai-assistant/internal/buildinfo.Version=%VER%" %*
if errorlevel 1 (
  echo [build] failed
  exit /b 1
)

set "RELEASE=ZhiAi-Assistant-%VER%-win-x64.exe"
copy /y "build\bin\ZhiAi-Assistant.exe" "build\bin\%RELEASE%" >nul
if errorlevel 1 (
  echo [build] failed to create release copy
  exit /b 1
)

set "HASH="
for /f "tokens=*" %%h in ('powershell -NoProfile -Command "(Get-FileHash -LiteralPath 'build\bin\%RELEASE%' -Algorithm SHA256).Hash.ToLower()"') do set "HASH=%%h"
if defined HASH (
  >"build\bin\%RELEASE%.sha256" echo !HASH!  %RELEASE%
) else (
  echo [build] warning: checksum not generated
)

echo [build] output: build\bin\ZhiAi-Assistant.exe
echo [build] release: build\bin\%RELEASE%
endlocal
