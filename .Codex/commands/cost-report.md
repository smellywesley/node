# /cost-report

Summarize provider usage from `data/logs/costs.jsonl`.

1. Validate each nonblank line as an independent JSON object.
2. Group by UTC day, provider, model, specialist, and session.
3. Sum known `amount_microusd` values separately from estimated or unknown costs.
4. Report input, output, cached, and reasoning tokens when present.
5. Flag duplicate entry IDs, missing session IDs, and malformed lines.

This command is read-only. Never infer an exact currency total from token counts
without a recorded price basis.
