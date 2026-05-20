@echo off
REM Double-click for a DIAGNOSTIC restart: stop, rebuild, relaunch the GDA
REM agent elevated with verbose logging. Every captured event is written
REM to agent\agent-verbose.log so Claude can read it back directly.
powershell -ExecutionPolicy Bypass -NoProfile -File "%~dp0restart-agent.ps1" -Trace
echo.
pause
