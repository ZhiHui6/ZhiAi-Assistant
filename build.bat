@echo off
setlocal
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

echo [build] output: build\bin\ZhiAi-Assistant.exe
endlocal
