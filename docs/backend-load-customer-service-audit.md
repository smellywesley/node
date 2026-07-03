# NODE Backend Load And Customer-Service Audit

Updated: 2026-06-25
Status: initial audit artifact
Scope: local-first AgentOS/NODE daemon, API, SQLite state, runner, dashboard, and public-alpha support posture.

## Executive Verdict

NODE is suitable for a local developer preview, but it is not yet ready to serve multiple paying teams as a hosted or shared service. The backend has strong process-control primitives, but customer-grade load and support readiness require explicit limits, metrics, queue visibility, role separation, and operational runbooks.


## Independent Audit Findings Added 2026-06-25

- P0 tenant isolation gap: the current daemon has global operator/approver credentials and global process/approval views. Paid team usage requires customer, project, actor, and role fields in state, APIs, audit exports, and billing records.
- P0 public-release credential gap: dashboard launch URLs must not preload the approver credential. Operators can open the console with the operator token; approver tokens stay separate and are requested only when an approval action is taken.
- P1 load gap: process/event/audit reads need pagination or streaming before customer-scale histories, otherwise large event logs can pressure memory and make support workflows slow.
- P1 worker containment gap: stdout/stderr capture, long lines, approval polling, and long-running workers need explicit memory, timeout, and goroutine ceilings before hosted or multi-team operation.
- P1 scheduling gap: the local scheduler is not tenant-fair. Design-partner previews should state local-first scope; hosted service requires per-tenant queueing and concurrency policy.
- P1 sensitive task transport gap: task payloads should move from environment variables toward stdin or mounted request files before handling customer secrets or proprietary prompts.
- P2 observability gap: add structured metrics for queue depth, worker duration, approval wait time, budget rejection, Docker failures, and audit export size.
- P2 public repo hygiene: scrub local telemetry files, align license language with source-available expectations, and keep unreviewed local audit drafts out of packaged releases.
## What Works Today

- Local daemon and CLI provide a clear operator path.
- SQLite event log gives durable process history for local use.
- Containers isolate worker execution from the host process.
- Approval gates, replay, audit export, and budget accounting are meaningful trust primitives.
- Security audit script reduces accidental public-release leakage.

## Load Readiness Risks

| Priority | Risk | Impact | Recommendation |
|---|---|---|---|
| P0 | SQLite is authoritative and local-only | Fine for one machine, weak for many concurrent teams or hosted scale | Keep v1 local; add explicit queue limits and document max concurrent workers. |
| P0 | Worker concurrency can exhaust CPU/RAM/Docker resources | Multiple long-running agents can degrade the operator machine | Add per-run resource metrics, queue depth, and hard worker pool telemetry. |
| P1 | No customer/project partitioning in the local API model | Hard to support team history, billing, or support tickets | Add project IDs to process views and audit exports before team beta. |
| P1 | Limited service-health diagnostics | Support cannot quickly explain daemon, Docker, DB, or policy failures | Expand `doctor` and dashboard health into problem/cause/fix checks. |
| P1 | Dashboard totals are not billing-grade | Customers may confuse estimated cost with amount owed | Keep billing copy as preview until append-only billable ledger exists. |

## Customer-Service Readiness Risks

| Priority | Risk | Impact | Recommendation |
|---|---|---|---|
| P0 | Errors need clearer recovery text | Users will ask support instead of self-solving | Standardize errors as problem, cause, fix, docs command. |
| P0 | Credential/token rotation must be obvious | Exposed local URLs create support/security incidents | Keep `rotate-token`; add visible docs link in dashboard auth/help. |
| P1 | No support bundle command | Hard to debug customer machines | Add `agentos support-bundle` exporting redacted health, config, recent events, and versions. |
| P1 | No role-based local users yet | Team beta cannot cleanly separate viewer/operator/approver | Implement local users/roles before paid local beta. |
| P2 | Hosted service requires tenant isolation | Public SaaS would add major trust burden | Do not ship hosted control plane before P1 local team beta. |

## Load Test Recommendations

1. Add a deterministic local stress script that starts N smoke/pay-ready runs with bounded concurrency.
2. Measure process creation latency, queue wait time, terminal success/failure count, SQLite write errors, and worker cleanup.
3. Add dashboard/CLI display for queue depth, active workers, terminal failures, and last daemon recovery.
4. Capture results in `outputs/backend-load-report.json` for release evidence.

Implemented script target: `scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2`.
The script starts an isolated daemon, runs bounded concurrent protocol-smoke processes, and writes `outputs\backend-load-report.json`. Docker must be running.

## Support Runbook Recommendations

1. `agentos doctor --support` should verify daemon health, DB open, Docker availability, image availability, token status, disk space, and recent failed events.
2. `agentos support-bundle <process-id>` should export redacted process metadata, event summaries, manifest capabilities, budget usage, daemon version, and OS/Docker versions.
3. Audit exports should include clear denial and approval summaries for non-engineering stakeholders.

## Commercial Gate

NODE can support design-partner demos now. It should not claim hosted enterprise readiness until the following exist:

- Project/customer identifiers throughout process state and audit exports.
- Local roles: owner, operator, approver, viewer.
- Billing ledger separate from operational usage totals.
- Load report from concurrent runs.
- Redacted support bundle command.
- Clear support docs for token rotation, Docker failure, DB corruption, and replay/audit export.
