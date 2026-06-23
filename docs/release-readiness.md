# AgentOS Release Readiness

Updated: 2026-06-23

## Current Alpha Gate

AgentOS can be shared as a local-first Windows developer preview when the release zip, checksum, docs, and smoke evidence all come from the same corrected source tree.

## Required Before Public Alpha

- `go test ./cmd/... ./internal/...` passes.
- `go vet ./cmd/... ./internal/...` passes.
- `agentos doctor` reports no FAIL items on a clean machine.
- `agentos validate examples\agents-sdk-coding\agent-process.yaml` passes static manifest validation.
- Extracted package starts the localhost dashboard without Go installed.
- Windows state directory, token, and SQLite files have current-user-only ACLs.
- Archive contains no runtime state, caches, credentials, `.git`, `.gstack`, `outputs`, or nested `dist` content.
- Release workflow publishes the packaged zip plus checksum, not only a standalone binary.

## Current Evidence

- Package path: `dist\agentos-v1-windows-amd64.zip`
- SHA-256: see `dist\agentos-v1-windows-amd64.zip.sha256` generated beside the archive
- Last verified archive entry count: 115
- Last verified forbidden archive entries: 0
- Last extracted smoke: `version`, `doctor`, `validate`, localhost `start/stop`, and Windows state ACL inspection passed
- Docker was not running during the last smoke, so the containerized example was not executed in that pass.

## Remaining Work

- Run the offline coding-agent container example with Docker running.
- Decide the real public license before external publication.
- Add signed release artifacts if distributing beyond local testing.
- Add a `version` smoke check to CI/release.


