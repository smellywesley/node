# Decision: Constrain the tested Agents SDK dependency graph

**ID:** 2026-06-11-006
**Status:** accepted
**Date (UTC):** 2026-06-11
**Owner:** runtime-dev
**Review date:** 2026-09-11

## Context

An unconstrained `openai-agents` range selected a newer SDK during a clean
container build. A transient metadata timeout then caused resolver backtracking
and made installation slow and nondeterministic.

## Options Considered

1. Allow any pre-1.0 SDK and accept build-time dependency drift.
2. Pin only the top-level SDK package.
3. Pin the tested SDK and constrain its complete known-good dependency graph.

## Decision

Use `openai-agents==0.17.4` and
`adapters/agents-sdk/constraints.txt`, captured from the verified adapter image.
SDK upgrades are explicit changes that must rerun adapter and Docker acceptance
tests.

## Consequences

### Positive

- Builds resolve deterministically without broad dependency backtracking.
- The adapter is tested against a named SDK version.

### Negative

- Security and compatibility updates require deliberate lock refreshes.
- Package availability still depends on the configured Python package index.

## Evidence

- The verified image reported OpenAI Agents SDK 0.17.4 and completed the offline
  coding-agent acceptance flow with 42 accounted tokens.

## Supersedes

- None.
