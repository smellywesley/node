# /session-start

Initialize or resume a durable work session.

1. Run `scripts/Initialize-AgentOsMemory.ps1`.
2. Run `scripts/learning.cmd sync` and review instincts at confidence `>= 0.7`.
3. Read `data/projects/current.md`, relevant decisions, and the last 20 session
   ledger entries.
4. Generate a stable session ID: `ses_<UTC compact timestamp>_<short random>`.
5. Select a primary specialist using `AGENTS.md`.
6. Append a `session_started` object to `data/logs/sessions.jsonl`.
7. Report current objective, active constraints, blockers, and planned validation.

Do not rewrite prior ledger entries. Redact secrets from all fields.
