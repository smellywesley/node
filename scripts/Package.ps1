param(
    [string]$Version = "v1"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$name = "agentos-$Version-windows-amd64"
$dist = Join-Path $root "dist"
$stage = Join-Path $dist $name
$archive = Join-Path $dist "$name.zip"

& (Join-Path $PSScriptRoot "Build.ps1") -Version $Version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if (Test-Path -LiteralPath $stage) {
    Remove-Item -LiteralPath $stage -Recurse -Force
}
New-Item -ItemType Directory -Force $stage | Out-Null

$files = @(
    "README.md",
    "AGENTS.md",
    ".gitignore",
    "go.mod",
    "go.sum",
    "LICENSE",
    "CHANGELOG.md",
    "CONTRIBUTING.md",
    "SECURITY.md"
)
$directories = @(
    ".Codex/commands",
    "adapters",
    "agents",
    "cmd",
    "data/templates",
    "docs",
    "examples",
    "internal",
    "scripts"
)

foreach ($file in $files) {
    $source = Join-Path $root $file
    if (Test-Path -LiteralPath $source) {
        Copy-Item -LiteralPath $source -Destination $stage
    }
}
foreach ($directory in $directories) {
    $source = Join-Path $root $directory
    if (Test-Path -LiteralPath $source) {
        $destination = Join-Path $stage $directory
        New-Item -ItemType Directory -Force (Split-Path -Parent $destination) | Out-Null
        Copy-Item -LiteralPath $source -Destination $destination -Recurse
    }
}
Get-ChildItem -LiteralPath $stage -Recurse -Directory -Filter "__pycache__" |
    Remove-Item -Recurse -Force
Get-ChildItem -LiteralPath $stage -Recurse -File -Filter "*.pyc" |
    Remove-Item -Force
New-Item -ItemType Directory -Force (Join-Path $stage "bin") | Out-Null
Copy-Item -LiteralPath (Join-Path $root "bin/agentos.exe") -Destination (Join-Path $stage "bin/agentos.exe")

$forbidden = Get-ChildItem -LiteralPath $stage -Recurse -Force | Where-Object {
    $relative = $_.FullName.Substring($stage.Length).TrimStart('\', '/').Replace('\', '/')
    $relative -match '(^|/)(work|dist|outputs|\.git|\.gstack|\.gocache)(/|$)' -or
    ($relative -match '(^|/)\.Codex(/|$)' -and $relative -notmatch '(^|/)\.Codex($|/commands(/|$))') -or
    $relative -match '(^|/)(__pycache__)(/|$)' -or
    $relative -match '(^|/)(token|approver-token)$' -or
    $relative -match '(agentos\.db|agentos\.db-wal|agentos\.db-shm|daemon\.(out|err|pid)|\.pyc)$'
}
if ($forbidden) {
    $list = ($forbidden | Select-Object -First 20 | ForEach-Object { $_.FullName }) -join [Environment]::NewLine
    throw "Package contains forbidden runtime or local-only files:$([Environment]::NewLine)$list"
}

if (Test-Path -LiteralPath $archive) {
    Remove-Item -LiteralPath $archive -Force
}
Compress-Archive -LiteralPath $stage -DestinationPath $archive -CompressionLevel Optimal
$hash = Get-FileHash -Algorithm SHA256 -LiteralPath $archive
Set-Content -LiteralPath "$archive.sha256" -Value "$($hash.Hash.ToLower())  $([IO.Path]::GetFileName($archive))"
Write-Host "Packaged $archive"
