# Continuous Learning v2 Activation Report

## Result

Continuous Learning v2 is active for Agent Process OS using project scope
`308545802bf9`. Local state is stored under ignored `data/learning/`.

## Integration

- `scripts/learning.cmd sync` ingests structural session metadata.
- Session start and close commands invoke synchronization.
- The native skill CLI handles status, evolve, export, import, promote,
  projects, and prune.
- Six matching `.Codex/commands` expose those workflows.
- `AGENTS.md` loads instincts at startup while keeping accepted decisions
  authoritative.

## Initial Instincts

- `broker-consequential-agent-effects` at confidence 0.7
- `redact-durable-agent-records` at confidence 0.7
- `verify-agent-recovery-with-hard-restart` at confidence 0.7

The reviewed export is `outputs/agentos-instincts.md`.

## Privacy

Observations contain timestamps, event IDs, session IDs, outcomes, and counts.
They exclude objectives, prompts, source paths and contents, credentials, tool
payloads, and model output. Raw learning state is excluded from packages.

## Verification

- Continuous Learning v2 upstream suite: 60 passed
- AgentOS Go tests and vet: passed
- Agents SDK adapter tests: 10 passed
- Bridge regression test: passed
- Two concurrent sync jobs: passed
- Unique session observations before this close: 2
- Exported instincts: 3
- Package entries: 93
- Packaged learning-state leaks: 0
- Source archive SHA-256:
  `fddffc9ae5aefe82ec7edc91abaaa370a412167994c96e3729397d40d2d5865c`
