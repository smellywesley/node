param(
    [int]$Count = 4,
    [int]$MaxParallel = 2,
    [string]$Address = "127.0.0.1:17481",
    [switch]$SkipBuild,
    [switch]$KeepRunning
)

$ErrorActionPreference = "Stop"

if ($Count -lt 1) { throw "Count must be at least 1." }
if ($MaxParallel -lt 1) { throw "MaxParallel must be at least 1." }

$root = Split-Path -Parent $PSScriptRoot
$agentos = Join-Path $root "bin\agentos.exe"
$manifest = Join-Path $root "examples\smoke\agent-process.yaml"
$stateHome = Join-Path $root "work\backend-load-home"
$outputs = Join-Path $root "outputs"
$reportPath = Join-Path $outputs "backend-load-report.json"
$daemonOut = Join-Path $stateHome "daemon.out"
$daemonErr = Join-Path $stateHome "daemon.err"

function Write-Step([string]$message) {
    Write-Host "[backend-load] $message"
}

function Assert-DockerReady {
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if (-not $docker) {
        Write-Host "problem: Docker is not installed or not on PATH."
        Write-Host "cause: backend load evidence runs AgentOS workers in OCI-compatible containers."
        Write-Host "fix: Install Docker Desktop or another Docker-compatible engine, start it, then rerun this script."
        exit 2
    }

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $dockerOutput = & docker version --format '{{.Server.Version}}' 2>&1
        $dockerExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }

    if ($dockerExitCode -ne 0) {
        Write-Host "problem: Docker is installed but the engine is not reachable."
        Write-Host "cause: Docker Desktop is stopped, still starting, or using an unavailable engine pipe."
        Write-Host "fix: Start Docker, wait until it says running, then rerun .\bin\agentos.exe doctor --support."
        if ($dockerOutput) { Write-Host "detail: $dockerOutput" }
        exit 2
    }
}

function New-LoadJob {
    param([int]$Index)

    Start-Job -ArgumentList $Index, $agentos, $manifest, $Address, $stateHome -ScriptBlock {
        param($Index, $AgentOS, $Manifest, $Address, $StateHome)

        $ErrorActionPreference = "Stop"
        $env:AGENTOS_ADDR = $Address
        $env:AGENTOS_HOME = $StateHome

        $started = Get-Date
        try {
            $runOutput = & $AgentOS run $Manifest
            if ($LASTEXITCODE -ne 0) { throw "agentos run failed with exit code $LASTEXITCODE" }
            $process = $runOutput | ConvertFrom-Json
            $processID = $process.id
            if ([string]::IsNullOrWhiteSpace($processID)) {
                throw "Run response did not include a process id: $runOutput"
            }

            $state = ""
            $inspection = $null
            for ($i = 0; $i -lt 120; $i++) {
                Start-Sleep -Milliseconds 500
                $inspectionOutput = & $AgentOS inspect $processID
                if ($LASTEXITCODE -ne 0) { throw "agentos inspect failed with exit code $LASTEXITCODE" }
                $inspection = $inspectionOutput | ConvertFrom-Json
                $state = $inspection.state
                if ($state -in @("succeeded", "failed", "cancelled")) { break }
            }

            $ended = Get-Date
            [pscustomobject]@{
                index = $Index
                process_id = $processID
                state = $state
                duration_ms = [int][math]::Round(($ended - $started).TotalMilliseconds)
                tokens = if ($inspection -and $inspection.usage) { [int]$inspection.usage.tokens } else { 0 }
                cost_usd = if ($inspection -and $inspection.usage) { [double]$inspection.usage.cost_usd } else { 0 }
                error = $null
            }
        } catch {
            $ended = Get-Date
            [pscustomobject]@{
                index = $Index
                process_id = $null
                state = "error"
                duration_ms = [int][math]::Round(($ended - $started).TotalMilliseconds)
                tokens = 0
                cost_usd = 0
                error = $_.Exception.Message
            }
        }
    }
}

Assert-DockerReady

if (-not $SkipBuild) {
    Write-Step "building AgentOS binary"
    & (Join-Path $PSScriptRoot "Build.ps1")
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Step "building smoke worker image"
$dockerfile = Join-Path $root "examples\smoke\Dockerfile"
docker build -f $dockerfile -t agentos/protocol-smoke:local $root
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

New-Item -ItemType Directory -Force $stateHome, $outputs | Out-Null
$env:AGENTOS_HOME = $stateHome
$env:AGENTOS_ADDR = $Address

Write-Step "starting isolated daemon on $Address"
$daemon = Start-Process -FilePath $agentos `
    -ArgumentList @("serve", "--addr", $Address) `
    -WorkingDirectory $root `
    -WindowStyle Hidden `
    -PassThru `
    -RedirectStandardOutput $daemonOut `
    -RedirectStandardError $daemonErr

$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
$jobs = @()
$results = @()

try {
    Start-Sleep -Seconds 2

    Write-Step "running $Count smoke processes with max parallelism $MaxParallel"
    $next = 1
    while ($next -le $Count -or (@($jobs | Where-Object { $_.State -eq "Running" }).Count -gt 0)) {
        while ($next -le $Count -and (@($jobs | Where-Object { $_.State -eq "Running" }).Count -lt $MaxParallel)) {
            $jobs += New-LoadJob -Index $next
            $next++
        }

        $finished = Wait-Job -Job $jobs -Any -Timeout 1
        if ($finished) {
            foreach ($job in @($finished)) {
                $results += Receive-Job -Job $job
                Remove-Job -Job $job
                $jobs = @($jobs | Where-Object { $_.Id -ne $job.Id })
            }
        }
    }
} finally {
    foreach ($job in @($jobs)) {
        Stop-Job -Job $job -ErrorAction SilentlyContinue
        Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
    }
    if (-not $KeepRunning -and $daemon -and -not $daemon.HasExited) {
        Stop-Process -Id $daemon.Id
        Wait-Process -Id $daemon.Id -ErrorAction SilentlyContinue
    }
    if ($KeepRunning) {
        Write-Step "daemon left running on $Address"
    }
}

$stopwatch.Stop()

$terminal = @($results | Where-Object { $_.state -in @("succeeded", "failed", "cancelled") })
$succeeded = @($results | Where-Object { $_.state -eq "succeeded" })
$failed = @($results | Where-Object { $_.state -ne "succeeded" })
$durations = @($results | ForEach-Object { $_.duration_ms })

$report = [pscustomobject]@{
    generated_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    scenario = "protocol-smoke bounded concurrency"
    address = $Address
    requested_runs = $Count
    max_parallel = $MaxParallel
    elapsed_ms = [int][math]::Round($stopwatch.Elapsed.TotalMilliseconds)
    counts = [pscustomobject]@{
        terminal = $terminal.Count
        succeeded = $succeeded.Count
        failed = $failed.Count
    }
    duration_ms = [pscustomobject]@{
        min = if ($durations.Count) { ($durations | Measure-Object -Minimum).Minimum } else { 0 }
        max = if ($durations.Count) { ($durations | Measure-Object -Maximum).Maximum } else { 0 }
        avg = if ($durations.Count) { [int][math]::Round(($durations | Measure-Object -Average).Average) } else { 0 }
    }
    results = @($results | Sort-Object index)
}

$report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $reportPath -Encoding UTF8

Write-Host ""
Write-Host "Backend load report written: $reportPath"
Write-Host "Succeeded: $($succeeded.Count)/$Count"
Write-Host "Elapsed: $($report.elapsed_ms) ms"

if ($failed.Count -gt 0 -or $terminal.Count -ne $Count) {
    exit 1
}
