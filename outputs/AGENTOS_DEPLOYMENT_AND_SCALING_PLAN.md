# AgentOS Deployment And Scaling Plan

Updated: 2026-06-23

## Decision

Ship AgentOS as a trusted local-first runtime before adding hosted scale. The product differentiator is the durable, permissioned, replayable agent process, not a hosted LLM workflow wrapper.

## Current Release Gate

Do not publish a public alpha until the active source tree, packaged archive, and local corrected package agree on the Windows safety fixes:

- `secureTokenPath` must only secure the token file, not reset or rewrite the parent state directory ACL.
- `agentos serve` must reject `AGENTOS_HOME` values that point at the user profile root or filesystem root.
- Regression tests must cover both behaviors.
- The source tree must have a clear repo boundary before release work is treated as canonical.

## Phase 0: Source And Safety Recovery

1. Re-establish the active source tree as the single canonical repository.
2. Backport the corrected Windows ACL and `AGENTOS_HOME` safety fixes.
3. Add or keep regression tests for broad state-home rejection and parent ACL preservation.
4. Run Go tests and vet against `cmd` and `internal` packages.
5. Rebuild the Windows archive from the corrected source only.

## Phase 1: Local Developer Preview

Goal: one developer can install, run the localhost dashboard, execute the offline coding-agent example, approve a write, replay the process, and export an audit bundle in under five minutes.

Required artifacts:

- Signed or checksummed Windows archive.
- Optional offline Docker image bundle for the deterministic example.
- `README.md` quick start.
- `agentos doctor` or equivalent preflight diagnostics.
- `agentos validate <manifest>` for manifest, mount, egress, secret, image, and budget checks.
- Redacted audit export examples.

## Phase 2: Team Beta

Keep execution local. Add optional team coordination:

- daemon enrollment tokens;
- signed policy packs;
- shared approval queue;
- audit bundle indexing;
- OIDC/RBAC for team operators;
- no default upload of source, prompts, secrets, or raw artifacts.

## Phase 3: Runner Fleet

Add runner pools after local trust is proven:

- worker abstraction for Docker, Podman/containerd, then Kubernetes;
- centralized scheduling with local worker event logs;
- tenant isolation and per-run resource ceilings;
- signed manifest bundles and policy versions;
- fleet health, cancellation, and recovery reporting.

## Phase 4: Managed Cloud

Cloud starts as a control plane, not the default process owner. Managed runners should be opt-in after the local and team trust model is boring, documented, and tested.

## Acceptance Suite

A release candidate must prove:

- daemon restart resumes without duplicating committed tool actions;
- undeclared filesystem, network, secret, and tool access is rejected;
- approval-gated actions pause and continue only after approval;
- token, cost, duration, concurrency, and child budgets are enforced;
- parent cancellation propagates to descendants;
- simultaneous agents remain isolated by workspace;
- replay reconstructs state deterministically without side effects;
- audit export is redacted;
- install plus first example completes in under five minutes.

## Council Verdict

Architect: Local-first public preview first, because trust is the product.
Skeptic: Do not disguise cloud ambition as scale; prove portable process state first.
Pragmatist: Distribution quality is the fastest path to adoption.
Critic: Installer, ACL, state directory, and uninstall behavior are part of the security model.

Consensus: ship a boring trusted local runtime, then optional team control plane, then managed runner fleet.
