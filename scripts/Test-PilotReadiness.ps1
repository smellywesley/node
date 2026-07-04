param(
    [string]$Output = "outputs\pilot-readiness-report.json",
    [switch]$NoWrite,
    [switch]$AllowBlockers,
    [switch]$FailOnBlockers
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$checks = @()
$generatedPaymentConfigPath = ""

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
    $generatedDir = Join-Path ([System.IO.Path]::GetTempPath()) "node-pilot-readiness"
    $generatedPath = Join-Path $generatedDir "payment-links.generated.js"
    $script:generatedPaymentConfigPath = $generatedPath
    New-Item -ItemType Directory -Force $generatedDir | Out-Null

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $configureOutput = & node "scripts\write-payment-config.mjs" --output $generatedPath 2>&1
        $configureExit = $LASTEXITCODE
        $checkOutput = if ($configureExit -eq 0) { & node "scripts\check-cta-config.mjs" $generatedPath 2>&1 } else { @() }
        $checkExit = if ($configureExit -eq 0) { $LASTEXITCODE } else { 1 }
    } finally {
        $ErrorActionPreference = $previous
    }

    if ($configureExit -ne 0) {
        return [pscustomobject]@{
            pass = $false
            evidence = "generated CTA config failed: $configureOutput"
            fix = "Set only reviewed public CTA env vars in the hosting provider, then rerun npm run configure:cta."
        }
    }

    $raw = Get-Content -LiteralPath $generatedPath -Raw
    $hasContactEmail = $raw -match '"contactEmail"\s*:\s*"[^"]+@[^"]+\.[^"]+"'
    $hasPilotContactUrl = $raw -match '"pilotContactUrl"\s*:\s*"https://(calendly\.com|www\.calendly\.com|tally\.so|www\.tally\.so|form\.typeform\.com|forms\.gle|docs\.google\.com)/[^"]+"'
    $hasStripeLink = $raw -match '"pilot"\s*:\s*"https://buy\.stripe\.com/[^"]+"'
    $configured = ($hasContactEmail -or $hasPilotContactUrl -or $hasStripeLink) -and ($checkExit -eq 0)
    $evidence = "generated_from_env=True; strict_cta_exit=$checkExit; contact_email_configured=$hasContactEmail; pilot_contact_url_configured=$hasPilotContactUrl; stripe_payment_link_configured=$hasStripeLink"

    return [pscustomobject]@{
        pass = $configured
        evidence = $evidence
        fix = "Set NODE_PUBLIC_CONTACT_EMAIL, NODE_PUBLIC_PILOT_CONTACT_URL, and/or NODE_PUBLIC_PILOT_PAYMENT_LINK in the hosting env, then run npm run configure:cta and npm run test:cta before claiming paid-pilot readiness."
    }
}

function Test-ProofDemoConfig {
    $generatedPath = $script:generatedPaymentConfigPath
    if (-not (Test-Path -LiteralPath $generatedPath)) {
        return [pscustomobject]@{
            pass = $false
            evidence = "generated CTA config is missing."
            fix = "Run npm run configure:cta in deploy\public-site with NODE_PUBLIC_PROOF_DEMO_URL set after recording the proof."
        }
    }

    $raw = Get-Content -LiteralPath $generatedPath -Raw
    $hasProofDemoUrl = $raw -match '"proofDemoUrl"\s*:\s*"https://(youtube\.com|www\.youtube\.com|youtu\.be|loom\.com|www\.loom\.com|vimeo\.com|www\.vimeo\.com)/[^"]+"'

    return [pscustomobject]@{
        pass = $hasProofDemoUrl
        evidence = "proof_demo_url_configured=$hasProofDemoUrl"
        fix = "Record the five-minute proof and set NODE_PUBLIC_PROOF_DEMO_URL to a reviewed public video URL."
    }
}

function Test-RecordingBrief {
    $recordingPath = Resolve-RepoPath "outputs\pay-ready-proof-recording.md"
    $hasRecordingBrief = (Test-Path -LiteralPath $recordingPath) -and (Test-FileContains "outputs\pay-ready-proof-recording.md" '^Status:\s+PASS\s*$')

    return [pscustomobject]@{
        pass = $hasRecordingBrief
        evidence = "recording_brief_pass=$hasRecordingBrief"
        fix = "Add outputs\pay-ready-proof-recording.md with Status: PASS, recording URL, date, owner, and packet hash after recording."
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

    $passes = Test-FileContains "outputs\pay-ready-proof.md" '^Status:\s+PASS\s*$'
    $auditExists = Test-Path -LiteralPath (Resolve-RepoPath "outputs\pay-ready-audit.json")
    $transcriptExists = Test-Path -LiteralPath (Resolve-RepoPath "outputs\pay-ready-proof-transcript.txt")
    $artifactExists = Test-Path -LiteralPath (Resolve-RepoPath "work\pay-ready-workspace\internal\backend_fix.txt")
    $passes = $passes -and $auditExists -and $transcriptExists -and $artifactExists
    return [pscustomobject]@{
        pass = $passes
        evidence = "status_pass=$passes; audit_exists=$auditExists; transcript_exists=$transcriptExists; approved_artifact_exists=$artifactExists"
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
        $maxParallel = [int]$report.max_parallel
        $terminal = [int]$report.counts.terminal
        $passes = ($requested -ge 4 -and $maxParallel -ge 2 -and $terminal -eq $requested -and $succeeded -eq $requested -and $failed -eq 0)
        return [pscustomobject]@{
            pass = $passes
            evidence = "requested_runs=$requested; max_parallel=$maxParallel; terminal=$terminal; succeeded=$succeeded; failed=$failed"
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

Push-Location (Resolve-RepoPath "deploy\public-site")
try {
    $paymentConfig = Test-PaymentConfig
} finally {
    Pop-Location
}
Add-ReadinessCheck "Public CTA" "Generated env CTA has real pilot path" $paymentConfig.pass $paymentConfig.evidence $paymentConfig.fix

$ctaGuard = (Test-FileContains "deploy\public-site\package.json" '"test:cta"') -and
    (Test-FileContains "deploy\public-site\package.json" '"test:cta:deploy"') -and
    (Test-FileContains "render.yaml" 'runtime:\s+static') -and
    (Test-FileContains "render.yaml" 'staticPublishPath:\s+\./deploy/public-site/dist') -and
    (Test-FileContains "render.yaml" 'configure:cta.*test:cta:deploy.*build')
Add-ReadinessCheck "Public CTA" "Render build uses static deploy-safe CTA check" $ctaGuard "strict and deploy CTA scripts plus Render static dist build present=$ctaGuard" "Keep configure:cta, test:cta:deploy, and build in the Render static-site build command. Use npm run test:cta separately for paid-pilot readiness."

$netlifyParity = (Test-FileContains "netlify.toml" 'publish\s+=\s+"deploy/public-site/dist"') -and
    (Test-FileContains "netlify.toml" 'configure:cta.*test:cta:deploy.*build')
Add-ReadinessCheck "Public CTA" "Netlify config does not bypass static build" $netlifyParity "Netlify dist publish and deploy CTA build chain present=$netlifyParity" "Keep Netlify publishing deploy/public-site/dist through the same deploy-safe build chain."

$vercelParity = (Test-FileContains "vercel.json" '"outputDirectory"\s*:\s*"deploy/public-site/dist"') -and
    (Test-FileContains "vercel.json" 'configure:cta.*test:cta:deploy.*build')
Add-ReadinessCheck "Public CTA" "Vercel config does not bypass static build" $vercelParity "Vercel dist output and deploy CTA build chain present=$vercelParity" "Keep Vercel publishing deploy/public-site/dist through the same deploy-safe build chain."

$macLinuxScripts = (Test-Path -LiteralPath (Resolve-RepoPath "scripts\build.sh")) -and
    (Test-Path -LiteralPath (Resolve-RepoPath "scripts\demo-pay-ready.sh")) -and
    (Test-Path -LiteralPath (Resolve-RepoPath "scripts\security-audit.sh"))
Add-ReadinessCheck "Buyer Demo" "Mac/Linux demo scripts exist" $macLinuxScripts "build.sh, demo-pay-ready.sh, and security-audit.sh present=$macLinuxScripts" "Add or restore Linux/macOS shell wrappers for the buyer demo."

$proofScript = Test-Path -LiteralPath (Resolve-RepoPath "scripts\New-PayReadyProofPacket.ps1")
Add-ReadinessCheck "Buyer Demo" "Proof packet generator exists" $proofScript "scripts\New-PayReadyProofPacket.ps1 present=$proofScript" "Restore scripts\New-PayReadyProofPacket.ps1 and scripts\new-pay-ready-proof-packet.cmd."

$proofPacket = Test-ProofPacket
Add-ReadinessCheck "Buyer Demo" "Fresh pay-ready proof packet passes" $proofPacket.pass $proofPacket.evidence $proofPacket.fix

$proofDemo = Test-ProofDemoConfig
Add-ReadinessCheck "Buyer Demo" "Public proof demo URL is configured" $proofDemo.pass $proofDemo.evidence $proofDemo.fix

$recordingBrief = Test-RecordingBrief
Add-ReadinessCheck "Buyer Demo" "Recorded proof demo brief exists" $recordingBrief.pass $recordingBrief.evidence $recordingBrief.fix

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

if ($requiredBlockers.Count -gt 0 -and -not $AllowBlockers) {
    exit 1
}
