# /session-close

Close the current durable work session.

1. Inspect changed paths and validation results.
2. Append `session_closed` to `data/logs/sessions.jsonl` using the current session
   ID. Include outcome, specialists, changed paths, verification, blockers, and
   one concrete next action.
3. Append one entry to `data/logs/costs.jsonl`. Use `estimate: true` and nullable
   amount/token fields when exact provider usage is unavailable.
4. Append a short reflection to today's file in `data/daily-logs/`.
5. Update only the current-state sections of `data/projects/current.md`.
6. Run `scripts/learning.cmd sync` after the session entry is durable.

Never revise historical log lines to correct a mistake. Append a `correction`
entry referencing the original session ID instead.
