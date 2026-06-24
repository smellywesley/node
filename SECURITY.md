# Security Policy

## Supported Status

AgentOS v1 is currently a local-first developer preview. Use it with repositories and agents you are prepared to inspect. Containers are treated as a security boundary for v1, but host OS, Docker Desktop, and WSL2 configuration remain part of the trusted computing base.

## Security Model

AgentOS keeps lifecycle, policy, approvals, budgets, recovery, replay, and audit authority in the daemon. Agent workers run in containers with explicit mounts, no network by default, dropped capabilities, resource limits, and declared secrets only.

On Windows, AgentOS secures its state directory, operator token, and SQLite database files with current-user-only ACLs. `AGENTOS_HOME` must point at a dedicated state subdirectory, not the user profile root or a filesystem root.

## Reporting

Do not paste secrets, prompts, private source, or raw audit bundles into public issues. For now, report security concerns privately to the project owner before public disclosure.

## Known Boundaries

- Do not mount the Docker socket into workers.
- Do not run untrusted agents with broad writable mounts.
- Treat `agentos dashboard --print-url` output as secret. If it is exposed, stop the daemon, run `agentos rotate-token`, restart the daemon, and reconnect all dashboards.
- The hosted or multi-tenant control plane is not part of v1.

## Security Audit Command

Run `scripts\security-audit.cmd` before public demos or publication. It fails if tracked files include runtime state, release archives, database files, token files, private-key blocks, common provider tokens, or credential-bearing AgentOS dashboard URLs.
