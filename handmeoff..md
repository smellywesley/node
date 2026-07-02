# NODE Handoff

Updated: 2026-07-01
Project path: `C:\Users\NewName\Documents\Codex\2026-06-10\i-want-to-buildent-an-operating`
Repository: `smellywesley/node`
Current priority: pay-ready local proof, not hosted SaaS.

## Product Direction

NODE is an agentic OS for enterprise teams. The local runtime, AgentOS, manages AI coding agents as durable, permissioned, replayable processes. The commercial wedge is not "another LLM wrapper". It is safe execution, policy enforcement, budgets, approvals, recovery, replay, audit bundles, and evidence for teams running coding agents on repositories.

Do not expose `agentos serve` publicly yet. Render should only host the static public site under `deploy/public-site` until tenant isolation, roles, project IDs, spend caps, usage ledger, metrics, pagination, and support runbooks exist.

## Current Implementation State

Implemented in the latest working tree:

- Public-site narrative/pricing/security improvements under `deploy/public-site`.
- Canonical plan refresh in `plan.md`.
- README Start Here section for public-site deploy vs local runtime demo.
- `agentos doctor --support` with problem/cause/fix guidance.
- `agentos support-bundle <process-id> [output.json]`.
- Support bundles intentionally collect daemon health plus the redacted audit export only.
- Docker readiness rescue path in pay-ready and live Agents SDK demo scripts.
- Local GitHub artifact workflow scaffold:
  - `scripts\demo-github-artifact.cmd`
  - `scripts\Demo-GitHubArtifact.ps1`
- Quieter Agents SDK adapter expected-failure logging unless `AGENTOS_DEBUG_ERRORS=1`.
- Focused CLI tests for redacted support-bundle behavior and Docker rescue guidance.

## Verification Already Run

Passed:

```powershell
git diff --check
cmd /c scripts\security-audit.cmd
cmd /c scripts\test.cmd
cmd /c scripts\build.cmd
.\bin\agentos.exe doctor --support
.\bin\agentos.exe validate .\examples\pay-ready\agent-process.yaml
```

Important caveat: Docker Desktop was not running. `doctor --support` completed with warnings for missing `AGENTOS_APPROVER_TOKEN` and Docker engine unavailable. Manifest validation passed structurally but warned that the Docker image could not be checked because Docker was off.

The Docker-off pay-ready script path now prints useful problem/cause/fix guidance instead of a raw PowerShell/Docker exception.

## Known Gaps

Not proven yet:

- Full Docker-on `scripts\demo-pay-ready.cmd` run.
- Full real OpenAI/Agents SDK run with an actual key.
- Real branch/diff/test/audit artifact from a Docker-on coding run.
- Render redeploy sanity check after pushing latest static site changes.
- Commit and push of the current local changes.

Do not claim pay-ready hosted SaaS. The project is currently a credible local-first developer preview with a stronger pay-ready proof path.

## Next Commands

When Docker Desktop is running:

```powershell
cd /d C:\Users\NewName\Documents\Codex\2026-06-10\i-want-to-buildent-an-operating
.\bin\agentos.exe doctor --support
.\scripts\demo-pay-ready.cmd
```

For GitHub-style local review evidence after the tree is clean/committed:

```powershell
.\scripts\demo-github-artifact.cmd
```

Before committing or pushing:

```powershell
git status -sb
git diff --check
cmd /c scripts\security-audit.cmd
cmd /c scripts\test.cmd
cmd /c scripts\build.cmd
```

Suggested commit message:

```text
feat: improve pay-ready devex proof path
```

Push manually from the user terminal after review:

```powershell
git push origin main
```

Then trigger Render manual deploy for the static public site.

## Safety Notes

- Keep MIT public posture intentional.
- Keep Stripe subscription/BYOK as the first paid model.
- Do not sell managed model usage until billable usage ledger, spend caps, and tenant/project isolation exist.
- Do not store or export raw tokens, Stripe secrets, raw webhook bodies, card numbers, bank details, CVC, raw runtime state, or raw support event payloads.
- Support bundles should remain redacted by default.
- Public site must not include daemon API URLs, operator tokens, audit bundles, SQLite state, or Stripe secret keys.