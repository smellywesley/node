# Contributing

AgentOS is early. Keep changes small, auditable, and aligned with the local-first process-supervisor thesis.

## Development Checks

```powershell
.\scripts\test.cmd
.\scripts\build.cmd
.\scripts\package.cmd
```

Before changing release, state, policy, or security behavior, also run:

```powershell
.\bin\agentos.exe doctor
.\bin\agentos.exe validate .\examples\agents-sdk-coding\agent-process.yaml
```

## Change Rules

- Preserve local-first operation.
- Keep SQLite authoritative for v1 state.
- Do not weaken approval, budget, idempotency, or replay guarantees.
- Do not store prompts, secrets, raw source, or credentials in project memory.
- Add tests for release, state, and security regressions.
