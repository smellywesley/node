param(
    [string]$Image = "agentos/agents-sdk-coding:local",
    [string]$Output = "dist/agentos-agents-sdk-coding-local.tar.gz"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$outputPath = Join-Path $root $Output
$temporaryTar = Join-Path $root "work/agentos-agents-sdk-coding-local.tar"

New-Item -ItemType Directory -Force (Split-Path -Parent $outputPath) | Out-Null
New-Item -ItemType Directory -Force (Split-Path -Parent $temporaryTar) | Out-Null

try {
    docker image inspect $Image *> $null
    if ($LASTEXITCODE -ne 0) {
        throw "Docker image $Image is not available locally."
    }
    docker save --output $temporaryTar $Image
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    $input = [IO.File]::OpenRead($temporaryTar)
    try {
        $outputStream = [IO.File]::Create($outputPath)
        try {
            $gzip = [IO.Compression.GZipStream]::new(
                $outputStream,
                [IO.Compression.CompressionLevel]::Optimal
            )
            try {
                $input.CopyTo($gzip)
            }
            finally {
                $gzip.Dispose()
            }
        }
        finally {
            $outputStream.Dispose()
        }
    }
    finally {
        $input.Dispose()
    }
}
finally {
    if (Test-Path -LiteralPath $temporaryTar) {
        Remove-Item -LiteralPath $temporaryTar -Force
    }
}

$hash = Get-FileHash -Algorithm SHA256 -LiteralPath $outputPath
Set-Content -LiteralPath "$outputPath.sha256" -Value (
    "$($hash.Hash.ToLower())  $([IO.Path]::GetFileName($outputPath))"
)
Write-Host "Packaged $outputPath"
