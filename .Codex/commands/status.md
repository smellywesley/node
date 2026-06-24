# /status

Reconstruct Agent Process OS work state from durable memory.

1. Read `data/projects/current.md`.
2. Read the latest relevant decisions.
3. Read the last 20 lines of `data/logs/sessions.jsonl`.
4. Summarize objective, completed work, active work, blockers, decisions, cost
   confidence, dirty paths, and the next executable action.
5. Call out inconsistencies instead of silently choosing one source.

This command is read-only.
