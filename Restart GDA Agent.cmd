@echo off
REM Double-click to stop, rebuild, and relaunch the GDA agent (elevated).
REM Delegates to restart-agent.ps1 next to this file.
powershell -ExecutionPolicy Bypass -NoProfile -File "%~dp0restart-agent.ps1"
echo.
pause
