# Live OpenAI Agents SDK Coding Demo

This demo is the provider-backed proof for the pay-ready roadmap. It runs an OpenAI Agents SDK agent, lets AgentOS account for live token usage, writes the model output through the brokered `fs.write` tool, pauses for approval, then exports an audit bundle.

It requires an OpenAI API key and may spend a small amount of API credit. AgentOS does not print the key, and the manifest injects it only because `OPENAI_API_KEY` is declared as a secret.

Run the complete proof:

```powershell
$env:OPENAI_API_KEY = "sk-..."
.\scripts\demo-live-agents-sdk.cmd
```

Expected proof:

- process ends in `succeeded`;
- `budget.usage_updated` contains nonzero live token and cost accounting;
- a pending approval is created for `/workspace/reviewed/live_add.go`;
- `work\agents-sdk-live-workspace\reviewed\live_add.go` exists after approval;
- `outputs\agents-sdk-live-audit.json` is exported with redacted tool payloads.

The demo manifest uses `gpt-5.4-mini` with the June 24, 2026 standard pricing snapshot from the OpenAI API pricing page. Before using this for production accounting, refresh and store the pricing snapshot for the selected model.