# Reviewed coding agent

This example runs an `Agent` from the OpenAI Agents SDK with a deterministic
offline model, so it needs no provider credential. The model output is routed
through AgentOS as an approval-gated `fs.write`; the worker has only a read-only
workspace mount and cannot write the artifact directly.

```powershell
docker build -f examples\agents-sdk-coding\Dockerfile `
  -t agentos/agents-sdk-coding:local .
New-Item -ItemType Directory -Force work\agents-sdk-coding-workspace
.\bin\agentos.exe run .\examples\agents-sdk-coding\agent-process.yaml
.\bin\agentos.exe approvals
.\bin\agentos.exe approve <approval-id> "reviewed"
```

The approved artifact appears at
`work/agents-sdk-coding-workspace/reviewed/add.go`.
