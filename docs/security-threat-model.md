# AgentOS Security Threat Model

Updated: 2026-07-03
Status: v1 local-first threat model; third-party review still required before hosted or multi-tenant claims.

## Scope

This document covers the local AgentOS runtime: CLI, loopback daemon, embedded dashboard, SQLite event store, policy engine, Docker worker runner, OpenAI Agents SDK adapter, billing preview endpoints, replay, and redacted audit export.

Out of scope for v1:

- public hosted control plane;
- multi-tenant SaaS;
- managed model-credit resale;
- remote runner fleet;
- SSO/RBAC for organizations;
- external secrets manager integrations.

The public NODE website is a static site only. It must not expose daemon endpoints, operator tokens, approver tokens, SQLite state, audit bundles, Stripe secrets, webhook secrets, source repositories, or worker APIs.

## Security Goal

AgentOS assumes agent workers are hostile or mistaken. The daemon must keep policy, approval, lifecycle, budget, and audit authority outside the model and outside the worker process.

The first paid pilot is acceptable only as a private/local deployment where the operator understands the current local-first trust model. Hosted enterprise readiness is blocked until the gates in this document are closed.

## Assets

| Asset | Why it matters | Current owner |
|---|---|---|
| Operator token | Authorizes daemon API access and dashboard state reads | Local daemon |
| Approver token | Authorizes approval/denial decisions | Human approver environment |
| SQLite event store | Durable process, approval, usage, billing, and audit state | Local filesystem |
| Repository/workspace mounts | Customer code and artifacts | Docker worker mount policy |
| Declared secrets | Customer provider keys and sensitive inputs | Manifest + runner injection |
| Audit bundles | Evidence for security review; may reveal metadata | Audit export path |
| Stripe configuration | Subscription state and hosted billing links | Server env/static public config split |

## Trust Boundaries

```text
Browser dashboard
  -> loopback HTTP API with operator token
  -> AgentOS daemon
      -> policy and approval engine
      -> SQLite event store
      -> Docker runner boundary
          -> hostile/mistaken agent worker
          -> declared mounts/secrets/network only
      -> redacted audit export
```

Boundary rules:

- The daemon rejects non-loopback API bindings.
- Stateful API calls require the operator token.
- Approval actions require the separate approver token.
- Worker containers must not receive the Docker socket.
- Worker containers default to no network and explicit mounts only.
- Public static site configuration may contain public Payment Links or contact email only; it must never contain Stripe secret keys or daemon credentials.

## STRIDE Review

| Category | Primary risk | Current mitigation | Remaining gate |
|---|---|---|---|
| Spoofing | Stolen dashboard URL or operator token | Random operator token, loopback binding, token rotation | Better dashboard token warnings and team roles |
| Tampering | Agent writes outside intended files | Policy checks, read-only root, declared `filesystem_write`, approval-gated writes | More integration tests for path edge cases across platforms |
| Repudiation | Operator cannot prove what happened | Append-only events, replay, redacted audit export, idempotency keys | Signed audit bundles and retention policy |
| Information disclosure | Secrets or prompts leak into logs/audits | Secret declaration, output redaction, redacted audit export | External redaction review and larger corpus tests |
| Denial of service | Worker or many runs exhaust local Docker/CPU/SQLite | Duration/concurrency budgets, cancellation, Docker cleanup | Backend load report, queue metrics, hard worker-pool telemetry |
| Elevation of privilege | Worker escapes container or abuses host resources | Docker isolation, dropped capabilities, no-new-privileges, no Docker socket | External container-boundary review before untrusted customer workloads |

## Hosted Readiness Blockers

Do not sell or deploy a public hosted control plane until these are complete:

1. Tenant/customer/project identifiers in process state, APIs, audit exports, and billing records.
2. Local roles or team roles: owner, operator, approver, viewer.
3. Billing ledger separate from operational usage totals.
4. Race/concurrency testing in CI for lifecycle, store, and API packages.
5. Load report from bounded concurrent local runs.
6. Pagination/streaming for large process, event, and audit histories.
7. Structured metrics for queue depth, worker duration, approval wait, budget rejection, Docker failures, and audit export size.
8. External review of container isolation, path handling, redaction, and billing webhook handling.

## External Review Packet

Before enterprise security claims, prepare:

- this threat model;
- `docs/backend-load-customer-service-audit.md`;
- `docs/billing-and-metering.md`;
- latest `outputs/backend-load-report.json`;
- latest security audit output;
- latest `go test ./...` and race-test evidence;
- the one-run pay-ready audit bundle with redacted payloads;
- Dockerfile and manifest for the proof run;
- list of known unsupported hosted/multi-tenant behaviors.

## Current Verdict

AgentOS has credible local runtime controls for a private design-partner pilot. It should not be represented as a production hosted enterprise control plane until the hosted readiness blockers above have evidence attached.
