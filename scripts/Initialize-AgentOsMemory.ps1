[CmdletBinding()]
param(
    [string]$Root
)

$ErrorActionPreference = "Stop"
if ([string]::IsNullOrWhiteSpace($Root)) {
    $scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
    $Root = Split-Path -Parent $scriptDirectory
}

$resolvedRoot = [System.IO.Path]::GetFullPath($Root)
$directories = @(
    "data\projects",
    "data\decisions",
    "data\daily-logs",
    "data\inbox",
    "data\logs"
)

foreach ($relativePath in $directories) {
    $path = Join-Path $resolvedRoot $relativePath
    [System.IO.Directory]::CreateDirectory($path) | Out-Null
}

$files = @(
    "data\logs\sessions.jsonl",
    "data\logs\costs.jsonl"
)

foreach ($relativePath in $files) {
    $path = Join-Path $resolvedRoot $relativePath
    if (-not [System.IO.File]::Exists($path)) {
        [System.IO.File]::WriteAllText(
            $path,
            "",
            [System.Text.UTF8Encoding]::new($false)
        )
    }
}

$today = [DateTime]::UtcNow.ToString("yyyy-MM-dd")
$dailyLog = Join-Path $resolvedRoot "data\daily-logs\$today.md"
if (-not [System.IO.File]::Exists($dailyLog)) {
    $template = Join-Path $resolvedRoot "data\templates\daily-log.md"
    if ([System.IO.File]::Exists($template)) {
        $content = [System.IO.File]::ReadAllText($template).Replace(
            "YYYY-MM-DD",
            $today
        )
        [System.IO.File]::WriteAllText(
            $dailyLog,
            $content,
            [System.Text.UTF8Encoding]::new($false)
        )
    }
}

Write-Output "Agent Process OS memory initialized at $resolvedRoot"
