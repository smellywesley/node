# Decision: Approver credential is not persisted with operator state

- Date: 2026-06-11
- Status: accepted

The daemon persists a local operator credential under `AGENTOS_HOME`, but the
approval credential is supplied separately through `AGENTOS_APPROVER_TOKEN`.
Approval listings expose the action, resource, and bound digest while redacting
the payload. This supports running approval commands under a distinct user or
secret source rather than granting the operator token self-approval authority.
