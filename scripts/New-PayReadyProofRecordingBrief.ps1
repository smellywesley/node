param(
    [Parameter(Mandatory = $true)]
    [string]$RecordingUrl,
    [string]$Owner = "",
    [string]$Output = "outputs\pay-ready-proof-recording.md"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$proofPath = Join-Path $root "outputs\pay-ready-proof.md"
$auditPath = Join-Path $root "outputs\pay-ready-audit.json"
$outputPath = if ([System.IO.Path]::IsPathRooted($Output)) { $Output } else { Join-Path $root $Output }
$reviewedHosts = @(
    "youtube.com",
    "www.youtube.com",
    "youtu.be",
    "loom.com",
    "www.loom.com",
    "vimeo.com",
    "www.vimeo.com"
)

function Assert-ReviewedRecordingUrl([string]$Value) {
    try {
        $uri = [System.Uri]$Value
    } catch {
        throw "RecordingUrl must be a valid https URL on YouTube, Loom, or Vimeo."
    }

    if ($uri.Scheme -ne "https") {
        throw "RecordingUrl must use https."
    }
    if ($reviewedHosts -notcontains $uri.Host) {
        throw "RecordingUrl host '$($uri.Host)' is not reviewed. Use YouTube, Loom, or Vimeo."
    }
    return $uri.AbsoluteUri
}

if (-not (Test-Path -LiteralPath $proofPath)) {
    throw "Missing proof packet at $proofPath. Run .\scripts\new-pay-ready-proof-packet.cmd first."
}
if (-not (Select-String -LiteralPath $proofPath -Pattern '^Status:\s+PASS\s*$' -Quiet)) {
    throw "Proof packet exists but does not report Status: PASS."
}

$safeUrl = Assert-ReviewedRecordingUrl $RecordingUrl
$proofHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $proofPath).Hash
$auditHash = if (Test-Path -LiteralPath $auditPath) { (Get-FileHash -Algorithm SHA256 -LiteralPath $auditPath).Hash } else { "" }
$generatedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$lines = @()
$lines += "# NODE Pay-Ready Proof Recording"
$lines += ""
$lines += "Generated: $generatedAt"
$lines += "Status: PASS"
$lines += "Recording URL: $safeUrl"
$lines += "Owner: $Owner"
$lines += "Proof packet: outputs/pay-ready-proof.md"
$lines += "Proof packet SHA-256: $proofHash"
$lines += "Audit bundle: outputs/pay-ready-audit.json"
$lines += "Audit bundle SHA-256: $auditHash"
$lines += ""
$lines += "## Recording Must Show"
$lines += ""
$lines += "- Local/private AgentOS proof, not hosted SaaS."
$lines += "- Allowed backend write approved by a human."
$lines += "- Forbidden frontend write denied."
$lines += "- Nonzero token/cost accounting."
$lines += "- Replay and redacted audit bundle export."
$lines += "- Hosted backend remains gated on tenant isolation, roles, billing ledger, load evidence, and external security review."

$outputDir = Split-Path -Parent $outputPath
if ($outputDir) {
    New-Item -ItemType Directory -Force $outputDir | Out-Null
}
$lines | Set-Content -LiteralPath $outputPath -Encoding UTF8

Write-Host "Pay-ready proof recording brief written: $outputPath"
Write-Host "Status: PASS"
