# NODE Design Partner Pilot Playbook

Updated: 2026-07-03
Status: operating playbook for paid design-partner pilots, not broad self-serve launch.

## Goal

Secure 3-5 serious design partners who already use or are actively evaluating AI coding agents, then prove that NODE makes those agents safer to run against real repositories.

This playbook keeps the commercial motion narrow:

> Safe execution and audit trails for AI coding agents.

Do not pitch NODE as a broad enterprise operating system, hosted SaaS, agent marketplace, or managed model-credit platform yet.

## Ideal Design Partner

| Signal | Good fit | Poor fit |
|---|---|---|
| Team | Platform, DevEx, AppSec, Security Engineering, AI tooling | Solo hobbyist or generic chatbot team |
| Current behavior | Running Cursor, Claude Code, Devin, Copilot agents, or internal repo agents | Still debating whether to allow AI coding tools at all |
| Pain | File scope, secrets, approvals, cost, audit, policy evidence | Wants a smarter model or generic workflow automation |
| Environment | Can run a local/private proof with Docker and BYOK | Requires hosted SSO/RBAC before any evaluation |
| Buying motion | Can sponsor a paid pilot or security review | Wants free consultancy or broad custom build |

## Qualification Questions

Ask these before promising a pilot:

1. Which AI coding agents are already touching real repositories?
2. What is the scariest action an agent could take in your environment?
3. Who is accountable for approving agent-created code or file writes?
4. Do you need audit evidence for SOC 2, internal policy, AppSec, or legal review?
5. Can you run a local/private Docker-based proof with your own model keys?
6. What would make this pilot worth paying for inside 30 days?

Disqualify or defer when:

- they require public hosted SaaS on day one;
- they want NODE to provide managed model credits;
- they cannot share any local/private test repository;
- they only want generic agent productivity, not governance.

## Pilot Offer

Package the first pilot as a private proof, not a platform subscription.

Recommended shape:

- Duration: 2-4 weeks.
- Deployment: local/private only.
- Model posture: BYOK.
- Scope: one repository, one agent workflow, one controlled run story.
- Deliverables:
  - 5-minute proof packet from `scripts\new-pay-ready-proof-packet.cmd`;
  - redacted audit bundle;
  - denied forbidden write;
  - approved in-scope write;
  - nonzero token/cost accounting;
  - replay evidence;
  - security/readiness notes and next blockers.

Do not include:

- hosted multi-tenant control plane;
- SSO/RBAC promises;
- managed model-credit billing;
- broad enterprise rollout;
- custom agent marketplace work.

## Demo Flow

Before the call:

```powershell
.\scripts\security-audit.cmd
.\scripts\new-pay-ready-proof-packet.cmd
.\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2
```

The call:

1. Open with the problem: coding agents are entering repositories faster than governance is catching up.
2. Show the public site positioning: safe execution and audit trails for AI coding agents.
3. Run or replay the proof packet:
   - allowed backend write pauses for approval;
   - forbidden frontend write is denied;
   - usage/cost is recorded;
   - replay reconstructs state;
   - audit bundle is exported and redacted.
4. State the non-claims:
   - local/private only;
   - not hosted enterprise SaaS yet;
   - managed model usage remains locked;
   - tenant roles and billing ledger are future gates.
5. Ask for their real workflow and one repository-shaped pilot scenario.

## Outreach Template

Subject: Safer AI coding-agent runs for your repos

Hi {name},

I am building NODE, a local/private control layer for AI coding agents. It gives each agent run explicit file/tool boundaries, approval gates, spend tracking, replay, and a redacted audit bundle.

The wedge is narrow: teams already letting coding agents touch repositories, but needing security or platform proof before usage spreads.

I am looking for 3-5 design partners to run a paid private pilot around one controlled repository workflow. The proof is simple: an agent tries an allowed backend write, a forbidden write is blocked, a human approval gate is exercised, cost is tracked, and an audit bundle proves what happened.

Worth a 25-minute fit check?

## Follow-Up Template

Subject: NODE pilot fit notes

Thanks for taking the time today.

What I heard:

- Current agent workflow: {workflow}
- Main governance concern: {concern}
- Required proof: {proof}
- Pilot blocker: {blocker}

Recommended next step:

- Run the local/private proof against {repo or sample repo}.
- Capture `outputs\pay-ready-proof.md` and the redacted audit bundle.
- Decide whether the next value is policy templates, GitHub artifact flow, or local team roles.

Important boundary: NODE is ready for local/private design-partner proof, not public hosted enterprise rollout yet.

## Pilot Scorecard

| Dimension | Score 0-2 | Notes |
|---|---:|---|
| Already using coding agents against repos |  |  |
| Has clear security/platform owner |  |  |
| Has urgent audit or approval pain |  |  |
| Can run local/private Docker proof |  |  |
| Can bring own model keys |  |  |
| Can pay for a pilot |  |  |
| Accepts hosted SaaS is not ready yet |  |  |

Interpretation:

- 11-14: strong pilot candidate.
- 7-10: nurture; run proof only if workflow is sharp.
- 0-6: do not customize yet.

## Evidence Packet To Send

Send only after removing secrets and confirming the buyer can receive it:

- `outputs\pay-ready-proof.md`
- `outputs\pay-ready-audit.json`
- `docs\security-threat-model.md`
- `docs\backend-load-customer-service-audit.md`
- `docs\billing-and-metering.md`
- short note listing unsupported hosted/team features.

## Success Criteria

A design-partner pilot is successful when:

1. The partner runs or reviews the proof against a repository-shaped workflow.
2. The denied action and approval gate are meaningful to their security/platform team.
3. The audit bundle answers a real internal review question.
4. They identify the next paid feature from evidence, not speculation.
5. They accept the local/private boundary or name the exact hosted blockers.

## What To Learn

Track these after every call:

- Which agent/tool are they using?
- Which action scares them most?
- Who owns approval?
- What artifact would satisfy their security review?
- What would they pay for now?
- What blocks a pilot?
- What feature request is actually a hosted-readiness blocker?

Do not generalize from compliments. Only count:

- paid pilots;
- concrete repository workflows;
- repeated security/audit pain;
- willingness to run the proof;
- willingness to name an owner and timeline.
