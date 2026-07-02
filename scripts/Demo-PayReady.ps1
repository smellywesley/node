param(
    [string]$Address = "127.0.0.1:17479",
    [switch]$KeepRunning
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$agentos = Join-Path $root "bin\agentos.exe"
$stateHome = Join-Path $root "work\pay-ready-home"
$workspace = Join-Path $root "work\pay-ready-workspace"
$internalDir = Join-Path $workspace "internal"
$webDir = Join-Path $workspace "web"
$outputs = Join-Path $root "outputs"
$auditPath = Join-Path $outputs "pay-ready-audit.json"
$daemonOut = Join-Path $stateHome "daemon.out"
$daemonErr = Join-Path $stateHome "daemon.err"

function Write-Step([string]$message) {
    Write-Host "[pay-ready] $message"
}

function Assert-UnderRoot([string]$RootPath, [string]$ChildPath) {
    $rootFull = [System.IO.Path]::GetFullPath($RootPath).TrimEnd('\', '/')
    $childFull = [System.IO.Path]::GetFullPath($ChildPath)
    if (-not $childFull.StartsWith($rootFull + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to touch path outside demo workspace: $ChildPath"
    }
}


function Assert-DockerReady {
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if (-not $docker) {
        Write-Host "problem: Docker is not installed or not on PATH."
        Write-Host "cause: AgentOS runs agent workers in OCI-compatible containers."
        Write-Host "fix: Install Docker Desktop, start it, then rerun this demo. Dashboard-only flows still work without Docker."
        exit 2
    }
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $dockerOutput = & docker version --format '{{.Server.Version}}' 2>&1
        $dockerExitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($dockerExitCode -ne 0) {
        Write-Host "problem: Docker is installed but the engine is not reachable."
        Write-Host "cause: Docker Desktop is stopped, still starting, or using an unavailable engine pipe."
        Write-Host "fix: Start Docker Desktop, wait until it says running, then rerun .\bin\agentos.exe doctor --support."
        if ($dockerOutput) {
            Write-Host "detail: $dockerOutput"
        }
        exit 2
    }
}
function Invoke-AgentOS {
    & $agentos @args
    if ($LASTEXITCODE -ne 0) {
        throw "agentos command failed: $($args -join ' ')"
    }
}

Assert-DockerReady

Write-Step "building AgentOS binary"
& (Join-Path $PSScriptRoot "Build.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Step "running security audit"
& (Join-Path $PSScriptRoot "SecurityAudit.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Step "building pay-ready worker image"
$dockerfile = Join-Path $root "examples\pay-ready\Dockerfile"
docker build -f $dockerfile -t agentos/pay-ready-demo:local $root
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

New-Item -ItemType Directory -Force $stateHome, $internalDir, $webDir, $outputs | Out-Null
foreach ($path in @(
    (Join-Path $internalDir "backend_fix.txt"),
    (Join-Path $webDir "app.js"),
    $auditPath
)) {
    Assert-UnderRoot $root $path
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Force
    }
}

$env:AGENTOS_HOME = $stateHome
$env:AGENTOS_ADDR = $Address
if ([string]::IsNullOrWhiteSpace($env:AGENTOS_APPROVER_TOKEN)) {
    $env:AGENTOS_APPROVER_TOKEN = [guid]::NewGuid().ToString("N") + [guid]::NewGuid().ToString("N")
}

Write-Step "validating manifest"
Invoke-AgentOS validate (Join-Path $root "examples\pay-ready\agent-process.yaml")

Write-Step "starting isolated daemon on $Address"
$daemon = Start-Process -FilePath $agentos `
    -ArgumentList @("serve", "--addr", $Address) `
    -WorkingDirectory $root `
    -WindowStyle Hidden `
    -PassThru `
    -RedirectStandardOutput $daemonOut `
    -RedirectStandardError $daemonErr

try {
    Start-Sleep -Seconds 2

    Write-Step "creating managed process"
    $runOutput = Invoke-AgentOS run (Join-Path $root "examples\pay-ready\agent-process.yaml")
    $process = $runOutput | ConvertFrom-Json
    $processID = $process.id
    if ([string]::IsNullOrWhiteSpace($processID)) {
        throw "Run response did not include a process id: $runOutput"
    }
    Write-Step "process id $processID"

    $approvalID = ""
    for ($i = 0; $i -lt 60; $i++) {
        Start-Sleep -Milliseconds 500
        $approvalOutput = Invoke-AgentOS approvals
        $approvals = @($approvalOutput | ConvertFrom-Json)
        $pending = @($approvals | Where-Object { $_.process_id -eq $processID -and $_.status -eq "pending" })
        if ($pending.Count -gt 0) {
            $approvalID = $pending[0].id
            break
        }
    }
    if ([string]::IsNullOrWhiteSpace($approvalID)) {
        throw "No approval was created for the backend write."
    }

    Write-Step "approving backend-only write"
    Invoke-AgentOS approve $approvalID "pay-ready demo backend-only approval" | Out-Null

    $state = ""
    $inspection = $null
    for ($i = 0; $i -lt 80; $i++) {
        Start-Sleep -Milliseconds 500
        $inspectionOutput = Invoke-AgentOS inspect $processID
        $inspection = $inspectionOutput | ConvertFrom-Json
        $state = $inspection.state
        if ($state -in @("succeeded", "failed", "cancelled")) {
            break
        }
    }
    if ($state -ne "succeeded") {
        throw "Process finished in unexpected state '$state'."
    }

    Write-Step "exporting replay and audit evidence"
    $replay = Invoke-AgentOS replay $processID | ConvertFrom-Json
    Invoke-AgentOS audit $processID $auditPath | Out-Null
    $logs = @(Invoke-AgentOS logs $processID | ConvertFrom-Json)

    $denied = @($logs | Where-Object { $_.type -eq "tool.denied" })
    $usage = @($logs | Where-Object { $_.type -eq "budget.usage_updated" })
    $allowedFile = Join-Path $internalDir "backend_fix.txt"
    $forbiddenFile = Join-Path $webDir "app.js"

    if ($replay.state -ne "succeeded") {
        throw "Replay did not reconstruct succeeded state."
    }
    if ($inspection.usage.tokens -le 0 -or $inspection.usage.cost_usd -le 0) {
        throw "Usage accounting did not produce nonzero tokens and cost."
    }
    if ($usage.Count -lt 1) {
        throw "Missing budget.usage_updated event."
    }
    if ($denied.Count -lt 1) {
        throw "Missing tool.denied event."
    }
    if (-not (Test-Path -LiteralPath $allowedFile)) {
        throw "Approved backend artifact was not created."
    }
    if (Test-Path -LiteralPath $forbiddenFile) {
        throw "Forbidden frontend artifact was created."
    }

    Write-Host ""
    Write-Host "Pay-ready demo passed"
    Write-Host "Process: $processID"
    Write-Host "State: $state"
    Write-Host ("Usage: {0} tokens, `${1:N4}" -f $inspection.usage.tokens, $inspection.usage.cost_usd)
    Write-Host "Denied actions: $($denied.Count)"
    Write-Host "Approved artifact: $allowedFile"
    Write-Host "Audit bundle: $auditPath"
}
finally {
    if (-not $KeepRunning -and $daemon -and -not $daemon.HasExited) {
        Stop-Process -Id $daemon.Id
        Wait-Process -Id $daemon.Id -ErrorAction SilentlyContinue
    }
    if ($KeepRunning) {
        Write-Step "daemon left running on $Address"
    }
}