param(
    [string]$BranchName = "agentos/pay-ready-artifact",
    [string]$Address = "127.0.0.1:17481",
    [switch]$KeepBranch
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$outputs = Join-Path $root "outputs"
$report = Join-Path $outputs "github-artifact-report.json"
$diffPath = Join-Path $outputs "github-artifact.diff"
$testPath = Join-Path $outputs "github-artifact-test.txt"

function Write-Step([string]$message) {
    Write-Host "[github-artifact] $message"
}

function Invoke-Git([string[]]$Arguments) {
    & git -C $root @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "git command failed: git $($Arguments -join ' ')"
    }
}

New-Item -ItemType Directory -Force $outputs | Out-Null

Write-Step "checking repository state"
$status = (& git -C $root status --porcelain)
if ($status) {
    throw "Working tree must be clean before creating a GitHub artifact branch. Commit or stash changes first."
}

$currentBranch = (& git -C $root branch --show-current).Trim()
if ([string]::IsNullOrWhiteSpace($currentBranch)) {
    throw "Could not determine current branch."
}

Write-Step "creating local artifact branch $BranchName"
Invoke-Git @("switch", "-C", $BranchName)

try {
    Write-Step "running pay-ready proof"
    & (Join-Path $PSScriptRoot "Demo-PayReady.ps1") -Address $Address
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Step "capturing diff and test output"
    & git -C $root diff -- . | Set-Content -Path $diffPath -Encoding utf8
    if ($LASTEXITCODE -ne 0) { throw "git diff failed" }

    & (Join-Path $PSScriptRoot "Test.ps1") *>&1 | Tee-Object -FilePath $testPath
    if ($LASTEXITCODE -ne 0) { throw "test suite failed; see $testPath" }

    $auditPath = Join-Path $outputs "pay-ready-audit.json"
    $summary = [ordered]@{
        schema_version = "agentos-github-artifact/v1"
        generated_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        base_branch = $currentBranch
        artifact_branch = $BranchName
        diff_path = $diffPath
        test_output_path = $testPath
        audit_bundle_path = $auditPath
        note = "Local GitHub-oriented artifact proof. Push branch or open PR manually after review."
    }
    $summary | ConvertTo-Json -Depth 6 | Set-Content -Path $report -Encoding utf8

    Write-Host ""
    Write-Host "GitHub artifact proof prepared"
    Write-Host "Branch: $BranchName"
    Write-Host "Diff: $diffPath"
    Write-Host "Tests: $testPath"
    Write-Host "Audit: $auditPath"
    Write-Host "Report: $report"
}
finally {
    if (-not $KeepBranch) {
        Write-Step "returning to $currentBranch"
        Invoke-Git @("switch", $currentBranch)
    }
}