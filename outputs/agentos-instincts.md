# Instincts export
# Date: 2026-06-12T01:33:10.099100
# Total: 3
# Scope: project
# Project: i-want-to-buildent-an-operating (308545802bf9)

---
id: broker-consequential-agent-effects
trigger: "when adding an agent action that changes host or external state"
confidence: 0.7
domain: security
source: project-memory
scope: project
project_id: 308545802bf9
project_name: i-want-to-buildent-an-operating
---

# broker-consequential-agent-effects

## Action
Route the effect through the daemon broker with declared capability, policy evaluation, stable idempotency key, approval when required, and append-only audit events.

## Evidence
- SDK tool-boundary decision, approval-principal decision, and brokered fs.write acceptance coverage.

---
id: redact-durable-agent-records
trigger: "when persisting agent logs, audits, session records, or costs"
confidence: 0.7
domain: security
source: project-memory
scope: project
project_id: 308545802bf9
project_name: i-want-to-buildent-an-operating
---

# redact-durable-agent-records

## Action
Persist structural metadata while redacting secrets, prompts, source contents, mount sources, environment values, and consequential payloads.

## Evidence
- Memory contract, redacted audit acceptance, and cost-ledger privacy invariant.

---
id: verify-agent-recovery-with-hard-restart
trigger: "when changing agent process lifecycle or recovery"
confidence: 0.7
domain: testing
source: project-memory
scope: project
project_id: 308545802bf9
project_name: i-want-to-buildent-an-operating
---

# verify-agent-recovery-with-hard-restart

## Action
Verify a hard daemon restart from a durable checkpoint, exactly one recovery transition, and no repeated committed tool action.

## Evidence
- Accepted recovery decision, focused recovery tests, and the v1 hard-kill acceptance run.

