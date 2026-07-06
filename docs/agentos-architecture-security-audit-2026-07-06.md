# AgentOS Architecture, Security, Benchmark, and Compliance Audit

Date: 2026-07-06
Scope: local AgentOS/NODE runtime, public site boundary, billing preview, proof/demo readiness, and hosted-readiness gaps.

## Verdict

AgentOS is a credible local/private control runtime for AI coding agents. The core primitives are real: loopback daemon, bearer-token local API, separate approver token, append-only SQLite events, brokered tool calls, digest-bound approvals, Docker sandboxing, no network by default, cost/token accounting, replay, and redacted audit export.

It is not yet a bulletproof hosted AI-agent operating system, and it should not be sold as SOC 2 compliant, HIPAA compliant, or production multi-tenant SaaS today. The honest sellable stage is: paid local/private pilots for one controlled repository workflow, with audit evidence that helps a buyer evaluate governance risk.

## Verified Evidence

Commands run from `C:\Users\NewName\Documents\Codex\2026-06-10\i-want-to-buildent-an-operating`:

| Check | Result | Evidence |
| --- | --- | --- |
| Unit/integration tests | PASS | `cmd /c scripts\test.cmd` passed Go packages plus 10 Python tests. |
| Local security audit | PASS | `cmd /c scripts\security-audit.cmd` reported no forbidden tracked paths or high-confidence secret patterns. |
| Pilot readiness report | BLOCKED | `cmd /c scripts\test-pilot-readiness.cmd -AllowBlockers` reported 4 required blockers. |
| Docker engine | BLOCKED | Docker CLI installed, but `dockerDesktopLinuxEngine` pipe is unavailable. |
| Backend load benchmark | BLOCKED today | `scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2` could not run because Docker engine is unreachable. |
| AgentShield external scan | NOT RUN | `npx ecc-agentshield scan` was blocked by approval review because it can fetch/execute third-party code against private repo contents. |

Pilot readiness blockers observed:

1. Generated environment CTA has no real pilot path in the current shell.
2. Public proof demo URL is not configured.
3. Recorded proof demo brief is missing or not marked PASS.
4. Docker Desktop engine is not reachable.

A prior `outputs\backend-load-report.json` exists and reports 4/4 succeeded, but it is stale relative to the current Docker-off state. Treat it as historical evidence, not today's benchmark.

## Strengths

- Local API is not public by default. The docs and deployment boundary correctly state the public website is static and does not expose daemon tokens, SQLite state, audit bundles, worker endpoints, or Stripe secrets.
- API requests require the operator token, browser origins are constrained to loopback, and approval decisions require a separate approver token.
- Manifest validation prevents direct writable mounts, direct network egress, or direct secret access when a matching approval rule would otherwise be bypassed.
- Docker execution is hardened with read-only root, dropped capabilities, no-new-privileges, non-root user, pid/memory/cpu limits, no network by default, and an exact-host egress proxy only when network destinations are declared.
- Tool execution is brokered through idempotency keys and durable tool-call records, with `outcome_unknown` handling for ambiguous side-effect windows.
- Replay is projection-only and side-effect-free.
- Audit export redacts raw approval payloads, tool requests/results, worker stdout/stderr, checkpoints, execution errors, and manifest environment values.
- Billing copy is appropriately conservative: managed usage is locked until tenant isolation, spend caps, and a billable usage ledger exist.

## Findings

### High: Hosted multi-tenant readiness is not implemented

The current runtime has global local credentials and global process, approval, audit, and billing views. The store schema does not partition by tenant, project, workspace, actor, or organization. This is acceptable for a local trusted operator pilot, but it is a hard blocker for hosted SaaS.

Required fix:
Add tenant/project/actor identifiers across process, event, approval, tool-call, audit, and billing records. Every API path must enforce scoped authorization before any hosted runtime is exposed.

### High: Role-based access and approval identity are not audit-grade yet

Separate operator and approver credentials are a strong local primitive, but approvals do not yet carry a durable human principal, role, MFA/session context, source IP/device, or immutable approver identity record. That prevents strong enterprise accountability.

Required fix:
Introduce local team roles first: owner, operator, approver, viewer. Record actor identity on process creation, approval decisions, audit exports, billing changes, and token rotation.

### High: Sensitive task and secret handling assumes trusted local host boundaries

Worker task text is passed through runtime configuration, and declared secrets are injected into the container environment. Host-level Docker inspection and local filesystem access remain part of the trusted boundary. That is fine for local/private pilots, but not for regulated hosted workloads.

Required fix:
Move toward a stronger secret/task transport path: short-lived secret files or brokered secret reads, no broad env persistence, explicit PHI/PII exclusion for pilots, and documented host trust assumptions.

### High: Billing is preview-grade, not a full commercial ledger

Stripe checkout exists, signed webhooks are handled, and webhook events are idempotently recorded. However, this is not a tenant-scoped billing ledger. Managed model usage is explicitly locked, and checkout readiness is not equivalent to complete subscription, entitlement, refund, tax, invoice, or usage-ledger correctness.

Required fix:
Before self-serve billing, require webhook configuration for checkout readiness, add tenant-scoped entitlements, add a billable-usage ledger separate from operational usage totals, make failed webhook processing retryable, and prevent unknown-customer collisions.

### Medium: Process/event/audit reads are unbounded

Process lists, event histories, approval lists, tool calls, replay, and audit exports load complete histories. This is manageable for small local demos, but it can become a reliability and denial-of-service issue for team or hosted usage.

Required fix:
Add pagination, streaming export, retention limits, and audit bundle size controls before team beta.

### Medium: Observability is not enough for production operations

The daemon has useful state transitions and proof scripts, but production operations need structured metrics and alertable telemetry: queue depth, worker duration, Docker failures, approval wait time, budget rejection, audit export size, webhook failures, and recovery events.

Required fix:
Add local metrics first, then hosted telemetry only after tenant isolation exists.

### Medium: Audit bundles are useful but not tamper-evident

Replay and redacted audit exports are valuable for buyer proof. They are not yet signed, hash-chained, or retention-managed evidence artifacts.

Required fix:
Hash-chain event records and sign exported audit bundles. Include packet hash, tool-call digests, policy version, approval actor, redaction version, and verification instructions.

### Medium: Benchmarking is not yet automated as a standing gate

Tests pass, and a backend load script exists. The current machine cannot run Docker-backed benchmarks because Docker Desktop is stopped or unavailable. Benchmark status should be a repeatable release gate, not a one-off artifact.

Required fix:
Add a benchmark target that runs: denial correctness, approval integrity, cost accounting, replay determinism, restart recovery, Docker containment, redaction leakage, and bounded concurrent load. Store JSON and markdown summaries under `outputs/`.

### Medium: Release and dependency gates need stronger evidence

CI runs tests and the local security audit, but release publishing should explicitly depend on test/security success or branch/tag protection evidence. Dependency vulnerability posture was not fully verified in this pass because networked package-audit tools can disclose dependency inventory or fetch third-party code.

Required fix:
Gate release artifacts on tests, security audit, race-sensitive backend packages, and approved dependency scanning. Add documented approval paths for `npm audit`, `govulncheck`, and `pip-audit` when dependency inventory can be shared.

### Low/Medium: Dashboard URL fragments should avoid approver-token preload

The dashboard correctly treats printed URLs as sensitive, and generated URLs should include only the operator token. The frontend still has code to read an `approver_token` URL fragment into session storage, which increases the chance of accidental combined operator/approver sharing.

Required fix:
Remove approver-token URL preload unless there is a narrow automation requirement, and keep the approver credential entered interactively or through a stronger local secret path.

### Low: Marketing can say "enterprise-grade primitives," not "enterprise-ready platform"

The public site can confidently say NODE gives local/private agent runs policy, sandboxing, approval, spend tracking, replay, and audit evidence. It should avoid implying hosted SOC 2/HIPAA readiness until those controls are implemented and reviewed.

## SOC 2 Readiness

SOC 2 is an independent assurance report over controls relevant to Trust Services categories such as security, availability, processing integrity, confidentiality, and privacy. AgentOS has useful building blocks for the Security and Processing Integrity story, but it does not have the operating evidence or hosted control environment required to claim SOC 2 readiness.

Current fit:
- Security: partial local controls exist through token auth, loopback binding, policy enforcement, and Docker isolation.
- Processing integrity: partial controls exist through event logs, idempotency keys, policy versions, replay, and audit exports.
- Confidentiality: partial controls exist through redaction and no public daemon exposure.
- Availability: early local recovery exists, but no SLOs, incident process, backups, monitoring, or hosted operational evidence.
- Privacy: not in scope unless personal data is processed; no privacy control program is present.

SOC 2 blockers:
- tenant and actor identity;
- role-based access control;
- change management evidence;
- monitoring and incident response;
- vulnerability management;
- retention and deletion policies;
- signed/tamper-evident audit bundles;
- vendor and subprocessors register;
- external security review.

Safe claim today:
"NODE produces audit evidence that can support SOC 2-style internal review for local/private AI-agent pilots. NODE is not SOC 2 certified."

## HIPAA Readiness

HIPAA Security Rule compliance requires administrative, physical, and technical safeguards for electronic protected health information. AgentOS should not process PHI today unless a formal HIPAA program, covered-entity/business-associate scoping, BAA posture, access controls, audit controls, integrity controls, person/entity authentication, transmission security, retention, and incident procedures are in place.

HIPAA blockers:
- no PHI data classification or exclusion control;
- no BAA posture for cloud/model/provider dependencies;
- no actor-scoped access log suitable for PHI access accounting;
- no tenant isolation or role-based access control;
- no documented encryption/key-management program;
- no retention/deletion workflow;
- no HIPAA risk analysis evidence;
- no breach/incident operating procedure.

Safe claim today:
"NODE is not HIPAA-ready. For pilots, do not use PHI."

## Benchmark Plan

Run this once Docker Desktop is running:

```powershell
.\scripts\build.cmd
.\scripts\security-audit.cmd
.\scripts\demo-pay-ready.cmd
.\scripts\new-pay-ready-proof-packet.cmd
.\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```

Benchmark scorecard:

| Dimension | Passing evidence |
| --- | --- |
| Policy denial | forbidden frontend write denied before landing |
| Approval integrity | allowed consequential write pauses and resumes only after approver decision |
| Sandbox containment | worker uses locked-down Docker settings and cannot access undeclared mounts/network/secrets |
| Cost accounting | nonzero token/cost usage recorded and budget cap enforced |
| Replay determinism | replay reconstructs process state without side effects |
| Recovery | restart recovery emits recovery event and reconciles ambiguous tool state |
| Redaction | audit export contains redacted payloads and no raw secrets/tool data |
| Load | bounded concurrent smoke runs all reach terminal success |
| Supportability | support bundle exports only health plus redacted audit data |

## Demo and Video Plan

The proof demo should show one complete run, not a generic landing-page animation:

1. Start Docker Desktop and wait until `docker version` shows server details.
2. Run `scripts\demo-pay-ready.cmd`.
3. Show the request: "Fix backend auth timeout. Do not touch frontend."
4. Show the contract: request, policy gate, NODE control, sandbox, cost meter, audit bundle.
5. Approve the allowed backend write.
6. Show the forbidden frontend write denied.
7. Show nonzero token/cost usage.
8. Export the redacted audit bundle.
9. Run replay and show side effects are false.
10. Generate the proof packet and recording brief.

After recording:

```powershell
.\scripts\new-pay-ready-proof-recording-brief.cmd -RecordingUrl <public-or-private-demo-url>
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```

## Next Implementation Steps

1. Fix operational blockers: start Docker Desktop, configure CTA environment for local readiness checks, add the proof demo URL, and generate the recording brief.
2. Add standing benchmark output: run backend load and save fresh `outputs\backend-load-report.json` plus a markdown summary.
3. Add team-local roles: owner, operator, approver, viewer.
4. Add tenant/project/actor IDs in the store and API response model before any hosted backend work.
5. Add pagination/streaming for events, approvals, tool calls, audit, and replay.
6. Add signed audit bundle and event hash-chain design.
7. Add billing ledger design before self-serve Pro: tenant entitlements, subscription state, webhook retries, and billable usage records.
8. Schedule external security review of container isolation, path handling, redaction, and webhook handling.
9. Only after these gates, revisit hosted deployment of the runtime backend.

## Sources

- HHS, HIPAA Security Rule: https://www.hhs.gov/hipaa/for-professionals/security/laws-regulations/index.html
- HHS, HIPAA Security Rule overview: https://www.hhs.gov/hipaa/for-professionals/security/index.html
- AICPA & CIMA, SOC suite of services: https://www.aicpa-cima.com/resources/landing/system-and-organization-controls-soc-suite-of-services

