# NODE Five-Minute Proof Demo Script

Updated: 2026-07-06
Status: recording script for the S$750 private-pilot proof

## Recording Goal

Record evidence that NODE controls an AI coding-agent run instead of asking the model to behave.

Target length: 5 minutes.

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

### 0:00-0:30 - The Problem

Say: "AI coding agents are starting to touch real repositories. Prompt instructions are not a control boundary. NODE wraps the run with policy, sandboxing, approval, cost, replay, and audit evidence."

### 0:30-1:15 - The Request And Policy

Show `examples/pay-ready/agent-process.yaml`.

Point out:

- backend write is allowed;
- frontend write is denied;
- approval is required;
- budget is capped.

### 1:15-2:30 - The Local Run

Run or replay:

```powershell
.\scripts\new-pay-ready-proof-packet.cmd
```

Narrate:

- Docker worker starts;
- managed process is created;
- approval appears;
- backend write is approved;
- forbidden frontend write is blocked.

### 2:30-3:45 - The Proof Packet

Open `outputs/pay-ready-proof.md`.

Show:

- Status: PASS;
- forbidden frontend write denied;
- approval gate exercised;
- nonzero usage/cost;
- audit bundle hash;
- approved backend artifact hash.

### 3:45-4:30 - The Audit Bundle

Open `outputs/pay-ready-audit.json` and show event evidence:

- approval event;
- `tool.denied` event;
- `budget.usage_updated` event;
- process state succeeded.

### 4:30-5:00 - The Buyer Ask

Say: "The S$750 private pilot repeats this proof against one real repository-shaped workflow: your request, your policy boundary, your approval rule, your audit packet. Hosted SaaS comes later after tenant isolation, roles, billing ledger, and external security review."

## After Recording

Upload to Loom, YouTube, or Vimeo, then run:

```powershell
.\scripts\new-pay-ready-proof-recording-brief.cmd -RecordingUrl https://loom.com/share/... -Owner Wesley
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```
