# /recover

Recover context after interruption or daemon/session failure.

1. Run the memory initializer; it must not overwrite existing state.
2. Read project state, recent decisions, and the tail of both JSONL ledgers.
3. Identify sessions with `session_started` but no later terminal event.
4. Inspect repository and runtime state before assuming an operation completed.
5. Append a `session_recovered` or `session_abandoned` event with evidence.
6. Resume only idempotent work; request confirmation before destructive actions.
