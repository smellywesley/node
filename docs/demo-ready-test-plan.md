# AgentOS Demo-Ready Test Plan

Updated: 2026-06-23

## Purpose

Prove that AgentOS is safe enough to demo and publish as a local-first developer preview. The test plan validates the unique claim: AgentOS manages durable, permissioned, replayable agent processes rather than acting as a thin LLM wrapper.

## Required Gates

| Gate | Command or Evidence | Pass Criteria |
|---|---|---|
| Go unit tests | `scripts\test.cmd` or `go test ./...` | All Go packages pass. |
| Python adapter tests | `python -m unittest discover -s adapters/agents-sdk/tests` | Offline SDK adapter tests pass. |
| Static vet | `go vet ./cmd/... ./internal/...` | No vet findings. |
| Manifest validation | `bin\agentos.exe validate examples\smoke\agent-process.yaml` | Manifest accepted. |
| Security audit | `scripts\security-audit.cmd` | Zero forbidden tracked paths and zero high-confidence secret findings. |
| Package audit | `scripts\package.cmd` plus package inspection | Release archive excludes work, state, tokens, DBs, git metadata, caches, and private local files. |
| Localhost demo | `scripts\start-localhost.cmd` | Dashboard starts on loopback with credentials only in browser session storage. |
| Smoke process | `agentos run examples\smoke\agent-process.yaml` | Process reaches `succeeded`, logs show lifecycle, replay returns same terminal state. |
| Token rotation | `agentos rotate-token` while daemon is stopped | Token file changes and command does not print the new token. |
| Benchmark | `outputs\BENCHMARK_REPORT.md` | Dashboard/static assets and operator workflows remain inside demo budgets. |

## Cybersecurity Checks

- Do not commit operator tokens, approver tokens, dashboard URLs with `#token=`, SQLite state, Docker image archives, release zips, or local runtime folders.
- Treat `agentos dashboard --print-url` as secret output.
- Rotate the operator token after any accidental exposure.
- Keep worker containers off the host Docker socket and off undeclared networks.
- Confirm every secret, filesystem path, network destination, and tool access is manifest-declared before execution.

## Demo Script

1. Build: `scripts\build.cmd`.
2. Audit: `scripts\security-audit.cmd`.
3. Start: `scripts\start-localhost.cmd -Address 127.0.0.1:7479`.
4. Open dashboard and verify daemon health.
5. Build smoke worker image if needed.
6. Run `examples\smoke\agent-process.yaml`.
7. Inspect process lifecycle, logs, replay, and audit bundle.
8. Stop daemon, rotate token, restart daemon, and confirm old dashboard sessions no longer authorize API calls.

## Exit Criteria

The demo is ready only when all gates pass on the current working tree, the exposed operator token has been rotated, and no credential-bearing material appears in tracked files or generated public reports.