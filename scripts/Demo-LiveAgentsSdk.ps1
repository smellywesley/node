param(
    [string]$Address = "127.0.0.1:17480",
    [switch]$KeepRunning
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$agentos = Join-Path $root "bin\agentos.exe"
$stateHome = Join-Path $root "work\agents-sdk-live-home"
$workspace = Join-Path $root "work\agents-sdk-live-workspace"
$reviewedDir = Join-Path $workspace "reviewed"
$outputs = Join-Path $root "outputs"
$auditPath = Join-Path $outputs "agents-sdk-live-audit.json"
$daemonOut = Join-Path $stateHome "daemon.out"
$daemonErr = Join-Path $stateHome "daemon.err"

function Write-Step([string]$message) {
    Write-Host "[live-sdk] $message"
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

if ([string]::IsNullOrWhiteSpace($env:OPENAI_API_KEY)) {
    Write-Host "OPENAI_API_KEY is not set. This live SDK demo is intentionally not run without an explicit provider key."
    Write-Host "Set OPENAI_API_KEY and rerun .\scripts\demo-live-agents-sdk.cmd when you are ready to spend a small amount of API credit."
    exit 2
}

Assert-DockerReady

Write-Step "building AgentOS binary"
& (Join-Path $PSScriptRoot "Build.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Step "running security audit"
& (Join-Path $PSScriptRoot "SecurityAudit.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Step "building live OpenAI Agents SDK worker image"
$dockerfile = Join-Path $root "examples\agents-sdk-live-coding\Dockerfile"
docker build -f $dockerfile -t agentos/openai-agents-live-coding:local $root
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

New-Item -ItemType Directory -Force $stateHome, $reviewedDir, $outputs | Out-Null
foreach ($path in @(
    (Join-Path $reviewedDir "live_add.go"),
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
Invoke-AgentOS validate (Join-Path $root "examples\agents-sdk-live-coding\agent-process.yaml")

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

    Write-Step "creating provider-backed managed process"
    $runOutput = Invoke-AgentOS run (Join-Path $root "examples\agents-sdk-live-coding\agent-process.yaml")
    $process = $runOutput | ConvertFrom-Json
    $processID = $process.id
    if ([string]::IsNullOrWhiteSpace($processID)) {
        throw "Run response did not include a process id: $runOutput"
    }
    Write-Step "process id $processID"

    $approvalID = ""
    for ($i = 0; $i -lt 240; $i++) {
        Start-Sleep -Milliseconds 500
        $inspectionOutput = Invoke-AgentOS inspect $processID
        $inspection = $inspectionOutput | ConvertFrom-Json
        if ($inspection.state -in @("failed", "cancelled")) {
            throw "Process ended before approval in state '$($inspection.state)': $($inspection.error)"
        }
        $approvalOutput = Invoke-AgentOS approvals
        $approvals = @($approvalOutput | ConvertFrom-Json)
        $pending = @($approvals | Where-Object { $_.process_id -eq $processID -and $_.status -eq "pending" })
        if ($pending.Count -gt 0) {
            $approvalID = $pending[0].id
            break
        }
    }
    if ([string]::IsNullOrWhiteSpace($approvalID)) {
        throw "No approval was created for the reviewed artifact write."
    }

    Write-Step "approving reviewed artifact write"
    Invoke-AgentOS approve $approvalID "live SDK reviewed artifact approval" | Out-Null

    $state = ""
    $inspection = $null
    for ($i = 0; $i -lt 120; $i++) {
        Start-Sleep -Milliseconds 500
        $inspectionOutput = Invoke-AgentOS inspect $processID
        $inspection = $inspectionOutput | ConvertFrom-Json
        $state = $inspection.state
        if ($state -in @("succeeded", "failed", "cancelled")) {
            break
        }
    }
    if ($state -ne "succeeded") {
        throw "Process finished in unexpected state '$state': $($inspection.error)"
    }

    Write-Step "exporting replay and audit evidence"
    $replay = Invoke-AgentOS replay $processID | ConvertFrom-Json
    Invoke-AgentOS audit $processID $auditPath | Out-Null
    $logs = @(Invoke-AgentOS logs $processID | ConvertFrom-Json)

    $usage = @($logs | Where-Object { $_.type -eq "budget.usage_updated" })
    $artifact = Join-Path $reviewedDir "live_add.go"

    if ($replay.state -ne "succeeded") {
        throw "Replay did not reconstruct succeeded state."
    }
    if ($inspection.usage.tokens -le 0 -or $inspection.usage.cost_usd -le 0) {
        throw "Live usage accounting did not produce nonzero tokens and cost."
    }
    if ($usage.Count -lt 1) {
        throw "Missing budget.usage_updated event."
    }
    if (-not (Test-Path -LiteralPath $artifact)) {
        throw "Reviewed live SDK artifact was not created."
    }

    Write-Host ""
    Write-Host "Live OpenAI Agents SDK demo passed"
    Write-Host "Process: $processID"
    Write-Host "State: $state"
    Write-Host ("Usage: {0} tokens, `${1:N6}" -f $inspection.usage.tokens, $inspection.usage.cost_usd)
    Write-Host "Artifact: $artifact"
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