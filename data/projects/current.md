# Project: Agent Process OS v1

## Objective

Build a local-first Go daemon that manages durable containerized coding-agent
processes with inspectable lifecycle, recovery, and cost records.

## Product Invariants

- Local-first operation
- Durable process identity and recoverable state
- Containerized execution with explicit boundaries
- Inspectable lifecycle and cost records

## Current State

**Status:** complete
**Updated (UTC):** 2026-06-11T17:31:50Z
**Active session:** none
**Primary specialist:** runtime-dev

## Scope

### In

- Durable process lifecycle and recovery
- Container execution boundaries
- Local state, logs, and operator workflows

### Out

- Hosted control plane dependency
- Unbounded autonomous execution
- Credentials stored in project memory

## Acceptance Criteria

- [x] Daemon-managed agent processes recover from daemon restart.
- [x] Lifecycle commands and projection replay are auditable.
- [x] Containers run with explicit resource, mount, and network policy.
- [x] Session and provider cost records are append-only.
- [x] Approval-gated effects remain absent until a distinct approver authorizes them.
- [x] Token, duration, concurrency, parent-child, and cancellation policies are tested.
- [x] The OpenAI Agents SDK example produces a reviewed Go artifact from a manifest.
- [x] Simultaneous agents write only to their separate repository workspaces.
- [x] A clean local Windows archive starts successfully after extraction.
- [x] Continuous Learning v2 uses a project-scoped, privacy-safe session bridge.

## Active Decisions

- Per-process internal Docker networks route declared HTTPS destinations through
  an exact-host allowlist proxy.
- External side effects use stable idempotency keys and may resolve to
  `outcome_unknown` after a crash.
- Direct SDK tools, MCP servers, and handoffs are disabled in v1.
- The tested Agents SDK dependency graph is constrained for reproducible images.
- Codex learning synchronizes structural session metadata into local
  Continuous Learning v2 instincts without storing prompts or source contents.

## Blockers

- None for the v1 acceptance scope.

## Next Action

- Use `dist/agentos-v1-windows-amd64.zip` for local installation and load
  `dist/agentos-agents-sdk-coding-local.tar.gz` for an offline first example.
  Review learned behavior with `scripts/learning.cmd status`. Validate the
  optional provider-backed example separately when an `OPENAI_API_KEY` is
  available.
