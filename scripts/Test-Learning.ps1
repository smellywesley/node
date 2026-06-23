$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$testRoot = Join-Path $root "work\learning-bridge-test"
$resolvedWork = [IO.Path]::GetFullPath((Join-Path $root "work"))
$resolvedTest = [IO.Path]::GetFullPath($testRoot)
if (-not $resolvedTest.StartsWith(
    $resolvedWork + "\",
    [StringComparison]::OrdinalIgnoreCase
)) {
    throw "Learning test root escaped the workspace work directory."
}

if (Test-Path -LiteralPath $resolvedTest) {
    Remove-Item -LiteralPath $resolvedTest -Recurse -Force
}

try {
    New-Item -ItemType Directory -Force (
        Join-Path $resolvedTest "data\logs"
    ) | Out-Null
    New-Item -ItemType Directory -Force (
        Join-Path $resolvedTest "data\templates"
    ) | Out-Null
    New-Item -ItemType Directory -Force (
        Join-Path $resolvedTest "outputs"
    ) | Out-Null

    Copy-Item -LiteralPath (
        Join-Path $root "data\templates\continuous-learning-config.json"
    ) -Destination (
        Join-Path $resolvedTest "data\templates\continuous-learning-config.json"
    )

    $session = @{
        schema_version = 1
        event_id = "evt_learning_bridge_test"
        timestamp = "2026-06-12T00:00:00Z"
        event = "session_closed"
        session_id = "ses_learning_bridge_test"
        specialists = @("qa")
        objective = "must-not-persist secret-value-123456"
        outcome = "complete"
        changed_paths = @("secret-source.go")
        verification = @("test with password=must-not-persist")
        blockers = @()
    } | ConvertTo-Json -Compress
    [IO.File]::WriteAllText(
        (Join-Path $resolvedTest "data\logs\sessions.jsonl"),
        $session + "`n",
        [Text.UTF8Encoding]::new($false)
    )

    & (Join-Path $root "scripts\Learning.ps1") sync -Root $resolvedTest | Out-Null
    & (Join-Path $root "scripts\Learning.ps1") sync -Root $resolvedTest | Out-Null
    & (Join-Path $root "scripts\Learning.ps1") export `
        "outputs\instincts.md" -Root $resolvedTest | Out-Null

    $registry = Get-Content (
        Join-Path $resolvedTest "data\learning\projects.json"
    ) -Raw | ConvertFrom-Json
    $projectId = ($registry.PSObject.Properties | Select-Object -First 1).Name
    $projectDir = Join-Path $resolvedTest "data\learning\projects\$projectId"
    $observations = @(
        Get-Content (Join-Path $projectDir "observations.jsonl")
    )
    $decodedCursor = Get-Content (
        Join-Path $resolvedTest "data\learning\ingested-session-events.json"
    ) -Raw | ConvertFrom-Json
    $cursor = @($decodedCursor | ForEach-Object { $_ })
    $instincts = @(
        Get-ChildItem (Join-Path $projectDir "instincts\personal") -File
    )
    $export = Get-Content (
        Join-Path $resolvedTest "outputs\instincts.md"
    ) -Raw

    if ($observations.Count -ne 1 -or $cursor.Count -ne 1) {
        throw "Learning ingestion was not idempotent: observations=$($observations.Count), cursor=$($cursor.Count)."
    }
    if ($instincts.Count -ne 3) {
        throw "Expected three seeded project instincts."
    }
    $observationText = $observations -join "`n"
    if ($observationText -match "must-not-persist|secret-source|password") {
        throw "Sensitive session fields leaked into observations."
    }
    if ($export -match "source_event_id|must-not-persist") {
        throw "Raw observation data leaked into the instinct export."
    }
    if (([regex]::Matches($export, "(?m)^id: ")).Count -ne 3) {
        throw "Instinct export did not contain three entries."
    }

    Write-Output "Continuous Learning bridge tests passed."
}
finally {
    if (Test-Path -LiteralPath $resolvedTest) {
        Remove-Item -LiteralPath $resolvedTest -Recurse -Force
    }
}
