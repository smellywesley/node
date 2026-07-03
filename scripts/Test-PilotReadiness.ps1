param(
    [string]$Output = "outputs\pilot-readiness-report.json",
    [switch]$NoWrite,
    [switch]$FailOnBlockers
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$checks = @()

function Resolve-RepoPath([string]$Path) {
    if ([System.IO.Path]::IsPathRooted($Path)) { return $Path }
    return Join-Path $root $Path
}

function Add-ReadinessCheck {
    param(
        [string]$Area,
        [string]$Name,
        [bool]$Pass,
        [string]$Evidence,
        [string]$Fix,
        [bool]$Required = $true
    )

    $script:checks += [pscustomobject]@{
        area = $Area
        name = $Name
        required = $Required
        pass = $Pass
        evidence = $Evidence
        fix = $Fix
    }
}

function Test-FileContains([string]$RelativePath, [string]$Pattern) {
    $path = Resolve-RepoPath $RelativePath
    if (-not (Test-Path -LiteralPath $path)) { return $false }
    return [bool](Select-String -LiteralPath $path -Pattern $Pattern -Quiet)
}

function Get-DockerReadiness {
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if (-not $docker) {
        return [pscustomobject]@{
            pass = $false
            evidence = "Docker executable not found on PATH."
            fix = "Install Docker Desktop or another Docker-compatible engine, start it, then rerun this audit."
        }
    }

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $dockerOutput = & docker version --format '{{.Server.Version}}' 2>&1
        $dockerExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }

    if ($dockerExitCode -eq 0) {
        return [pscustomobject]@{
            pass = $true
            evidence = "Docker engine reachable; server version $dockerOutput."
            fix = ""
        }
    }

    return [pscustomobject]@{
        pass = $false
        evidence = "Docker installed but engine unreachable: $dockerOutput"
        fix = "Start Docker Desktop, wait until it is running, then rerun .\bin\agentos.exe doctor --support."
    }
}

function Test-PaymentConfig {
    $paymentPath = Resolve-RepoPath "deploy\public-site\public\payment-links.js"
    if (-not (Test-Path -LiteralPath $paymentPath)) {
        return [pscustomobject]@{
            pass = $false
            evidence = "deploy\public-site\public\payment-links.js is missing."
            fix = "Run npm run configure:cta in deploy\public-site with NODE_PUBLIC_CONTACT_EMAIL or NODE_PUBLIC_PILOT_PAYMENT_LINK set."
        }
    }

    $raw = Get-Content -LiteralPath $paymentPath -Raw
    $hasContactEmail = $raw -match '"contactEmail"\s*:\s*"[^"]+@[^"]+\.[^"]+"'
    $hasStripeLink = $raw -match '"(pilot|pro|enterprise)"\s*:\s*"https://buy\.stripe\.com/[^"]+"'
    $configured = $hasContactEmail -or $hasStripeLink
    $evidence = "contact_email_configured=$hasContactEmail; stripe_payment_link_configured=$hasStripeLink"

    return [pscustomobject]@{
        pass = $configured
        evidence = $evidence
        fix = "Set NODE_PUBLIC_CONTACT_EMAIL and/or NODE_PUBLIC_PILOT_PAYMENT_LINK, then run npm run configure:cta and npm run test:cta before deploying."
    }
}

function Test-ProofPacket {
    $proofPath = Resolve-RepoPath "outputs\pay-ready-proof.md"
    if (-not (Test-Path -LiteralPath $proofPath)) {
        return [pscustomobject]@{
            pass = $false
            evidence = "outputs\pay-ready-proof.md is missing."
            fix = "Start Docker and run .\scripts\new-pay-ready-proof-packet.cmd after a fresh pay-ready demo."
        }
    }

    $passes = Test-FileContains "outputs\pay-ready-proof.md" '^Status:\s+PASS'
    return [pscustomobject]@{
        pass = $passes
        evidence = if ($passes) { "Proof packet exists and reports Status: PASS." } else { "Proof packet exists but does not report Status: PASS." }
        fix = "Rerun .\scripts\new-pay-ready-proof-packet.cmd with Docker running and fix any failed proof checks."
    }
}

function Test-BackendLoadReport {
    $reportPath = Resolve-RepoPath "outputs\backend-load-report.json"
    if (-not (Test-Path -LiteralPath $reportPath)) {
        return [pscustomobject]@{
            pass = $false
            evidence = "outputs\backend-load-report.json is missing."
            fix = "Start Docker and run .\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2."
        }
    }

    try {
        $report = Get-Content -LiteralPath $reportPath -Raw | ConvertFrom-Json
        $requested = [int]$report.requested_runs
        $succeeded = [int]$report.counts.succeeded
        $failed = [int]$report.counts.failed
        $passes = ($requested -gt 0 -and $succeeded -eq $requested -and $failed -eq 0)
        return [pscustomobject]@{
            pass = $passes
            evidence = "requested_runs=$requested; succeeded=$succeeded; failed=$failed"
            fix = "Rerun .\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2 with Docker running and investigate failed runs."
        }
    } catch {
        return [pscustomobject]@{
            pass = $false
            evidence = "Could not parse backend load report: $($_.Exception.Message)"
            fix = "Delete the stale report and rerun .\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2."
        }
    }
}

$paymentConfig = Test-PaymentConfig
Add-ReadinessCheck "Public CTA" "Real contact email or Stripe pilot link configured" $paymentConfig.pass $paymentConfig.evidence $paymentConfig.fix

$ctaGuard = (Test-FileContains "deploy\public-site\package.json" '"test:cta"') -and
    (Test-FileContains "render.yaml" 'configure:cta.*test:cta.*build')
Add-ReadinessCheck "Public CTA" "Render build blocks unconfigured CTAs" $ctaGuard "package test:cta and Render configure/test/build chain present=$ctaGuard" "Keep npm run configure:cta and npm run test:cta in the Render build command."

$macLinuxScripts = (Test-Path -LiteralPath (Resolve-RepoPath "scripts\build.sh")) -and
    (Test-Path -LiteralPath (Resolve-RepoPath "scripts\demo-pay-ready.sh")) -and
    (Test-Path -LiteralPath (Resolve-RepoPath "scripts\security-audit.sh"))
Add-ReadinessCheck "Buyer Demo" "Mac/Linux demo scripts exist" $macLinuxScripts "build.sh, demo-pay-ready.sh, and security-audit.sh present=$macLinuxScripts" "Add or restore Linux/macOS shell wrappers for the buyer demo."

$proofScript = Test-Path -LiteralPath (Resolve-RepoPath "scripts\New-PayReadyProofPacket.ps1")
Add-ReadinessCheck "Buyer Demo" "Proof packet generator exists" $proofScript "scripts\New-PayReadyProofPacket.ps1 present=$proofScript" "Restore scripts\New-PayReadyProofPacket.ps1 and scripts\new-pay-ready-proof-packet.cmd."

$proofPacket = Test-ProofPacket
Add-ReadinessCheck "Buyer Demo" "Fresh pay-ready proof packet passes" $proofPacket.pass $proofPacket.evidence $proofPacket.fix

$docker = Get-DockerReadiness
Add-ReadinessCheck "Backend Runtime" "Docker engine reachable for local agent runs" $docker.pass $docker.evidence $docker.fix

$loadScript = Test-Path -LiteralPath (Resolve-RepoPath "scripts\Measure-BackendLoad.ps1")
Add-ReadinessCheck "Backend Runtime" "Backend load script exists" $loadScript "scripts\Measure-BackendLoad.ps1 present=$loadScript" "Restore scripts\Measure-BackendLoad.ps1 and scripts\measure-backend-load.cmd."

$loadReport = Test-BackendLoadReport
Add-ReadinessCheck "Backend Runtime" "Backend load report has all runs succeeded" $loadReport.pass $loadReport.evidence $loadReport.fix

$threatModel = Test-Path -LiteralPath (Resolve-RepoPath "docs\security-threat-model.md")
Add-ReadinessCheck "Security" "Threat model is documented" $threatModel "docs\security-threat-model.md present=$threatModel" "Restore the local-first threat model and hosted-readiness blockers."

$raceCi = Test-FileContains ".github\workflows\ci.yml" 'go test -race ./internal/core ./internal/store ./internal/api'
Add-ReadinessCheck "Security" "Race-sensitive backend packages run in CI" $raceCi "CI race step present=$raceCi" "Keep go test -race for internal/core, internal/store, and internal/api on Linux CI."

$playbook = Test-Path -LiteralPath (Resolve-RepoPath "docs\design-partner-pilot-playbook.md")
Add-ReadinessCheck "Sales" "Design partner pilot playbook exists" $playbook "docs\design-partner-pilot-playbook.md present=$playbook" "Restore the pilot playbook before selling design-partner pilots."

$requiredBlockers = @($checks | Where-Object { $_.required -and -not $_.pass })
$optionalBlockers = @($checks | Where-Object { -not $_.required -and -not $_.pass })
$status = if ($requiredBlockers.Count -eq 0) { "READY" } else { "BLOCKED" }
$generatedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$report = [pscustomobject]@{
    generated_at = $generatedAt
    status = $status
    required_blockers = $requiredBlockers.Count
    optional_blockers = $optionalBlockers.Count
    checks = $checks
}

Write-Host ""
Write-Host "Pilot readiness: $status"
Write-Host "Required blockers: $($requiredBlockers.Count)"
Write-Host ""
$checks |
    Select-Object area, name, required, pass, evidence, fix |
    Format-Table -AutoSize -Wrap

if (-not $NoWrite) {
    $outputPath = Resolve-RepoPath $Output
    $outputDir = Split-Path -Parent $outputPath
    if ($outputDir) {
        New-Item -ItemType Directory -Force $outputDir | Out-Null
    }
    $report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $outputPath -Encoding UTF8
    Write-Host ""
    Write-Host "Pilot readiness report written: $outputPath"
}

if ($FailOnBlockers -and $requiredBlockers.Count -gt 0) {
    exit 1
}
