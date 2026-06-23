$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$goCommand = Get-Command go -ErrorAction SilentlyContinue
if ($goCommand) {
    $goExe = $goCommand.Source
} else {
    $localGo = Join-Path $root "work/toolchain/go/bin/go.exe"
    if (-not (Test-Path $localGo)) {
        throw "Go was not found. Install Go 1.24+ or place the portable toolchain at work/toolchain/go."
    }
    $goExe = $localGo
}

$env:GOTOOLCHAIN = "local"
$env:GOCACHE = Join-Path $root "work/gocache"
$env:GOMODCACHE = Join-Path $root "work/gomodcache"
Push-Location $root
try {
    & $goExe test ./cmd/... ./internal/...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    & $goExe vet ./cmd/... ./internal/...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    $pythonExe = $env:AGENTOS_PYTHON
    $pythonPrefix = @()
    if (-not $pythonExe) {
        $python = Get-Command python -ErrorAction SilentlyContinue
        if ($python -and $python.Source -notmatch '\\Microsoft\\WindowsApps\\') {
            $candidateVersion = & $python.Source -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')" 2>$null
            if ($LASTEXITCODE -eq 0 -and $candidateVersion) {
                $pythonExe = $python.Source
            }
        }
        if (-not $pythonExe) {
            $py = Get-Command py -ErrorAction SilentlyContinue
            if ($py) {
                $candidateVersion = & $py.Source -3 -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')" 2>$null
                if ($LASTEXITCODE -eq 0 -and $candidateVersion) {
                    $pythonExe = $py.Source
                    $pythonPrefix = @("-3")
                }
            }
        }
    }
    if ($pythonExe) {
        $version = & $pythonExe @pythonPrefix -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')"
        $parts = $version.Trim().Split(".")
        if ([int]$parts[0] -lt 3 -or ([int]$parts[0] -eq 3 -and [int]$parts[1] -lt 11)) {
            throw "Python 3.11+ is required for adapter tests; found $version"
        }
        $env:PYTHONPATH = Join-Path $root "adapters/agents-sdk/src"
        & $pythonExe @pythonPrefix -m unittest discover -s adapters/agents-sdk/tests
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } else {
        Write-Warning "Python 3.11+ was not found; adapter tests were skipped. Set AGENTOS_PYTHON to run them."
    }
}
finally {
    Pop-Location
}
