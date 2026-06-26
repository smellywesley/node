# AgentOS Demo-Ready Public Alpha Plan

<!-- Autoplan restore point: .gstack/autoplan/restore-2026-06-23-demo-ready.md -->

Updated: 2026-06-23
Status: in progress
Owner: local developer preview

## Guiding Light

AgentOS should not be another prompt wrapper. The product exists to make an AI agent a managed process with identity, lifecycle, permissions, budgets, recovery, replay, and audit history. Linux and Windows still manage hardware and ordinary processes; AgentOS manages probabilistic agent work.

```text
Developer -> CLI/Dashboard -> authenticated local API -> daemon -> policy -> container worker
                                      |                       |
                                      v                       v
                                SQLite event log        model + tools
```

## Current State

No `plan.md` existed at the start of this autoplan pass, so this file is now the canonical plan. The repo already contains a working Go daemon, CLI, SQLite event store, Docker worker execution, approval flow, budget enforcement, replay, audit export, OpenAI Agents SDK adapter, offline coding example, release scripts, and localhost dashboard. Current hardening work adds token rotation, tracked-file secret scanning, demo-readiness gates, and public-release guardrails.

## CEO Review

### Premise Challenge

The strongest premise is correct: most agent products collapse into an LLM call plus tools, then leave process management to prompts, logs, or a SaaS workflow graph. The differentiator must be the managed agent process, not the model. The risky premise is that developers will immediately understand "AI agent OS"; the product must demonstrate concrete operator value in five minutes: start a daemon, run a containerized task, pause on approval, inspect state, replay history, and prove no secret leaked.

### What Already Exists

| Area | Existing Asset | Leverage |
|---|---|---|
| Runtime | Go daemon, CLI, state projection, SQLite events | Keep v1 small and local-first. |
| Isolation | Docker workers, mount/network/resource policy | Make safety visible in demo. |
| Control | Lifecycle commands, approvals, cancellation | Show the operator loop live. |
| Trust | Replay, audit export, redaction, idempotency records | Position against opaque agent wrappers. |
| Adapter | Offline OpenAI Agents SDK coding example | Proves model/provider replacement. |
| Distribution | Windows package scripts and docs | Good base for public alpha. |

### Dream State Delta

| Horizon | State |
|---|---|
| Current | Local daemon and CLI work, but demo readiness depends on manually remembering security and benchmark gates. |
| Demo-ready | One documented path builds, audits, starts localhost, runs smoke, replays, rotates leaked tokens, and produces a benchmark/audit report. |
| 12-month ideal | AgentOS is the default local control plane developers use before trusting autonomous coding agents with repositories, secrets, budgets, or long-running tasks. |

### Alternatives Considered

| Alternative | Why Not Now |
|---|---|
| Thin SDK wrapper | Easier to ship, but fails the strategic claim. |
| Hosted control plane first | Better collaboration, but much higher security and trust burden. |
| Workflow graph product | Familiar category, but less defensible than durable process semantics. |
| Full custom kernel | Conceptually loud, but unnecessary and misleading for v1. |

### Not In Scope

Hosted multi-tenant control plane, general agent marketplace, arbitrary agent societies, shared global memory, browser automation marketplace, custom kernel, distributed consensus, and host-process execution are outside the demo-ready public alpha.

### Error And Rescue Registry

| Failure | User Impact | Rescue |
|---|---|---|
| Docker unavailable | Agent process cannot run. | Dashboard still starts; `doctor` explains Docker requirement. |
| Token exposed | Dashboard/API credential compromised on loopback. | Stop daemon, run `agentos rotate-token`, restart. |
| Worker crash after action | Operator cannot know whether external action committed. | Record idempotency key and `outcome_unknown`; require reconciliation. |
| Undeclared access requested | Agent blocked. | Emit authorization event and explain missing manifest capability. |
| Package includes runtime state | Public release leaks local data. | Security audit and package inspection fail release. |

### Failure Modes Registry

| Mode | Severity | Mitigation |
|---|---|---|
| Secret in tracked docs/logs | Critical | High-confidence tracked-file scan in `scripts/security-audit.cmd`. |
| Dashboard token copied into public issue | High | Rotation command plus docs warning. |
| Docker socket mounted into worker | Critical | Policy boundary and release checklist ban. |
| Replay accidentally performs effects | Critical | Projection-only replay, no model/tool execution. |
| Parent cancellation misses child | High | Tested cancellation propagation and budget inheritance. |

### CEO Completion Summary

Proceed with a demo-ready public alpha only if security gates are automatic, token exposure has a repair path, and the demo proves process semantics visibly. The market claim should be "the local operating layer for agent processes," not "an operating system that replaces your OS."

## Design Review

The UI scope is the localhost dashboard. It must make invisible safety controls visible without overwhelming a first-time developer.

| Dimension | Score | Decision |
|---|---:|---|
| First impression | 7/10 | Lead with daemon health, process state, approvals, and replay. |
| Empty state | 6/10 | Show next CLI command and sample manifest path. |
| Running state | 8/10 | Process table and inspector are the core demo. |
| Waiting approval | 8/10 | Approval panel must explain capability, digest, and consequence. |
| Error state | 6/10 | Add problem/cause/fix language over time. |
| Security clarity | 7/10 | Avoid showing raw tokens after initial bootstrap. |
| Accessibility/responsive | 6/10 | Keep keyboard and contrast checks in release gate. |

Required dashboard states: unauthenticated, empty, running, succeeded, failed, waiting approval, replay/audit, stale token, and daemon unavailable.

## Engineering Review

### Architecture

```text
manifest -> CLI/API -> daemon -> state machine -> event store -> projection
                       |        -> policy engine -> approvals
                       |        -> budget ledger -> parent/child limits
                       |        -> worker runner -> Docker container
                       |        -> audit/replay -> deterministic view
```

### Scope Challenge

The implementation is appropriately local-first. The main engineering risk is not missing features; it is accidental trust expansion before the security model is boring. The alpha should prefer explicit unsupported states over permissive defaults.

### Test Diagram

```text
CLI parsing -> command tests
manifest parser -> validate examples
API auth -> handler tests + doctor
state machine -> projection/replay tests
policy engine -> undeclared fs/network/secret/tool denial tests
worker runner -> smoke container tests
recovery -> daemon restart test
release -> package audit + security audit
```

The detailed test artifact is `docs/demo-ready-test-plan.md`.

### Engineering Scorecard

| Area | Score | Notes |
|---|---:|---|
| Lifecycle durability | 8/10 | Implemented and evidenced. |
| Security boundaries | 7/10 | Good v1 posture; public release needs repeated audit. |
| Recovery/idempotency | 8/10 | Strong differentiator. |
| Packaging | 7/10 | Windows path works; keep archive scan strict. |
| Observability | 7/10 | Logs/replay/audit exist; dashboard screenshot evidence still missing. |
| CI confidence | 7/10 | Add security audit to CI, keep smoke documented. |

## DX Review

### Developer Journey Map

| Stage | Target Experience |
|---|---|
| Discover | Understand "managed agent process" in one paragraph. |
| Install | Unzip, run `doctor`, start localhost. |
| First run | Run smoke manifest with one command path. |
| Inspect | See process state, logs, approvals, budgets. |
| Trust | Replay and audit without side effects. |
| Recover | Kill/restart daemon and observe non-duplication. |
| Secure | Rotate token and run security audit. |
| Extend | Swap adapter/model while daemon keeps policy authority. |
| Publish | Package passes audit and excludes local state. |

### TTHW Assessment

Current target: under five minutes from release zip to dashboard plus smoke process on a machine with Docker ready. The dashboard alone should start in under one minute. The developer must never need to manually paste an operator token into a public artifact.

### DX Scorecard

| Dimension | Score |
|---|---:|
| Time to hello world | 7/10 |
| CLI naming | 8/10 |
| Error messages | 6/10 |
| Docs findability | 7/10 |
| Copy-paste examples | 7/10 |
| Upgrade/reset path | 6/10 |
| Security guidance | 8/10 |
| Escape hatches | 7/10 |

## Benchmark Standards

### Dashboard Budgets

| Metric | Budget |
|---|---:|
| FCP | < 1.8s |
| LCP | < 2.5s |
| Total JS | < 500KB |
| Total CSS | < 100KB |
| Transfer | < 2MB |
| Requests | < 50 |

### AgentOS Operational Budgets

| Operation | Budget |
|---|---:|
| `agentos doctor` without Docker-ready worker run | < 2s |
| `agentos validate examples\smoke\agent-process.yaml` | < 2s |
| Localhost dashboard start | < 10s |
| Smoke process terminal state | < 30s |
| Security audit findings | 0 |
| Forbidden package entries | 0 |

Benchmark reports are saved under `.gstack/benchmark-reports/` for local history and summarized in `outputs/BENCHMARK_REPORT.md` when a demo run completes.

## Cybersecurity And Leakage Gates

- Security audit must pass before public push or demo branch publication.
- Release archive must exclude `work`, `bin`, `dist` recursion, SQLite DBs, token files, `.git`, `.gstack`, `.agents`, `.codex`, Python bytecode, and local learning state.
- `.Codex/commands/*.md` may be tracked as product command documentation; other `.Codex` runtime state remains blocked.
- No credential-bearing dashboard URL or 64-character token fragment may appear in tracked files.
- Operator and approver credentials stay separate.
- Stateful API calls require same-origin loopback requests and operator auth.
- Approval decisions require the separate approver credential.
- Workers never receive the Docker socket.
- Workers get only declared mounts, egress, tools, and secrets.


## Pay-Ready Roadmap

The product is not pay-ready merely because the local demo runs. The commercial gate is documented in `docs/pay-ready-roadmap.md` and should be treated as the next canonical build target.

Brutal current verdict: AgentOS is a credible prototype with strong primitives, but it becomes sellable only after it proves one narrow buyer story end to end:

```text
Connect repo -> describe coding task -> enforce permissions -> run real agent -> block forbidden action -> produce PR artifact -> show tokens/cost -> export audit bundle
```

Pay-ready P0 requires:

- Real OpenAI/Agents SDK coding run with nonzero live usage accounting.
- Forbidden-file demo where an agent attempts an out-of-scope write and AgentOS denies it.
- GitHub-oriented artifact path: branch or diff, tests, audit bundle.
- Dashboard before/after explanation: without AgentOS vs with AgentOS.
- One-command install/demo path from release zip.

Do not build hosted SaaS, marketplace features, general multi-agent chat, or complex organization management before this P0 proof works.
## Implementation Tasks

- [x] Ship local pay-ready enforcement demo: approval-gated backend write, denied frontend write, nonzero usage/cost, replay, audit export.
- [ ] Add provider-backed OpenAI/Agents SDK coding run with live token/cost tracking. Scaffold exists in `examples/agents-sdk-live-coding`; live verification requires an explicit `OPENAI_API_KEY`.
- [x] Add Stripe-hosted Pro subscription checkout and portal path for BYOK subscriptions; no card data stored by NODE.
- [ ] Add billing/metering ledger before managed usage billing: customer/project/process usage rows, pricing snapshots, spend caps, exports.
- [ ] Add GitHub-oriented artifact flow: branch/diff/test result/audit bundle.
- [x] Add one-command pay-ready demo script and release-zip path.
- [x] Add dashboard before/after explanation for skeptical buyers.

- [x] Add `agentos rotate-token [--force]` without printing the new token.
- [x] Add tracked-file security audit and CI gate.
- [x] Add guided dashboard demo path with prefilled smoke manifest.
- [x] Finish `.Codex/commands` allowlist in audit and `.gitignore`.
- [x] Create and run demo benchmark report.
- [x] Rotate the exposed localhost operator token.
- [x] Rebuild/package and update release evidence.
- [x] Capture dashboard evidence with gstack browser text check.

## Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|
| 1 | CEO | Create `plan.md` because none existed | Auto-decided | durable artifact | The product needs a canonical demo/public-alpha plan on disk. | Keeping plan only in chat. |
| 2 | CEO | Keep v1 local-first | Auto-decided | reduce trust burden | Local developer preview matches current implementation and security posture. | Hosted control plane now. |
| 3 | Eng | Add token rotation | Auto-decided | secure recovery | Exposed dashboard credentials need an operator repair path. | Documentation-only warning. |
| 4 | Eng | Add tracked-file security audit | Auto-decided | automate safety | Public release should fail on known leakage classes. | Manual checklist only. |
| 5 | Eng | Allow `.Codex/commands/*.md` but block other `.Codex` state | Auto-decided | least privilege | Command docs are product source; runtime state is not. | Blocking all `.Codex` content. |
| 6 | DX | Treat benchmark as product and operational gates | Auto-decided | prove the promise | AgentOS must benchmark both dashboard UX and process operations. | Page-speed-only benchmark. |
| 7 | Release | Require demo gates before public push | Auto-decided | trust before distribution | The project is security-sensitive and should not publish unverified local state. | Push first, audit later. |
| 8 | Billing | Start BYOK-first, add managed usage only after a billable ledger | Auto-decided | avoid unbounded spend risk | If AgentOS uses our provider key, we pay OpenAI first and need spend caps plus an append-only billing ledger before customer billing. | Charging only from dashboard totals. |

## Cross-Phase Themes

**Trust must be visible.** CEO, design, engineering, and DX all point at the same requirement: the demo must show why AgentOS is safer and more inspectable than an LLM wrapper.

**Local-first is the wedge.** It avoids cloud trust questions while the process abstraction hardens.

**Security cannot be a doc-only promise.** Token rotation, tracked-file scans, package audits, and replay semantics need to run as commands.

## GSTACK REVIEW REPORT

Status: implemented locally with verification gates passing; ready for user review before public push.

Review scores:

- CEO: 8/10. Strong differentiated thesis; messaging must stay concrete.
- Design: 7/10. Dashboard can demo the operator loop; state-specific polish remains.
- Engineering: 8/10. Runtime architecture supports the claim; release gates must stay strict.
- DX: 7/10. Five-minute first run is plausible; Docker readiness and browser evidence are the main friction.
- Benchmark readiness: 9/10. Dashboard and operational budgets passed with saved evidence.
- Security readiness: 8/10. Audit passed and exposed local token was rotated; hosted threat model remains future work.

Final gate: continue implementation, run tests, rotate exposed local token, benchmark localhost, and publish only after zero leakage findings.
