# NODE Five-Minute Proof Demo Script

Updated: 2026-07-07
Status: AppSec buyer recording script for the S$750 private-pilot proof

## Recording Goal

Record a five-minute courtroom exhibit, not a product tour.

The buyer should leave with one belief:

> NODE proves what an AI coding agent was allowed to do, what was blocked, who approved it, what it cost, and what evidence remains after the run.

The demo must not depend on the model misbehaving on camera. The proof works because NODE enforces the boundary outside the model.

## Buyer Lens

Primary buyer: AppSec and security engineering teams approving AI coding agents.

Secondary buyers:

- Platform and DevEx teams rolling out AI coding agents across engineering.
- Regulated software teams that need internal audit evidence.
- AI-native startups scaling agents before governance catches up.

Do not sell this as hosted enterprise SaaS yet. Sell a private/local proof for one risky repository workflow.

## Evidence Claims

Use these only as sourced market context, not as claims about NODE customers:

- Microsoft rollout research associated AI coding-agent adoption with about 24% more merged PRs, while org-scale token spend can reach millions annually: https://arxiv.org/abs/2607.01418
- AIDev found 932,791 agent-authored pull requests across Codex, Devin, Copilot, Cursor, and Claude Code ecosystems: https://arxiv.org/abs/2602.09185
- Claude Code permission-gate testing reported an 81.0% false negative rate in ambiguous authorization scenarios: https://arxiv.org/abs/2604.04978
- AI development-tool research found prompt-injection and tool-abuse risk across MCP/client ecosystems: https://arxiv.org/abs/2603.21642
- GitInject shows agentic GitHub and CI workflows can be attacked through configuration and credential paths: https://arxiv.org/abs/2606.09935

## Before Recording

Start Docker Desktop and run:

```powershell
.\bin\agentos.exe doctor --support
.\scripts\new-pay-ready-proof-packet.cmd
```

Keep these files ready:

- `examples/pay-ready/agent-process.yaml`
- `outputs/pay-ready-proof.md`
- `outputs/pay-ready-audit.json`
- `outputs/pay-ready-proof-transcript.txt`
- `work/pay-ready-workspace/internal/backend_fix.txt`

## Shot List

### 0:00-0:25 - Market Shock

Say:

"AI coding agents are already producing software work. Microsoft-scale rollout research shows measurable PR lift, and public datasets now track hundreds of thousands of agent-authored pull requests. The enterprise question is no longer whether agents can code. It is whether security can prove what they were allowed to do."

On screen:

- Public site hero or proof slide.
- Optional: brief text overlay with the three numbers: 24% PR lift, 932,791 agent PRs, 81.0% permission-gate false negatives.

### 0:25-0:55 - Failure Risk

Say:

"Prompt instructions are not a security boundary. If the agent is told 'do not touch frontend,' that is still just text unless the runtime enforces it. NODE turns that instruction into a policy gate, an approval gate, a cost meter, and an audit bundle."

Show:

- The request: `Fix auth timeout. Do not touch frontend.`
- The controlled execution path: `Request -> Policy Gate -> NODE Control -> Sandbox -> Cost Meter -> Audit Bundle`

### 0:55-1:25 - NODE Control Map

Show `examples/pay-ready/agent-process.yaml`.

Point out:

- `filesystem_write` is scoped to `/workspace/internal`.
- `frontend/**` is denied by policy.
- approval is required for the allowed backend write.
- budget is capped outside the model.
- the worker runs as a supervised process.

Say:

"NODE is not asking the model to be careful. NODE is deciding what the process may do."

### 1:25-2:40 - Live Run

Run or replay:

```powershell
.\scripts\new-pay-ready-proof-packet.cmd
```

Narrate:

- Docker worker starts.
- A managed process is created.
- The agent requests work against a repository-shaped workspace.
- The allowed backend write pauses for approval.
- The forbidden frontend write is denied before it lands.
- Cost and token usage are recorded outside the model.

Required visual evidence:

- process id appears;
- approval event appears;
- `tool.denied` or forbidden write evidence appears;
- cost/tokens are nonzero.

### 2:40-3:35 - Proof Packet

Open `outputs/pay-ready-proof.md`.

Show:

- `Status: PASS`
- managed process succeeded;
- forbidden frontend write denied;
- approval gate exercised;
- nonzero usage/cost;
- audit bundle SHA-256;
- approved backend artifact SHA-256.

Say:

"The proof is not the landing page. The proof is this packet: checks, hashes, events, and artifacts."

### 3:35-4:25 - Audit Evidence

Open `outputs/pay-ready-audit.json` and show event evidence:

- approval event;
- `tool.denied` event;
- `budget.usage_updated` event;
- process state succeeded;
- redacted replayable event trail.

Say:

"This is the AppSec moment. After the run, the team can inspect what happened without trusting a screenshot or model summary."

### 4:25-5:00 - Paid Pilot Ask

Say:

"For S$750, bring one risky coding-agent workflow from your team. We turn it into one controlled proof: one policy boundary, one sandboxed run, one approval gate, one spend cap, one denied action, one audit bundle, and one go/no-go recommendation. Hosted SaaS comes later after tenant isolation, roles, billing ledger, load evidence, and external security review."

Close with:

"NODE is the control layer around the agents your team is already adopting."

## Acceptance Checklist

The recording is acceptable only if it shows:

- one denied action;
- one approval event;
- one nonzero cost or token record;
- one audit export;
- proof packet hashes;
- the S$750 private/local pilot CTA;
- no SOC 2, HIPAA, or hosted SaaS claims.

## After Recording

Upload to Loom, YouTube, or Vimeo, then run:

```powershell
.\scripts\new-pay-ready-proof-recording-brief.cmd -RecordingUrl https://loom.com/share/... -Owner Wesley
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```
