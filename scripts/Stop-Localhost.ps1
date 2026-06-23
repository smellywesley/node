$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$runtimeDirectory = Join-Path $root "work/localhost"
$pidFile = Join-Path $runtimeDirectory "daemon.pid"
$expectedBinary = [IO.Path]::GetFullPath((Join-Path $root "bin/agentos.exe"))

if (-not (Test-Path -LiteralPath $pidFile)) {
    Write-Host "No localhost daemon PID file was found."
    exit 0
}

$daemonID = [int](Get-Content -LiteralPath $pidFile -Raw).Trim()
$process = Get-Process -Id $daemonID -ErrorAction SilentlyContinue
if (-not $process) {
    Remove-Item -LiteralPath $pidFile -Force
    Write-Host "The recorded daemon is no longer running."
    exit 0
}

$actualBinary = $process.Path
if (-not $actualBinary -or [IO.Path]::GetFullPath($actualBinary) -ne $expectedBinary) {
    throw "PID $daemonID does not belong to this workspace's AgentOS binary; refusing to stop it."
}

Stop-Process -Id $daemonID
Wait-Process -Id $daemonID -Timeout 10 -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $pidFile -Force
Write-Host "Stopped AgentOS localhost daemon PID $daemonID."
