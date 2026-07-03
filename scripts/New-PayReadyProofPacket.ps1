param(
    [string]$Address = "127.0.0.1:17482",
    [switch]$SkipDemo
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$outputs = Join-Path $root "outputs"
$transcriptPath = Join-Path $outputs "pay-ready-proof-transcript.txt"
$packetPath = Join-Path $outputs "pay-ready-proof.md"
$auditPath = Join-Path $outputs "pay-ready-audit.json"
$artifactPath = Join-Path $root "work\pay-ready-workspace\internal\backend_fix.txt"
$forbiddenPath = Join-Path $root "work\pay-ready-workspace\web\app.js"

function Escape-Markdown([string]$value) {
    if ([string]::IsNullOrWhiteSpace($value)) { return "" }
    return $value.Replace("|", "\|")
}

New-Item -ItemType Directory -Force $outputs | Out-Null

if (-not $SkipDemo) {
    $demoScript = Join-Path $PSScriptRoot "Demo-PayReady.ps1"
    $demoOutput = & $demoScript -Address $Address 2>&1
    $demoExit = $LASTEXITCODE
    $demoOutput | Set-Content -LiteralPath $transcriptPath -Encoding UTF8
    if ($demoExit -ne 0) {
        throw "Pay-ready demo failed. Transcript written to $transcriptPath"
    }
}

if (-not (Test-Path -LiteralPath $auditPath)) {
    throw "Missing audit bundle at $auditPath. Run this script without -SkipDemo after starting Docker."
}

$rawAudit = Get-Content -LiteralPath $auditPath -Raw
$audit = $rawAudit | ConvertFrom-Json
$events = @($audit.events)
if ($events.Count -eq 0 -and $audit.process.events) {
    $events = @($audit.process.events)
}

$denials = @($events | Where-Object { $_.type -eq "tool.denied" })
$approvals = @($events | Where-Object { $_.type -like "approval.*" })
$usageEvents = @($events | Where-Object { $_.type -eq "budget.usage_updated" })
$toolCalls = @($events | Where-Object { $_.type -like "tool.*" })

$auditHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $auditPath).Hash
$artifactHash = if (Test-Path -LiteralPath $artifactPath) { (Get-FileHash -Algorithm SHA256 -LiteralPath $artifactPath).Hash } else { "" }
$forbiddenExists = Test-Path -LiteralPath $forbiddenPath
$transcriptHash = if (Test-Path -LiteralPath $transcriptPath) { (Get-FileHash -Algorithm SHA256 -LiteralPath $transcriptPath).Hash } else { "" }

$state = if ($audit.process.state) { $audit.process.state } elseif ($audit.state) { $audit.state } else { "unknown" }
$processID = if ($audit.process.id) { $audit.process.id } elseif ($audit.id) { $audit.id } else { "unknown" }
$tokens = if ($audit.process.usage.tokens) { $audit.process.usage.tokens } elseif ($audit.usage.tokens) { $audit.usage.tokens } else { 0 }
$cost = if ($audit.process.usage.cost_usd) { $audit.process.usage.cost_usd } elseif ($audit.usage.cost_usd) { $audit.usage.cost_usd } else { 0 }

$checks = @(
    [pscustomobject]@{ Check = "Real managed process completed"; Evidence = "process_id=$processID state=$state"; Pass = ($state -eq "succeeded") },
    [pscustomobject]@{ Check = "Forbidden frontend write denied"; Evidence = "$($denials.Count) tool.denied event(s); forbidden file exists=$forbiddenExists"; Pass = ($denials.Count -gt 0 -and -not $forbiddenExists) },
    [pscustomobject]@{ Check = "Approval gate exercised"; Evidence = "$($approvals.Count) approval event(s)"; Pass = ($approvals.Count -gt 0) },
    [pscustomobject]@{ Check = "Nonzero usage/cost"; Evidence = "$tokens tokens; cost_usd=$cost"; Pass = ([double]$tokens -gt 0 -and [double]$cost -gt 0) },
    [pscustomobject]@{ Check = "Audit bundle exists"; Evidence = "sha256=$auditHash"; Pass = (Test-Path -LiteralPath $auditPath) },
    [pscustomobject]@{ Check = "Approved backend artifact exists"; Evidence = if ($artifactHash) { "sha256=$artifactHash" } else { "missing" }; Pass = [bool]$artifactHash }
)

$status = if (@($checks | Where-Object { -not $_.Pass }).Count -eq 0) { "PASS" } else { "FAIL" }
$generatedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$lines = @()
$lines += "# NODE Pay-Ready Proof Packet"
$lines += ""
$lines += "Generated: $generatedAt"
$lines += "Status: $status"
$lines += "Scenario: five-minute local proof for safe execution and audit trails for AI coding agents."
$lines += ""
$lines += "## Buyer Story"
$lines += ""
$lines += "A coding agent is asked to fix backend code and not touch the frontend. AgentOS runs it as a managed process, pauses an allowed backend write for approval, blocks a forbidden frontend write, records nonzero usage/cost, and exports a redacted audit bundle."
$lines += ""
$lines += "## Evidence Files"
$lines += ""
$lines += "| Artifact | Path | SHA-256 |"
$lines += "|---|---|---|"
$lines += "| Audit bundle | outputs/pay-ready-audit.json | $auditHash |"
$lines += "| Demo transcript | outputs/pay-ready-proof-transcript.txt | $transcriptHash |"
$lines += "| Approved backend artifact | work/pay-ready-workspace/internal/backend_fix.txt | $artifactHash |"
$lines += ""
$lines += "## Proof Checks"
$lines += ""
$lines += "| Check | Result | Evidence |"
$lines += "|---|---|---|"
foreach ($check in $checks) {
    $result = if ($check.Pass) { "PASS" } else { "FAIL" }
    $lines += "| $(Escape-Markdown $check.Check) | $result | $(Escape-Markdown $check.Evidence) |"
}
$lines += ""
$lines += "## Event Counts"
$lines += ""
$lines += "- Tool events: $($toolCalls.Count)"
$lines += "- Denials: $($denials.Count)"
$lines += "- Approval events: $($approvals.Count)"
$lines += "- Usage events: $($usageEvents.Count)"
$lines += ""
$lines += "## Recording Checklist"
$lines += ""
$lines += "1. Show the public site pilot path: safe execution and audit trails for AI coding agents."
$lines += "2. Run `.\scripts\New-PayReadyProofPacket.ps1` with Docker running."
$lines += "3. Narrate the approval gate, denied frontend write, nonzero usage/cost, replay, and audit bundle."
$lines += "4. Show this packet and the audit bundle hash at the end of the recording."
$lines += ""
$lines += "## Non-Claims"
$lines += ""
$lines += "- This proof is local/private. It does not claim hosted multi-tenant readiness."
$lines += "- Managed model usage remains locked until tenant isolation, spend caps, and a billable usage ledger exist."
$lines += "- Enterprise deployment waits for roles, load evidence, and external security review."

$lines | Set-Content -LiteralPath $packetPath -Encoding UTF8

Write-Host "Pay-ready proof packet written: $packetPath"
Write-Host "Status: $status"
if ($status -ne "PASS") {
    exit 1
}
