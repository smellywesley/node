# Decision: Explicit ambiguous outcomes for external effects

- Date: 2026-06-10
- Status: accepted

AgentOS does not claim universal exactly-once external side effects. Every tool
request has a stable idempotency key and durable requested, authorized,
started, and terminal records. If the daemon cannot prove whether an external
effect committed before interruption, the terminal result is
`outcome_unknown`. Operators or tool-specific reconciliation must resolve it;
the scheduler must not blindly retry it.
