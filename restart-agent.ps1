# Restart GDA Agent - stop the running agent, rebuild it, relaunch.
#
# Why this exists: the agent captures raw UDP and so must run as admin,
# and Windows locks agent.exe while it's running (a rebuild fails until
# the old process exits). This script does the whole stop -> build ->
# relaunch dance so picking up an agent-side fix is one action.
#
# Because the running agent is elevated, stopping it needs admin too, so
# the script self-elevates up front (one UAC prompt). Everything then
# runs as admin and the relaunched agent inherits elevation - no second
# prompt.
#
# Run it after Claude pushes agent changes:
#   - double-click "Restart GDA Agent.cmd", or
#   - double-click "Restart GDA Agent (Verbose).cmd" for a diagnostic
#     capture (writes agent-verbose.log to %LocalAppData%\GDA), or
#   - from a shell:  pwsh -ExecutionPolicy Bypass -File restart-agent.ps1 [-Trace]
#
# The relaunch always passes --open-browser, so the meter opens on its
# own. Web-only fixes don't need this at all - just hard-refresh.

param(
    # -Trace turns on verbose logging; the agent tees every event line to
    # %LocalAppData%\GDA\agent-verbose.log so a capture can be read back.
    [switch]$Trace
)

$ErrorActionPreference = "Stop"

# Self-elevate: a non-admin process can't Stop-Process the elevated
# agent (Access denied). Relaunch this same script as admin, forwarding
# -Trace, then exit the non-elevated copy.
$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$isAdmin = (New-Object Security.Principal.WindowsPrincipal($id)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "Requesting administrator rights..." -ForegroundColor Cyan
    $argv = @("-ExecutionPolicy", "Bypass", "-NoProfile", "-File", "`"$PSCommandPath`"")
    if ($Trace) { $argv += "-Trace" }
    Start-Process -FilePath "powershell.exe" -Verb RunAs -ArgumentList $argv
    exit
}

$agentDir = Join-Path $PSScriptRoot "agent"
$exe      = Join-Path $agentDir "agent.exe"

Write-Host "GDA agent restart (elevated)" -ForegroundColor Cyan

# 1. Stop any running agent so the .exe unlocks.
$running = Get-Process -Name agent -ErrorAction SilentlyContinue
if ($running) {
    Write-Host ("  stopping running agent (pid {0})..." -f ($running.Id -join ', '))
    $running | Stop-Process -Force
    # Give Windows a moment to release the file handle.
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep -Milliseconds 150
        try { [System.IO.File]::OpenWrite($exe).Close(); break } catch { }
    }
} else {
    Write-Host "  no running agent found - fresh start."
}

# 2. Rebuild.
Write-Host "  building agent.exe..."
Push-Location $agentDir
try {
    & go build -o agent.exe ./cmd/agent
    if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }
} finally {
    Pop-Location
}
Write-Host "  build OK." -ForegroundColor Green

# 3. Relaunch. We're already elevated, so a plain Start-Process inherits
#    admin (no second UAC). Always opens the meter; adds --verbose for a
#    diagnostic capture.
$launchArgs = @("--open-browser")
if ($Trace) {
    $launchArgs += "--verbose"
    Write-Host "  TRACE on - events will be written to %LocalAppData%\GDA\agent-verbose.log" -ForegroundColor Yellow
}
Write-Host "  relaunching agent..."
Start-Process -FilePath $exe -WorkingDirectory $agentDir -ArgumentList $launchArgs

Write-Host "Done - agent is restarting and the meter will open." -ForegroundColor Green
Start-Sleep -Seconds 2
