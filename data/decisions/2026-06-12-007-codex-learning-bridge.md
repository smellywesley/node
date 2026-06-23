# Decision: Adapt Continuous Learning v2 to the durable session ledger

**ID:** 2026-06-12-007
**Status:** accepted
**Date (UTC):** 2026-06-11
**Owner:** kernel coordinator
**Review date:** 2026-09-11

## Context

Continuous Learning v2 captures Claude Code `PreToolUse` and `PostToolUse`
hooks. This Codex repository does not expose that hook contract, but it already
has append-only session, decision, cost, and daily ledgers.

Copying the Claude hook configuration would appear enabled while capturing
nothing. Persisting raw Codex prompts or source contents would also violate the
AgentOS memory privacy contract.

## Options Considered

1. Copy the Claude hook configuration and accept that Codex does not invoke it.
2. Add a background observer that records full terminal and conversation data.
3. Bridge durable session metadata into the native Continuous Learning v2
   project store and synchronize at session boundaries.

## Decision

Use option 3. `scripts/learning.cmd sync` maps only structural session metadata
to project-scoped observations, seeds evidence-backed AgentOS instincts, and
delegates status, evolve, import, export, promote, projects, and prune commands
to the skill's native CLI.

Learning state remains local and ignored under `data/learning`. Reviewed
instinct exports are explicit artifacts under `outputs/`.

## Consequences

### Positive

- Learning is deterministic, local, project-scoped, and compatible with Codex.
- Raw prompts, source contents, credentials, and tool payloads are not retained.
- Session start and close make ingestion continuous without a background model.

### Negative

- Session-level observations have less behavioral detail than per-tool hooks.
- The Continuous Learning v2 skill must be installed for management commands.
- Cross-project promotion requires other projects to adopt compatible IDs.

## Evidence

- The bridge regression test proves idempotent ingestion and secret exclusion.
- The skill's 60-test suite passes on Windows after path-validation hardening.
- The current project resolves to scope `308545802bf9`, not the parent profile.

## Supersedes

- None.
