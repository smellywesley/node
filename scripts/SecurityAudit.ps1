param(
    [switch]$Json
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$tracked = @()
try {
    $tracked = @(git ls-files --cached --others --exclude-standard 2>$null)
} catch {
    $tracked = @()
}
if ($tracked.Count -eq 0) {
    $tracked = @(Get-ChildItem -Recurse -File | ForEach-Object { Resolve-Path -Relative $_.FullName })
}

$forbiddenPaths = @(
    '^bin/',
    '^dist/',
    '^deploy/public-site/work/',
    '^deploy/public-site-v1/',
    '^deploy/public-site-v2/',
    '^work/',
    '^\.git/',
    '^\.gstack/',
    '^\.Codex/(?!commands/)',
    '^\.codex/',
    '^\.agents/',
    '^data/learning/',
    '(^|/)__pycache__/',
    '\.pyc$',
    '\.db$',
    '\.db-',
    '\.sqlite$',
    '\.sqlite-',
    '(^|/)(token|approver-token)$',
    '\.zip$',
    '\.tar\.gz$'
)

$secretPatterns = @(
    @{ Name = 'OpenAI API key'; Pattern = 'sk-proj-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9]{20,}' },
    @{ Name = 'Stripe API key'; Pattern = 'sk_(test|live)_[A-Za-z0-9_]{24,}' },
    @{ Name = 'Stripe webhook secret'; Pattern = 'whsec_[A-Za-z0-9_]{24,}' },
    @{ Name = 'GitHub token'; Pattern = 'github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9_]{20,}' },
    @{ Name = 'Slack token'; Pattern = 'xox[baprs]-[A-Za-z0-9-]{20,}' },
    @{ Name = 'AWS access key'; Pattern = 'AKIA[0-9A-Z]{16}' },
    @{ Name = 'Private key block'; Pattern = '-----BEGIN [A-Z ]*PRIVATE KEY-----' },
    @{ Name = 'AgentOS dashboard credential URL'; Pattern = 'https?://(127\.0\.0\.1|localhost|\[::1\]):[0-9]+/#([^\s]+&)?token=[0-9a-fA-F]{64}' },
    @{ Name = 'AgentOS URL token fragment'; Pattern = '#([^\s]+&)?token=[0-9a-fA-F]{64}' }
)

$findings = @()
foreach ($path in $tracked) {
    $normalized = ($path -replace '\\', '/')
    if ($normalized.StartsWith('./')) {
        $normalized = $normalized.Substring(2)
    }
    foreach ($pattern in $forbiddenPaths) {
        if ($normalized -cmatch $pattern) {
            $findings += [pscustomobject]@{ Type = 'forbidden_path'; Path = $normalized; Detail = $pattern }
        }
    }
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        continue
    }
    $info = Get-Item -LiteralPath $path
    if ($info.Length -gt 2MB) {
        continue
    }
    try {
        $content = Get-Content -LiteralPath $path -Raw -ErrorAction Stop
    } catch {
        continue
    }
    if ($null -eq $content) {
        $content = ''
    }
    foreach ($rule in $secretPatterns) {
        $matches = [regex]::Matches($content, $rule.Pattern)
        foreach ($match in $matches) {
            $line = ($content.Substring(0, $match.Index) -split "`n").Count
            $findings += [pscustomobject]@{ Type = 'secret_pattern'; Path = $normalized; Detail = "$($rule.Name) at line $line" }
        }
    }
}

if ($Json) {
    [pscustomobject]@{ findings = $findings; count = $findings.Count } | ConvertTo-Json -Depth 4
} elseif ($findings.Count -eq 0) {
    Write-Output 'security audit passed: no forbidden tracked paths or high-confidence secret patterns found'
} else {
    Write-Output 'security audit failed:'
    $findings | Format-Table -AutoSize
}

if ($findings.Count -gt 0) {
    exit 1
}
