# AgentOS Pay-Ready Roadmap

Updated: 2026-06-24
Status: planning artifact
Purpose: Convert the brutally honest commercial gap into a build sequence that can produce paid-user evidence.

## Brutal Verdict

AgentOS is not pay-ready yet.

The core thesis is credible: teams will need a control layer for agent processes. The current product already has the right primitives: daemon, CLI, SQLite events, container isolation, approvals, usage accounting, replay, audit export, OpenAI Agents SDK adapter, packaging, and dashboard.

The missing piece is not more architecture. The missing piece is a buyer-grade proof path:

```text
Connect repo -> describe coding task -> enforce permissions -> run real agent -> block forbidden action -> produce PR artifact -> show tokens/cost -> export audit bundle
```

Until that path works in one command and one dashboard story, the product is interesting infrastructure, not something most teams will pay for.

## Target Buyer

Pay-ready v1 should target engineering teams experimenting with AI coding agents, not all agent users.

Best initial buyer:

- Runs coding agents against real repositories.
- Worries about file scope, secrets, network access, and runaway cost.
- Needs auditability for security review or team trust.
- Already pays for AI tooling and has felt agent uncertainty.

Poor initial buyer:

- Solo hobbyist who only wants a chatbot.
- Nontechnical user who expects the OS to make agents smarter.
- Enterprise buyer needing full hosted RBAC and SSO on day one.
- Workflow automation buyer who mainly wants Zapier-style integrations.

## Positioning

Do not lead with "AI operating system" as the paid pitch. It is useful internally, but too abstract for buying.

Lead with:

> AgentOS lets teams run coding agents against repositories with enforced permissions, budgets, approvals, replay, and audit proof.

Shorter:

> Safe execution and audit trails for AI coding agents.

## Pay-Ready Milestones

### P0 - Paid-Interest Demo

Goal: make a skeptical developer believe the security and cost story in under five minutes.

Required features:

- Real OpenAI/Agents SDK coding run with live token and cost tracking.
- Backend-only manifest generated from a natural-language request.
- A forbidden-file demo where the agent attempts a write outside scope and AgentOS blocks it.
- Dashboard before/after panel: without AgentOS vs with AgentOS.
- Exportable audit bundle attached to the run.
- One-command demo script that builds, audits, starts daemon, runs example, and opens dashboard.

Acceptance test:

```text
User runs one command.
Dashboard opens.
Agent tries allowed backend write: approval gate appears.
Agent tries forbidden frontend/docs write: denied event appears.
Usage shows nonzero tokens and computed cost.
Replay returns same state.
Audit export redacts payloads and includes denial/approval/tool records.
```

Commercial meaning: this is enough to collect serious design-partner feedback. It is not enough for broad self-serve paid launch.

### P1 - Payable Local Team Beta

Goal: make a small engineering team willing to pay for local/self-hosted use.

Required features:

- GitHub PR workflow: create branch, run agent, run tests, produce diff, export audit bundle.
- Project profiles: repo path, default safe scopes, model/pricing defaults, approved tools.
- Local users/roles: owner, operator, approver, viewer.
- Shared local history: searchable process list, filters by project/user/state/cost.
- Team-safe token model: separate operator, approver, and read-only dashboard tokens.
- Better error messages: every failure says problem, cause, fix, docs command.
- One-command install and one-command demo from release zip.
- Billable usage ledger for managed usage: customer/project/process rows, pricing snapshots, spend caps, and exportable reports.

Acceptance test:

```text
Two users can inspect the same project history.
Only an approver can approve an approval-gated write.
A coding run produces a branch/diff/test result/audit bundle.
A viewer cannot start, approve, or cancel runs.
A cost cap violation records usage, fails the run, and explains the fix.
```

Commercial meaning: this can support design partners or paid pilots.

### P2 - Scalable Product

Goal: serve many teams without each one assembling its own operational glue.

Required features:

- Optional hosted control plane or self-hosted server mode.
- Organization/project RBAC with SSO later, not before local team beta.
- Remote runner registration with per-run isolation and heartbeat.
- Central audit index and retention policy.
- Billing and license enforcement backed by an append-only usage ledger, not dashboard totals.
- Policy templates for common repo scopes.
- Integrations: GitHub checks, Slack approvals, SIEM/export hooks.

Acceptance test:

```text
Multiple projects run isolated agents concurrently.
Central dashboard shows state, approvals, spend, audit exports, and failures.
Remote runners cannot access undeclared projects or secrets.
Audit trail survives runner loss.
Admins can revoke users and tokens.
```

Commercial meaning: this is where enterprise pricing becomes plausible.

## Feature Priority Matrix

| Feature | Pay Impact | Build Now? | Reason |
|---|---:|---|---|
| Real SDK run with nonzero usage | Critical | Yes | Proves spend visibility and real-agent path. |
| Forbidden-file blocked demo | Critical | Yes | Proves the permission story instantly. |
| GitHub PR workflow | High | Yes, after P0 | Converts safety into developer value. |
| Before/after dashboard | High | Yes | Makes the buyer understand why AgentOS exists. |
| One-command demo | High | Yes | Reduces evaluation friction. |
| Local roles | Medium | P1 | Needed for team beta, not for first proof. |
| Shared audit search | Medium | P1 | Useful after multiple runs exist. |
| Hosted SaaS control plane | High but risky | No | Too much trust burden before wedge proof. |
| Marketplace/general agents | Low now | No | Distracts from the coding-agent buyer. |
| Custom kernel language | Low now | No | Sounds impressive but does not close payment gap. |

## Implementation Plan

### Phase 1 - Proof Demo

1. Add a metered Agents SDK demo that emits real usage frames. Status: scaffolded in `examples\agents-sdk-live-coding`; live verification requires an explicit `OPENAI_API_KEY`.
2. Add a forbidden-write worker that attempts both an allowed and denied write. Status: done in `examples\pay-ready`.
3. Add a pay-ready demo manifest using backend-only permissions. Status: done in `examples\pay-ready\agent-process.yaml`.
4. Add dashboard run summary explaining: allowed, blocked, approved, tokens, cost, audit. Status: in progress through the guided dashboard and pay-ready manifest compiler.
5. Add `scripts\demo-pay-ready.cmd` for one-command local demo. Status: done.
6. Add tests for denied filesystem write and usage/cost display. Status: partially covered by the executable demo verifier; automated unit/e2e coverage still needed.

### Phase 2 - GitHub Artifact Flow

1. Add repo workspace profile with branch name and test command.
2. Add controlled Git branch creation inside the workspace policy.
3. Add test-result capture as process events.
4. Add audit bundle export that includes branch, diff summary, tests, approvals, denials, and usage.
5. Add README walkthrough: run agent -> approve write -> inspect diff -> export audit.

### Phase 3 - Local Team Beta

1. Add local users and roles to SQLite.
2. Split dashboard permissions by role.
3. Add project history filters and process search.
4. Add project-level budget defaults.
5. Add retention/export settings.
6. Add migration tests and upgrade guidance.

## Required Product Proofs

### Proof 1 - AgentOS Blocks What The Prompt Forgets

Scenario:

```text
Request: Fix backend code, do not touch anything else.
Agent attempts: write /workspace/internal/fix.go.
Agent also attempts: write /workspace/web/app.js.
```

Expected:

- Backend write becomes approval-gated or authorized based on policy.
- Frontend write is denied because it is outside declared filesystem_write.
- Dashboard shows a human-readable denial.
- Audit bundle contains the denied action digest and redacted payload.

### Proof 2 - AgentOS Tracks Real Cost

Scenario:

```text
Run a provider-backed Agents SDK task with pricing configured.
Worker emits usage with input_tokens and output_tokens.
```

Expected:

- Dashboard shows nonzero tokens.
- Cost equals configured pricing formula.
- `budget.usage_updated` event is written.
- Run fails if token or cost cap is exceeded.

### Proof 3 - AgentOS Produces A Reviewable Artifact

Scenario:

```text
Run coding agent on repo.
Agent proposes a patch.
Tests run.
Audit bundle exports.
```

Expected:

- Branch/diff/test result are visible.
- Artifact can be reviewed without trusting raw model output.
- Audit bundle explains every consequential action.

## Pricing Hypothesis

Do not price until P0 proof is credible.

Likely model after P1:

- Open-source local runtime: free.
- Pro local: 20-50 USD per user per month.
- Team/self-hosted: 200-2000 USD per month depending on seats/projects/runners.
- Enterprise: custom support, deployment, retention, SSO, compliance exports.

Charging before the P0 demo works will create churn and confusion.

## Hard No List

Do not build these before P0/P1 proof:

- Agent marketplace.
- General social multi-agent chat.
- Hosted SaaS-first control plane.
- Custom kernel or custom programming language.
- Broad GUI builder.
- Memory product detached from execution control.
- Complex organization management before local roles prove useful.

## Definition Of Pay-Ready

AgentOS becomes pay-ready when a skeptical engineering lead can say yes to all of these:

- I saw a real agent run, not a fake smoke task.
- I saw token and cost numbers update from actual usage.
- I saw AgentOS block an out-of-scope action.
- I saw a human approval gate stop a consequential action.
- I saw a GitHub-ready artifact or PR workflow.
- I exported an audit bundle I could give to security.
- I installed and ran the demo without a maintainer guiding me.

Until then, the product should be sold as a design-partner alpha, not a general paid SaaS.
