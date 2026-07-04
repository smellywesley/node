# NODE

NODE is an agentic OS for enterprise teams. Its local runtime, AgentOS, is a
process supervisor for AI coding agents. It does not
replace Linux or Windows. It adds durable lifecycle, policy, approvals, budgets,
container isolation, recovery, replay, and auditability around probabilistic
agent workloads.

```text
Developer -> CLI -> authenticated local API -> daemon -> policy -> container
                              |                    |
                              v                    v
                        SQLite event log       model + tools
```

## What works

- Durable process state and append-only SQLite events
- Restart recovery for interrupted non-terminal processes
- Durable checkpoints passed back to restarted workers
- Container execution with read-only root, dropped capabilities, resource limits,
  immutable image resolution, and no network by default
- Capability checks and digest-bound, single-use approvals
- Daemon-brokered approval-gated filesystem writes
- Idempotent tool-call records with explicit `outcome_unknown`
- Token, cost, duration, concurrency, and child-process limits
- Parent cancellation and child capability/budget narrowing
- Projection replay with no external side effects
- Redacted audit bundle export
- OpenAI Agents SDK JSON-lines adapter with streamed usage accounting
- Persistent Agentic OS kernel, specialists, commands, and file memory

## Start Here

Pick one track:

1. **Public site deploy**: deploy `deploy/public-site` as a static Render site. This publishes the NODE story and paid-pilot request path only; it does not expose the local daemon, tokens, SQLite state, audit bundles, or worker endpoints.
2. **Local runtime demo**: build the CLI, run `doctor --support`, then run the pay-ready proof. Docker Desktop or another Docker-compatible engine must be running for containerized agent runs.

```powershell
.\scripts\build.cmd
.\bin\agentos.exe doctor --support
.\scripts\demo-pay-ready.cmd
```

On Linux or macOS:

```bash
./scripts/build.sh
./bin/agentos doctor --support
./scripts/demo-pay-ready.sh
```

The pay-ready proof is the official local buyer story: allowed backend write, denied forbidden write, approval gate, nonzero usage/cost, replay, and redacted audit export.
For local review evidence after the pay-ready demo, use:

```powershell
.\scripts\demo-github-artifact.cmd
.\scripts\new-pay-ready-proof-packet.cmd
.\scripts\new-pay-ready-proof-recording-brief.cmd -RecordingUrl https://www.loom.com/share/...
.\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2
.\scripts\test-pilot-readiness.cmd
```

The GitHub artifact demo creates a local branch, captures diff/test evidence, and links the run audit bundle without pushing anything to GitHub. The proof packet script captures the pay-ready transcript, audit hash, event counts, and recording checklist in `outputs\pay-ready-proof.md`. After recording, the recording brief script stamps the reviewed video URL and proof packet hash in `outputs\pay-ready-proof-recording.md`. The backend load script runs bounded concurrent smoke processes and writes `outputs\backend-load-report.json`.
The pilot readiness audit summarizes public CTA, proof packet, Docker, load, security, and sales-playbook gates in one local report.
## Quick start

Requirements: Docker Desktop or another Docker-compatible engine for running agent containers. The localhost dashboard itself works without Docker.

From a release zip:

```powershell
Expand-Archive .\agentos-v1-windows-amd64.zip -DestinationPath .
cd .\agentos-v1-windows-amd64
Get-FileHash .\bin\agentos.exe
.\bin\agentos.exe doctor
.\scripts\start-localhost.cmd
```

`agentos doctor` checks loopback binding, state directory safety, approval-token readiness, and Docker availability. Treat `dashboard --print-url` output as a secret because it contains a credential-bearing URL fragment. If an operator token is exposed, stop the daemon and run `agentos rotate-token` before reconnecting dashboards.

For the localhost control plane, build and open the dashboard in one command:

```powershell
.\scripts\start-localhost.cmd
```

This starts an isolated demo daemon under `work\localhost`, opens
`http://127.0.0.1:7467`, and places both credentials only in the launched
browser tab's session storage. The dashboard itself works without Docker;
running an agent process requires a Docker-compatible engine. Stop the demo
daemon with:

```powershell
.\scripts\stop-localhost.cmd
```

## Pay-ready local proof

The fastest way to see why AgentOS matters is the pay-ready demo:

```powershell
.\scripts\demo-pay-ready.cmd
```

```bash
./scripts/demo-pay-ready.sh
```

It builds a local worker, starts an isolated daemon, runs the task "Fix the code on the backend. Do not touch anything else.", pauses for an approval-gated backend write, denies a forbidden frontend write, records nonzero token/cost usage, replays the process state, and exports `outputs\pay-ready-audit.json`.

This is still a local proof. The provider-backed OpenAI/Agents SDK coding run and GitHub PR flow remain the next commercial-readiness gates.

For the provider-backed SDK proof, set an explicit key and run:

```powershell
$env:OPENAI_API_KEY = "sk-..."
.\scripts\demo-live-agents-sdk.cmd
```

That demo spends API credit, so it exits before running if `OPENAI_API_KEY` is absent.

Build from source:

```powershell
.\scripts\build.cmd
```

```bash
./scripts/build.sh
```

Start the daemon:

```powershell
$env:AGENTOS_APPROVER_TOKEN = [guid]::NewGuid().ToString("N") + [guid]::NewGuid().ToString("N")
.\bin\agentos.exe serve
```

In another terminal:

```powershell
$env:AGENTOS_APPROVER_TOKEN = "<the same approval secret>"
.\bin\agentos.exe dashboard
.\bin\agentos.exe ps
docker build -f .\examples\agents-sdk-coding\Dockerfile `
  -t agentos/agents-sdk-coding:local .
New-Item -ItemType Directory -Force .\work\agents-sdk-coding-workspace
.\bin\agentos.exe validate .\examples\agents-sdk-coding\agent-process.yaml
.\bin\agentos.exe run .\examples\agents-sdk-coding\agent-process.yaml
.\bin\agentos.exe approvals
.\bin\agentos.exe approve <approval-id> "reviewed"
```

The approved Go artifact appears under
`work\agents-sdk-coding-workspace\reviewed`. This example uses the installed
OpenAI Agents SDK with an offline deterministic model, so it needs no provider
key.

For an offline first run, load the separately distributed example image bundle
instead of building from PyPI:

```powershell
docker load -i .\agentos-agents-sdk-coding-local.tar.gz
```

Regenerate that bundle from a verified local image with
`.\scripts\package-example-image.cmd`.

The daemon stores SQLite state and a random operator credential in
`%USERPROFILE%\.agentos` by default. Approval decisions require the separate,
non-persisted `AGENTOS_APPROVER_TOKEN`. Override state location with
`AGENTOS_HOME`, but it must point at a dedicated subdirectory, never the user
profile root or a filesystem root.

The dashboard is embedded in the Go binary and has no frontend build
dependency. Static assets are public on loopback, while every stateful API call
still requires the operator credential. Browser API requests must be same-origin
with the loopback daemon. Approval actions continue to require the separate
approver credential. `dashboard --print-url` is intended for automation and
prints a credential-bearing URL; treat its output as a secret.

## CLI

```text
agentos serve
agentos dashboard [--print-url]
agentos rotate-token [--force]
agentos run <manifest.yaml>
agentos ps
agentos inspect <process-id>
agentos suspend|resume|cancel <process-id>
agentos approvals
agentos approve|deny <approval-id> [reason]
agentos logs <process-id>
agentos replay <process-id>
agentos audit <process-id> [output.json]
```

## Security model

Agents are treated as hostile workloads. The daemon:

- never mounts the Docker socket into workers;
- rejects non-loopback API bindings and authenticates every non-health request;
- rejects symlink mount roots and unsafe Windows device/UNC path forms;
- rejects manifests whose direct mounts, egress, or secrets would bypass a
  matching approval rule;
- disables networking unless the manifest declares destinations, then places
  the worker on an isolated internal network with a per-process allowlist proxy;
- injects only explicitly declared secret names and redacts their values from
  captured output;
- resolves image tags to repository digests before launch;
- keeps policy, approvals, lifecycle, and budget authority outside adapters.
- requires a distinct approver credential for approval decisions.

Containers are defense in depth, not a perfect security boundary. On Windows,
Docker Desktop and WSL2 boundaries must be audited separately before using
untrusted agents.

## Replay semantics

`agentos replay` is projection replay: it deterministically rebuilds the process
state from recorded events and performs no model calls, tools, or external side
effects. A live rerun must be created as a new process.

External side effects cannot generally be made exactly-once across a crash.
AgentOS records stable idempotency keys and supports an explicit
`outcome_unknown` result so operators reconcile ambiguous operations instead of
blindly retrying them.

## OpenAI Agents SDK adapter

The adapter lives in [`adapters/agents-sdk`](adapters/agents-sdk). The example
under [`examples/agents-sdk`](examples/agents-sdk) demonstrates a provider-backed
agent and requires `OPENAI_API_KEY`. The offline example under
[`examples/agents-sdk-coding`](examples/agents-sdk-coding) runs without a key and
writes a reviewed artifact through the daemon broker. Direct SDK tools, MCP
servers, and handoffs are disabled in v1.

## Public live site

The public website package lives in `deploy/public-site`. It is a static marketing and pricing site for NODE, safe for Cloudflare Pages, Netlify, Render, or Vercel because it ships no operator tokens, daemon endpoints, Stripe secrets, SQLite state, or audit bundles.

Use it for the first live deployment while keeping the AgentOS daemon on loopback. See `deploy/public-site/README.md` and `docs/deployment-and-scaling.md` for the deployment boundary.

## Development

```powershell
.\scripts\test.cmd
.\scripts\security-audit.cmd
.\scripts\package.cmd
```

Release readiness lives in `docs\release-readiness.md`; deployment and scaling live in `docs\deployment-and-scaling.md`; billing and metering live in `docs\billing-and-metering.md`; the local-first threat model lives in `docs\security-threat-model.md`; paid pilot operations live in `docs\design-partner-pilot-playbook.md`; uninstall/reset guidance lives in `docs\uninstall-and-reset.md`. Run `scripts\security-audit.cmd` and `scripts\test-pilot-readiness.cmd` before publishing, deploying, or demoing from a shared branch.

Architecture decisions and session memory live under `data/`. Use the commands
in `.Codex/commands/` to reconstruct and close persistent work sessions.

## Continuous learning

The repository integrates ECC Continuous Learning v2 with project-local,
privacy-safe state:

```powershell
.\scripts\learning.cmd sync
.\scripts\learning.cmd status
.\scripts\learning.cmd evolve
.\scripts\learning.cmd export
```

Raw learning state is ignored under `data\learning`. The bridge ingests only
session outcome and count metadata from `data\logs\sessions.jsonl`; it does not
store prompts, source contents, credentials, tool payloads, or model output.
The default reviewed export is `outputs\agentos-instincts.md`.

The matching Codex commands are `/instinct-status`, `/evolve`,
`/instinct-export`, `/instinct-import`, `/promote`, and `/projects`.

## License

NODE is released under the MIT License. See `LICENSE` for details.
