# NODE First Private Pilot Runbook

Updated: 2026-07-07
Status: operator runbook for S$750 entry private pilots

## Commercial Shape

Use the S$750 pilot as an entry proof, not a discounted enterprise rollout.

Primary buyer: AppSec and security engineering teams that need evidence before approving AI coding agents.

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

## Five-Minute Proof Promise

Use this sentence to frame every fit call:

> Bring one risky coding-agent workflow. NODE will prove what the agent was allowed to do, what was blocked, who approved it, what it cost, and what audit evidence remains.

This is a courtroom exhibit, not a product tour. The buyer should see:

- one denied action;
- one approval event;
- one nonzero cost/token record;
- one audit export;
- proof packet hashes;
- the S$750 private/local pilot ask.

Do not rely on the model misbehaving. The proof works because NODE enforces policy outside the model.

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

1. Open with the buyer pain: coding agents are already producing work, but security needs proof before rollout.
2. Show the failure risk: prompt instructions are not an enforceable boundary.
3. Show the control path: Request -> Policy Gate -> NODE Control -> Sandbox -> Cost Meter -> Audit Bundle.
4. Show the request: fix backend auth timeout, do not touch frontend.
5. Show the policy: backend allowed, frontend denied.
6. Show the approval gate: consequential backend write pauses for human approval.
7. Show the cost meter: token/cost accounting is outside the model.
8. Show the audit bundle: denial, approval, usage, artifact, replay, hashes.
9. Close with the next paid scope: their workflow, their policy boundary, their proof artifact.

## Market Evidence To Reference

Use these only as sourced context:

- Microsoft rollout research associated AI coding-agent adoption with roughly 24% more merged PRs, while token spend at organization scale can reach millions annually: https://arxiv.org/abs/2607.01418
- AIDev found 932,791 agent-authored pull requests across Codex, Devin, Copilot, Cursor, and Claude Code ecosystems: https://arxiv.org/abs/2602.09185
- Claude Code permission-gate testing reported an 81.0% false negative rate in ambiguous authorization scenarios: https://arxiv.org/abs/2604.04978
- AI development-tool research found prompt-injection and tool-abuse risks across MCP/client ecosystems: https://arxiv.org/abs/2603.21642
- GitInject shows agentic GitHub and CI workflows can be attacked through configuration and credential paths: https://arxiv.org/abs/2606.09935

## Follow-Up Packet

Send after redaction review:

- `outputs/pay-ready-proof.md`
- `outputs/pay-ready-audit.json`
- `outputs/backend-load-report.json`
- `docs/security-threat-model.md`
- `docs/agentos-architecture-security-audit-2026-07-06.md`

## Production Boundary

This is ready for founder-led private pilots only. Hosted backend production waits for tenant isolation, roles, billing ledger, load evidence, observability, rollback path, and external security review.
