# AgentOS Release Readiness

Updated: 2026-07-01

## Current Alpha Gate

AgentOS can be shared as a local-first Windows developer preview when the release zip, checksum, docs, and smoke evidence all come from the same corrected source tree.

## Required Before Public Alpha

- `go test ./cmd/... ./internal/...` passes.
- `go vet ./cmd/... ./internal/...` passes.
- `scripts\security-audit.cmd` passes with zero forbidden tracked paths and zero high-confidence secret findings.
- `agentos doctor` reports no FAIL items on a clean machine.
- `agentos validate examples\agents-sdk-coding\agent-process.yaml` passes static manifest validation.
- Extracted package starts the localhost dashboard without Go installed.
- Windows state directory, token, and SQLite files have current-user-only ACLs.
- Archive contains no runtime state, caches, credentials, `.git`, `.gstack`, `outputs`, or nested `dist` content.
- Release workflow publishes the packaged zip plus checksum, not only a standalone binary.


## Next Public/Demo Push Checklist

Run these before the next public push or Render deploy:

```powershell
git diff --check
.\scripts\security-audit.cmd
.\scripts\test.cmd
.\scripts\build.cmd
.\bin\agentos.exe doctor --support
.\bin\agentos.exe validate .\examples\pay-ready\agent-process.yaml
```

Docker-off path: `doctor --support` and `demo-pay-ready.cmd` must explain the problem, cause, and fix instead of failing silently.

Docker-on path: `scripts\demo-pay-ready.cmd` must complete the allowed write, denied forbidden write, approval, usage/cost, replay, and redacted audit export checks.

Support path: after a real run, `agentos support-bundle <process-id> <output.json>` must export daemon health plus the redacted audit bundle only. It must not include raw event, replay, token, SQLite, or runtime-state payloads.
## Current Evidence

- Package path: `dist\agentos-v1-windows-amd64.zip`
- SHA-256: see `dist\agentos-v1-windows-amd64.zip.sha256` generated beside the archive
- Last verified archive entry count: 115
- Last verified forbidden archive entries: 0
- Last extracted smoke: `version`, `doctor`, `validate`, localhost `start/stop`, and Windows state ACL inspection passed
- Docker was not running during the last smoke, so the containerized example was not executed in that pass.

## Remaining Work

- Run the offline coding-agent container example with Docker running.
- Keep the MIT public posture intentional before external publication.
- Add signed release artifacts if distributing beyond local testing.
- Add a `version` smoke check to CI/release.



## Current Demo Evidence

Verified on 2026-06-23:

- `scripts\test.cmd` passed.
- `go vet ./cmd/... ./internal/...` passed.
- `scripts\security-audit.cmd` passed.
- Localhost daemon is healthy on `127.0.0.1:7479`.
- Dashboard serves the guided demo path `See the agent process OS in one run`.
- Smoke process `01fb054b-02c9-4505-90bd-a817c9804b43` reached `succeeded` and replayed to `succeeded`.
- Release archive `dist\agentos-v1-windows-amd64.zip` has 119 entries and zero forbidden entries.
- Release archive SHA-256: `C3141AB56F5BB19F242853195EBA8380F078F21B43A4AD4AEE8FE7EC2ECAC230`.