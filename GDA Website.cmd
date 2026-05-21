@echo off
REM ============================================================
REM   GDA Website  -  CLOUDFLARE mode
REM ============================================================
REM Double-click to push the meter to the Cloudflare-hosted website
REM (albion-meter-web.pages.dev) so you can open it from your phone
REM or another device. This DOES use the daily Cloudflare request
REM limit, so prefer "GDA App (Local)" for everyday play on this PC.
REM
REM (Rebuilds + relaunches the agent elevated, then opens the website.)
powershell -ExecutionPolicy Bypass -NoProfile -File "%~dp0restart-agent.ps1"
echo.
pause
