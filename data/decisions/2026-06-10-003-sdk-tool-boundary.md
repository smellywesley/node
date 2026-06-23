# Decision: SDK tools are pure by default

- Date: 2026-06-10
- Status: accepted

The OpenAI Agents SDK adapter does not execute direct SDK tools, MCP servers, or
handoffs in v1. Effects use the AgentOS broker so capability, approval,
idempotency, cancellation, and audit policy remain authoritative. The first
brokered implementation is approval-gated `fs.write`.
