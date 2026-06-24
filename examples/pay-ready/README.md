# Pay-Ready Demo

This demo proves the narrow buyer story from `docs/pay-ready-roadmap.md`:

```text
describe coding task -> enforce permissions -> approve allowed write -> block forbidden write -> show tokens/cost -> export audit
```

The worker intentionally does three things:

- emits a nonzero usage frame: 1,800 tokens and 0.003 USD using the manifest pricing;
- requests an approved backend write to `/workspace/internal/backend_fix.txt`;
- requests a forbidden frontend write to `/workspace/web/app.js`, which AgentOS denies.

Run the complete local proof:

```powershell
.\scripts\demo-pay-ready.cmd
```

The script builds the worker image, starts an isolated local daemon, creates the process, approves the backend write, waits for completion, replays state, exports `outputs\pay-ready-audit.json`, and verifies that the forbidden frontend file was not created.

Manual flow:

```powershell
.\scripts\build.cmd
docker build -f .\examples\pay-ready\Dockerfile -t agentos/pay-ready-demo:local .
New-Item -ItemType Directory -Force .\work\pay-ready-workspace\internal, .\work\pay-ready-workspace\web
.\bin\agentos.exe validate .\examples\pay-ready\agent-process.yaml
.\bin\agentos.exe run .\examples\pay-ready\agent-process.yaml
.\bin\agentos.exe approvals
.\bin\agentos.exe approve <approval-id> "reviewed backend-only write"
.\bin\agentos.exe inspect <process-id>
.\bin\agentos.exe logs <process-id>
.\bin\agentos.exe replay <process-id>
.\bin\agentos.exe audit <process-id> .\outputs\pay-ready-audit.json
```

Expected proof:

- process ends in `succeeded`;
- `budget.usage_updated` contains nonzero token and cost accounting;
- `tool.denied` exists for `/workspace/web/app.js`;
- `work\pay-ready-workspace\internal\backend_fix.txt` exists;
- `work\pay-ready-workspace\web\app.js` does not exist;
- audit export redacts action payloads.