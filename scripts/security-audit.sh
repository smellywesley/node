#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if command -v git >/dev/null 2>&1; then
  mapfile -t TRACKED < <(git ls-files --cached --others --exclude-standard 2>/dev/null || true)
else
  mapfile -t TRACKED < <(find . -type f -print | sed 's#^\./##')
fi

if [[ "${#TRACKED[@]}" -eq 0 ]]; then
  mapfile -t TRACKED < <(find . -type f -print | sed 's#^\./##')
fi

forbidden_paths=(
  '^bin/'
  '^dist/'
  '^work/'
  '^\.git/'
  '^\.gstack/'
  '^\.Codex/([^c]|c[^o]|co[^m]|com[^m]|comm[^a]|comma[^n]|comman[^d]|command[^s]|commands/.+)'
  '^\.codex/'
  '^\.agents/'
  '^data/learning/'
  '(^|/)__pycache__/'
  '\.pyc$'
  '\.db$'
  '\.db-'
  '\.sqlite$'
  '\.sqlite-'
  '(^|/)(token|approver-token)$'
  '\.zip$'
  '\.tar\.gz$'
)

secret_patterns=(
  'OpenAI API key|sk-proj-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9]{20,}'
  'Stripe API key|sk_(test|live)_[A-Za-z0-9_]{24,}'
  'Stripe webhook secret|whsec_[A-Za-z0-9_]{24,}'
  'GitHub token|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9_]{20,}'
  'Slack token|xox[baprs]-[A-Za-z0-9-]{20,}'
  'AWS access key|AKIA[0-9A-Z]{16}'
  'Private key block|-----BEGIN [A-Z ]*PRIVATE KEY-----'
  'AgentOS dashboard credential URL|https?://(127\.0\.0\.1|localhost|\[::1\]):[0-9]+/#([^[:space:]]+&)?token=[0-9a-fA-F]{64}'
  'AgentOS URL token fragment|#([^[:space:]]+&)?token=[0-9a-fA-F]{64}'
)

findings=0

for path in "${TRACKED[@]}"; do
  normalized="${path//\\//}"
  normalized="${normalized#./}"

  for pattern in "${forbidden_paths[@]}"; do
    if [[ "$normalized" =~ $pattern ]]; then
      printf 'forbidden_path\t%s\t%s\n' "$normalized" "$pattern"
      findings=$((findings + 1))
    fi
  done

  [[ -f "$path" ]] || continue
  size=$(wc -c < "$path" 2>/dev/null || echo 0)
  [[ "$size" -le 2097152 ]] || continue

  for rule in "${secret_patterns[@]}"; do
    name="${rule%%|*}"
    pattern="${rule#*|}"
    if grep -nE "$pattern" "$path" >/tmp/agentos-audit-match.$$ 2>/dev/null; then
      while IFS=: read -r line _; do
        printf 'secret_pattern\t%s\t%s at line %s\n' "$normalized" "$name" "$line"
        findings=$((findings + 1))
      done </tmp/agentos-audit-match.$$
    fi
    rm -f /tmp/agentos-audit-match.$$
  done
done

if [[ "$findings" -gt 0 ]]; then
  echo "security audit failed: $findings finding(s)" >&2
  exit 1
fi

echo "security audit passed: no forbidden tracked paths or high-confidence secret patterns found"
