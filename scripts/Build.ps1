param(
    [string]$Output = "bin/agentos.exe",
    [string]$Version = "dev",
    [string]$Commit = "",
    [string]$BuiltAt = ""
)

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

if (-not $Commit) {
    if (Test-Path -LiteralPath (Join-Path $root ".git")) {
        $Commit = (& git -C $root rev-parse --short HEAD 2>$null)
    }
    if (-not $Commit) { $Commit = "unknown" }
}
if (-not $BuiltAt) {
    $BuiltAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}

$outputPath = Join-Path $root $Output
New-Item -ItemType Directory -Force (Split-Path -Parent $outputPath) | Out-Null
$env:GOTOOLCHAIN = "local"
$env:GOCACHE = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $root "work/gocache" }
$env:GOMODCACHE = if ($env:GOMODCACHE) { $env:GOMODCACHE } else { Join-Path $root "work/gomodcache" }
$ldflags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.builtAt=$BuiltAt"
& $goExe build -buildvcs=false -trimpath -ldflags $ldflags -o $outputPath ./cmd/agentos
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Built $outputPath"

