[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        "init", "sync", "status", "evolve", "export", "import",
        "promote", "projects", "prune"
    )]
    [string]$Command = "status",

    [Parameter(Position = 1)]
    [string]$Argument,

    [string]$Root
)

$ErrorActionPreference = "Stop"
if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Split-Path -Parent $PSScriptRoot
}
$Root = [IO.Path]::GetFullPath($Root)
$learningRoot = Join-Path $Root "data\learning"
$sessionsFile = Join-Path $Root "data\logs\sessions.jsonl"
$configFile = Join-Path $learningRoot "config.json"
$ingestedFile = Join-Path $learningRoot "ingested-session-events.json"
$templateConfig = Join-Path $Root "data\templates\continuous-learning-config.json"

function Resolve-Python {
    $candidates = @(
        $env:AGENTOS_PYTHON,
        (Join-Path $env:USERPROFILE ".cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe"),
        (Join-Path $env:USERPROFILE "AppData\Local\Programs\Python\Python312\python.exe")
    ) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return [IO.Path]::GetFullPath($candidate)
        }
    }
    $python = Get-Command python -ErrorAction SilentlyContinue
    if ($python) {
        $version = & $python.Source -c "import sys; print(sys.version_info[:2] >= (3, 11))"
        if ($LASTEXITCODE -eq 0 -and $version.Trim() -eq "True") {
            return $python.Source
        }
    }
    throw "Python 3.11+ was not found. Set AGENTOS_PYTHON to a compatible interpreter."
}

function Resolve-SkillRoot {
    $candidates = @(
        (Join-Path $env:USERPROFILE ".agents\skills\ecc\continuous-learning-v2"),
        "C:\Users\NewName\.agents\skills\ecc\continuous-learning-v2",
        "C:\Users\owowe\.agents\skills\ecc\continuous-learning-v2"
    )
    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath (Join-Path $candidate "scripts\instinct-cli.py")) {
            return [IO.Path]::GetFullPath($candidate)
        }
    }
    throw "The continuous-learning-v2 skill is not installed under .agents\skills\ecc."
}

$pythonExe = Resolve-Python
$skillRoot = Resolve-SkillRoot
$cli = Join-Path $skillRoot "scripts\instinct-cli.py"

function Get-LearningMutexName {
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        $hashBytes = $sha256.ComputeHash([Text.Encoding]::UTF8.GetBytes($Root))
    }
    finally {
        $sha256.Dispose()
    }
    $hashText = -join ($hashBytes | ForEach-Object { $_.ToString("x2") })
    return "Local\AgentOSLearning-" + $hashText.Substring(0, 16)
}

function Initialize-Learning {
    [IO.Directory]::CreateDirectory($learningRoot) | Out-Null
    if (-not (Test-Path -LiteralPath $configFile)) {
        Copy-Item -LiteralPath $templateConfig -Destination $configFile
    }
    if (-not (Test-Path -LiteralPath $ingestedFile)) {
        [IO.File]::WriteAllText(
            $ingestedFile,
            "[]`n",
            [Text.UTF8Encoding]::new($false)
        )
    }
}

function Invoke-InstinctCli {
    param([string[]]$CliArguments)

    $previousHome = $env:CLV2_HOMUNCULUS_DIR
    $previousProject = $env:CLAUDE_PROJECT_DIR
    $previousConfig = $env:CLV2_CONFIG
    $mutex = [Threading.Mutex]::new($false, (Get-LearningMutexName))
    try {
        if (-not $mutex.WaitOne([TimeSpan]::FromSeconds(30))) {
            throw "Timed out waiting for the Continuous Learning CLI lock."
        }
        $env:CLV2_HOMUNCULUS_DIR = $learningRoot
        $env:CLAUDE_PROJECT_DIR = $Root
        $env:CLV2_CONFIG = $configFile
        & $pythonExe $cli @CliArguments
        if ($LASTEXITCODE -ne 0) {
            throw "continuous-learning-v2 command failed: $($CliArguments -join ' ')"
        }
    }
    finally {
        $env:CLV2_HOMUNCULUS_DIR = $previousHome
        $env:CLAUDE_PROJECT_DIR = $previousProject
        $env:CLV2_CONFIG = $previousConfig
        try { $mutex.ReleaseMutex() } catch {}
        $mutex.Dispose()
    }
}

function Get-ProjectMetadata {
    Invoke-InstinctCli @("status") *> $null
    $registryPath = Join-Path $learningRoot "projects.json"
    $registry = Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json
    $normalizedRoot = $Root.TrimEnd("\")
    foreach ($property in $registry.PSObject.Properties) {
        if ([string]$property.Value.root -eq $normalizedRoot) {
            return [pscustomobject]@{
                Id = $property.Name
                Name = [string]$property.Value.name
                Directory = Join-Path $learningRoot "projects\$($property.Name)"
            }
        }
    }
    throw "Continuous Learning did not register the current project root."
}

function Write-AtomicJson {
    param([string]$Path, [object]$Value)
    $temporary = "$Path.tmp.$PID"
    $json = $Value | ConvertTo-Json -Depth 8
    [IO.File]::WriteAllText(
        $temporary,
        $json + "`n",
        [Text.UTF8Encoding]::new($false)
    )
    Move-Item -LiteralPath $temporary -Destination $Path -Force
}

function Add-SessionObservations {
    param([object]$Project)

    if (-not (Test-Path -LiteralPath $sessionsFile)) {
        return 0
    }
    $observationsFile = Join-Path $Project.Directory "observations.jsonl"
    $mutex = [Threading.Mutex]::new($false, (Get-LearningMutexName))
    try {
        if (-not $mutex.WaitOne([TimeSpan]::FromSeconds(15))) {
            throw "Timed out waiting for the learning observation lock."
        }

        $known = @{}
        $decodedIds = Get-Content -LiteralPath $ingestedFile -Raw | ConvertFrom-Json
        foreach ($id in $decodedIds) {
            $text = [string]$id
            if (-not [string]::IsNullOrWhiteSpace($text) -and -not $text.Contains(" ")) {
                $known[$text] = $true
            }
        }

        $storedLines = [Collections.Generic.List[string]]::new()
        $storedIds = @{}
        if (Test-Path -LiteralPath $observationsFile) {
            foreach ($line in Get-Content -LiteralPath $observationsFile) {
                if ([string]::IsNullOrWhiteSpace($line)) {
                    continue
                }
                $observation = $line | ConvertFrom-Json
                $sourceId = [string]$observation.source_event_id
                if (-not [string]::IsNullOrWhiteSpace($sourceId)) {
                    if ($storedIds.ContainsKey($sourceId)) {
                        continue
                    }
                    $storedIds[$sourceId] = $true
                    $known[$sourceId] = $true
                }
                $storedLines.Add(($observation | ConvertTo-Json -Compress))
            }
        }

        $newCount = 0
        foreach ($line in Get-Content -LiteralPath $sessionsFile) {
            if ([string]::IsNullOrWhiteSpace($line)) {
                continue
            }
            $event = $line | ConvertFrom-Json
            $eventId = [string]$event.event_id
            if ([string]::IsNullOrWhiteSpace($eventId) -or $known.ContainsKey($eventId)) {
                continue
            }

            $summary = [ordered]@{
                outcome = [string]$event.outcome
                specialist_count = @($event.specialists).Count
                changed_path_count = @($event.changed_paths).Count
                verification_count = @($event.verification).Count
                blocker_count = @($event.blockers).Count
            }
            $observation = [ordered]@{
                timestamp = [string]$event.timestamp
                event = "session_complete"
                tool = "AgentOSSession"
                session = [string]$event.session_id
                project_id = $Project.Id
                project_name = $Project.Name
                output = ($summary | ConvertTo-Json -Compress)
                source_event_id = $eventId
            }
            $storedLines.Add(($observation | ConvertTo-Json -Compress))
            $known[$eventId] = $true
            $newCount++
        }

        $temporaryObservations = "$observationsFile.tmp.$PID"
        $observationContent = if ($storedLines.Count -gt 0) {
            ($storedLines -join "`n") + "`n"
        } else {
            ""
        }
        [IO.File]::WriteAllText(
            $temporaryObservations,
            $observationContent,
            [Text.UTF8Encoding]::new($false)
        )
        Move-Item -LiteralPath $temporaryObservations -Destination $observationsFile -Force
        Write-AtomicJson -Path $ingestedFile -Value @($known.Keys | Sort-Object)
        return $newCount
    }
    finally {
        try { $mutex.ReleaseMutex() } catch {}
        $mutex.Dispose()
    }
}

function Add-ProjectInstincts {
    param([object]$Project)

    $directory = Join-Path $Project.Directory "instincts\personal"
    [IO.Directory]::CreateDirectory($directory) | Out-Null
    $definitions = @(
        @{
            Id = "verify-agent-recovery-with-hard-restart"
            Trigger = "when changing agent process lifecycle or recovery"
            Domain = "testing"
            Action = "Verify a hard daemon restart from a durable checkpoint, exactly one recovery transition, and no repeated committed tool action."
            Evidence = "Accepted recovery decision, focused recovery tests, and the v1 hard-kill acceptance run."
        },
        @{
            Id = "broker-consequential-agent-effects"
            Trigger = "when adding an agent action that changes host or external state"
            Domain = "security"
            Action = "Route the effect through the daemon broker with declared capability, policy evaluation, stable idempotency key, approval when required, and append-only audit events."
            Evidence = "SDK tool-boundary decision, approval-principal decision, and brokered fs.write acceptance coverage."
        },
        @{
            Id = "redact-durable-agent-records"
            Trigger = "when persisting agent logs, audits, session records, or costs"
            Domain = "security"
            Action = "Persist structural metadata while redacting secrets, prompts, source contents, mount sources, environment values, and consequential payloads."
            Evidence = "Memory contract, redacted audit acceptance, and cost-ledger privacy invariant."
        }
    )

    $created = 0
    foreach ($definition in $definitions) {
        $path = Join-Path $directory "$($definition.Id).yaml"
        if (Test-Path -LiteralPath $path) {
            continue
        }
        $content = @"
---
id: $($definition.Id)
trigger: "$($definition.Trigger)"
confidence: 0.7
domain: $($definition.Domain)
source: project-memory
scope: project
project_id: $($Project.Id)
project_name: $($Project.Name)
---

# $($definition.Id)

## Action
$($definition.Action)

## Evidence
- $($definition.Evidence)
"@
        [IO.File]::WriteAllText(
            $path,
            $content.TrimStart() + "`n",
            [Text.UTF8Encoding]::new($false)
        )
        $created++
    }
    return $created
}

Initialize-Learning

switch ($Command) {
    "init" {
        $project = Get-ProjectMetadata
        Write-Output "Continuous Learning initialized for $($project.Name) [$($project.Id)]"
        Write-Output "Storage: $($project.Directory)"
    }
    "sync" {
        $project = Get-ProjectMetadata
        $observations = Add-SessionObservations -Project $project
        $instincts = Add-ProjectInstincts -Project $project
        Write-Output "Learning sync complete for $($project.Name) [$($project.Id)]"
        Write-Output "New observations: $observations"
        Write-Output "New instincts: $instincts"
    }
    "status" { Invoke-InstinctCli @("status") }
    "evolve" { Invoke-InstinctCli @("evolve") }
    "projects" { Invoke-InstinctCli @("projects") }
    "prune" { Invoke-InstinctCli @("prune") }
    "promote" {
        if ($Argument) { Invoke-InstinctCli @("promote", $Argument) }
        else { Invoke-InstinctCli @("promote", "--dry-run") }
    }
    "export" {
        $output = if ($Argument) {
            [IO.Path]::GetFullPath((Join-Path $Root $Argument))
        } else {
            Join-Path $Root "outputs\agentos-instincts.md"
        }
        Invoke-InstinctCli @("export", "--scope", "project", "--output", $output)
    }
    "import" {
        if (-not $Argument) {
            throw "Usage: scripts\learning.cmd import <instinct-file>"
        }
        $input = [IO.Path]::GetFullPath((Join-Path $Root $Argument))
        Invoke-InstinctCli @("import", $input, "--scope", "project")
    }
}
