@echo off
setlocal
cd /d "%~dp0"

where wails >nul 2>nul
if errorlevel 1 (
  echo [dev] wails CLI not found in PATH
  echo [dev] install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest
  exit /b 1
)

echo [dev] closing running instances
set "KILL_DEV_CMD=$s=(Get-CimInstance Win32_Process -Filter ('ProcessId=' + $PID)).ParentProcessId; Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'cmd.exe' -and $_.ProcessId -ne $s -and $_.CommandLine -like '*dev.bat*' } | ForEach-Object { taskkill /F /T /PID $_.ProcessId 2>&1 | Out-Null }"
powershell -NoProfile -ExecutionPolicy Bypass -Command "%KILL_DEV_CMD%"
for %%N in (ZhiAi-Assistant-dev.exe ZhiAi-Assistant.exe) do (
  taskkill /F /IM "%%N" >nul 2>nul
)
ping -n 2 127.0.0.1 >nul

echo [dev] starting: wails dev %*
echo [dev] hot reload enabled, devserver http://localhost:34115
wails dev %*
if errorlevel 1 (
  echo [dev] failed
  exit /b 1
)

endlocal
