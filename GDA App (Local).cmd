@echo off
REM ============================================================
REM   GDA App  -  LOCAL mode
REM ============================================================
REM Double-click to run the meter entirely on THIS PC. The agent
REM serves the meter at http://localhost:8787 and does NOT push to
REM Cloudflare, so it never uses the daily request limit. Best for
REM everyday solo/party play on your own machine.
REM
REM (Rebuilds + relaunches the agent elevated, then opens the local
REM  meter in your browser.)
powershell -ExecutionPolicy Bypass -NoProfile -File "%~dp0restart-agent.ps1" -Local
echo.
pause
