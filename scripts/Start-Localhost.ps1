param(
    [string]$Address = "127.0.0.1:7467",
    [string]$StateDirectory = "",
    [switch]$NoOpen
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$runtimeDirectory = Join-Path $root "work/localhost"
if (-not $StateDirectory) {
    $StateDirectory = Join-Path $runtimeDirectory "state"
}
$pidFile = Join-Path $runtimeDirectory "daemon.pid"
$stdoutFile = Join-Path $runtimeDirectory "daemon.out"
$stderrFile = Join-Path $runtimeDirectory "daemon.err"
$binary = Join-Path $root "bin/agentos.exe"

New-Item -ItemType Directory -Force $runtimeDirectory | Out-Null

try {
    $health = Invoke-RestMethod -Uri "http://$Address/v1/health" -TimeoutSec 2
    if ($health.status -eq "ok") {
        throw "A daemon is already listening at http://$Address. Stop it or choose another -Address."
    }
} catch {
    if ($_.Exception.Message -like "A daemon is already listening*") {
        throw
    }
}

if (-not (Test-Path -LiteralPath $binary)) {
    & (Join-Path $PSScriptRoot "Build.ps1")
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$env:AGENTOS_HOME = $StateDirectory
$env:AGENTOS_ADDR = $Address
$env:AGENTOS_APPROVER_TOKEN = [guid]::NewGuid().ToString("N") + [guid]::NewGuid().ToString("N")

$processPath = [Environment]::GetEnvironmentVariable("Path", "Process")
if ($processPath) {
    [Environment]::SetEnvironmentVariable("PATH", $null, "Process")
    [Environment]::SetEnvironmentVariable("Path", $processPath, "Process")
}

$daemon = Start-Process `
    -FilePath $binary `
    -ArgumentList @("serve", "--addr", $Address) `
    -WorkingDirectory $root `
    -WindowStyle Hidden `
    -RedirectStandardOutput $stdoutFile `
    -RedirectStandardError $stderrFile `
    -PassThru

Set-Content -LiteralPath $pidFile -Value $daemon.Id

$ready = $false
for ($attempt = 0; $attempt -lt 40; $attempt++) {
    Start-Sleep -Milliseconds 250
    if ($daemon.HasExited) {
        $details = if (Test-Path -LiteralPath $stderrFile) {
            Get-Content -LiteralPath $stderrFile -Raw
        } else {
            "no daemon error output"
        }
        throw "AgentOS daemon exited during startup: $details"
    }
    try {
        $health = Invoke-RestMethod -Uri "http://$Address/v1/health" -TimeoutSec 1
        if ($health.status -eq "ok") {
            $ready = $true
            break
        }
    } catch {
        # The daemon may still be opening SQLite and recovering processes.
    }
}

if (-not $ready) {
    Stop-Process -Id $daemon.Id -Force -ErrorAction SilentlyContinue
    throw "AgentOS did not become healthy within 10 seconds. See $stderrFile."
}

try {
    if ($NoOpen) {
        & $binary dashboard --print-url
    } else {
        & $binary dashboard
    }
    if ($LASTEXITCODE -ne 0) {
        throw "The dashboard launcher returned exit code $LASTEXITCODE."
    }
} catch {
    Stop-Process -Id $daemon.Id -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $pidFile -Force -ErrorAction SilentlyContinue
    throw "The daemon started, but the dashboard launcher failed: $($_.Exception.Message)"
}

Write-Host ""
Write-Host "AgentOS localhost is ready."
Write-Host "Dashboard: http://$Address/"
Write-Host "State:     $StateDirectory"
Write-Host "Daemon:    PID $($daemon.Id)"
Write-Host "Stop with: .\scripts\stop-localhost.cmd"
Write-Host "Docker is only required when you run a containerized agent process."
