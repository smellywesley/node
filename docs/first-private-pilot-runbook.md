# NODE First Private Pilot Runbook

Updated: 2026-07-06
Status: operator runbook for S$750 entry private pilots

## Commercial Shape

Use the S$750 pilot as an entry proof, not a discounted enterprise rollout.

The buyer gets one focused control story:

- one repository-shaped workflow;
- one policy boundary;
- one sandboxed run;
- one approval gate;
- one spend cap;
- one denied action;
- one exported audit bundle;
- one written go/no-go recommendation.

Do not include hosted multi-tenant backend, SSO/RBAC, managed model credits, SOC 2 claims, HIPAA readiness, or broad custom agent work in the S$750 package.

## Buyer Flow

```text
public site -> pilot-fit intake -> Calendly founder proof call -> written scope -> Stripe payment link or invoice -> local/private proof session -> proof packet follow-up
```

Current entry payment link:

```text
https://buy.stripe.com/bJeeVfc0h6cJ95m9dK7g400
```

## Qualification Gate

Ask before accepting payment:

1. Which coding agents are already touching repositories?
2. What action would be dangerous without approval?
3. Which paths, tools, secrets, or deploy files must stay blocked?
4. Who approves risky writes?
5. What audit evidence would make security or platform say yes?
6. Can they run a local/private Docker and BYOK proof?
7. Can S$750 reserve the first proof if the fit is clear?

Strong fit: they have an agent workflow, an owner, a risky action, local/private acceptance, and willingness to pay.

Weak fit: they need hosted SaaS, SSO, procurement-heavy enterprise rollout, managed model credits, or generic chatbot automation before seeing proof.

## Local Proof Commands

Run from the repo root with Docker Desktop running:

```powershell
.\bin\agentos.exe doctor --support
.\scripts\security-audit.cmd
.\scripts\new-pay-ready-proof-packet.cmd
.\scripts\measure-backend-load.cmd -Count 4 -MaxParallel 2
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```

The proof packet should show:

- managed process succeeded;
- frontend write denied;
- backend write approved;
- nonzero usage/cost;
- audit bundle hash;
- backend artifact hash;
- replay evidence.

## Demo Talk Track

1. Open with the buyer pain: coding agents are entering repos faster than governance can prove control.
2. Show the request: fix backend auth timeout, do not touch frontend.
3. Show the policy: backend allowed, frontend denied.
4. Show the approval gate: consequential backend write pauses for human approval.
5. Show the cost meter: token/cost accounting is outside the model.
6. Show the audit bundle: denial, approval, usage, artifact, replay.
7. Close with the next paid scope: their workflow, their policy boundary, their proof artifact.

## Follow-Up Packet

Send after redaction review:

- `outputs/pay-ready-proof.md`
- `outputs/pay-ready-audit.json`
- `outputs/backend-load-report.json`
- `docs/security-threat-model.md`
- `docs/agentos-architecture-security-audit-2026-07-06.md`

## Production Boundary

This is ready for founder-led private pilots only. Hosted backend production waits for tenant isolation, roles, billing ledger, load evidence, observability, rollback path, and external security review.
