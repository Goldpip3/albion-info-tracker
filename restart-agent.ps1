# Restart GDA Agent - stop the running agent, rebuild it, relaunch elevated.
#
# Why this exists: the agent captures raw UDP and so must run as admin,
# and Windows locks agent.exe while it's running (a rebuild fails until
# the old process exits). This script does the whole stop -> build ->
# relaunch-elevated dance so picking up an agent-side fix is one action.
#
# Run it after Claude pushes agent changes:
#   - double-click "Restart GDA Agent.cmd", or
#   - double-click "Restart GDA Agent (Verbose).cmd" for a diagnostic
#     capture (writes agent-verbose.log next to agent.exe), or
#   - from a shell:  pwsh -ExecutionPolicy Bypass -File restart-agent.ps1 [-Trace]
#
# The relaunch always passes --open-browser, so the meter opens on its
# own. You get one UAC prompt (the elevated relaunch). Web-only fixes
# don't need this at all - just hard-refresh the browser.

param(
    # -Trace turns on verbose logging; the agent tees every event line to
    # agent-verbose.log next to the exe so a capture can be read back.
    [switch]$Trace
)

$ErrorActionPreference = "Stop"
$agentDir = Join-Path $PSScriptRoot "agent"
$exe      = Join-Path $agentDir "agent.exe"

Write-Host "GDA agent restart" -ForegroundColor Cyan

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

# 3. Relaunch elevated (triggers a single UAC prompt). Always opens the
#    meter; adds --verbose for a diagnostic capture.
$launchArgs = @("--open-browser")
if ($Trace) {
    $launchArgs += "--verbose"
    Write-Host "  TRACE on - events will be written to agent\agent-verbose.log" -ForegroundColor Yellow
}
Write-Host "  relaunching elevated (approve the UAC prompt)..."
Start-Process -FilePath $exe -WorkingDirectory $agentDir -Verb RunAs -ArgumentList $launchArgs

Write-Host "Done - agent is restarting and the meter will open." -ForegroundColor Green
