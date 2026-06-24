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

The pricing values in `agent-process.yaml` are declared manifest inputs. Before using this for production accounting, update them to the current pricing for the selected OpenAI model.