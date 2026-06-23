# AgentOS Localhost Proof And Improvement Plan

## Product Reframe

AgentOS v1 already has the hard runtime foundations. The next risk is not missing
kernel features; it is that developers cannot see or trust those features
quickly. The next product phase should turn durable identity, policy, approvals,
budgets, recovery, and replay into an observable localhost experience.

The primary promise is:

> Run an AI agent as a managed process, watch what authority it has, interrupt
> it safely, approve consequential effects, and prove afterward exactly what
> happened.

## Premises

1. Local coding-agent developers are the first user.
2. A visible proof loop is more valuable now than distributed scheduling.
3. The daemon remains authoritative; the dashboard is only an operator client.
4. The separate approval credential remains a hard product invariant.
5. Docker remains the v1 worker boundary, but dashboard inspection must work
   when Docker is unavailable.

## What Already Exists

- Durable SQLite process identity and append-only events
- Explicit lifecycle transitions and restart recovery
- OCI container execution with mount, network, secret, and resource policy
- Capability evaluation and digest-bound single-use approvals
- Idempotent tool records and explicit ambiguous outcomes
- Token, cost, duration, concurrency, and child ceilings
- Parent-child cancellation and budget propagation
- Projection replay and redacted audit export
- OpenAI Agents SDK adapter and offline coding-agent example

## Not In Scope

- Hosted control plane
- Multi-machine scheduling or consensus
- General agent social networks or unrestricted messaging
- Replacing the host OS or container runtime
- Persisting browser credentials
- Moving approval authority into the operator credential

## Reviewed Delivery Sequence

### P0: Localhost Proof

- Embed a dependency-free dashboard in the Go daemon.
- Add `agentos dashboard` and one-command localhost start/stop scripts.
- Show process state, attempts, usage, budgets, capabilities, events, approvals,
  replay, lifecycle controls, and audit export.
- Keep credentials in URL fragments only during launch, remove them immediately,
  and retain them only in browser session storage.
- Allow same-origin loopback browser requests while rejecting foreign origins.

### P1: Make Correctness Legible

- Add `agentos validate <manifest>` with policy and environment diagnostics.
- Add a preflight screen showing image availability, mount resolution, declared
  egress, missing secrets, and estimated maximum spend before execution.
- Add server-sent events for process and approval updates instead of polling.
- Surface recovery markers, checkpoint age, idempotency keys, and
  `outcome_unknown` reconciliation prominently.
- Add an acceptance command that runs the approval, cancellation, replay, and
  restart-recovery proof suite and emits a signed local report.

### P2: Make Agent Work Reviewable

- Add workspace snapshots and before/after diffs.
- Add artifact review, provenance, and rollback metadata.
- Add policy-pack versioning and signed manifest bundles.
- Publish a narrow adapter SDK for additional model runtimes while preserving
  brokered tool authority.
- Add Podman/containerd support only after the Docker path has real users.

### Deferred

- Remote dashboards, teams, and fleet scheduling
- Global shared memory
- Marketplace and arbitrary agent-to-agent protocols
- GUI authoring of complex workflow graphs

These are deferred because they expand the trust and consistency model before
the single-machine process abstraction has adoption evidence.

## Architecture

```text
Browser tab
  |  embedded HTML/CSS/JS
  |  Bearer operator token in sessionStorage
  |  separate approver token for approval POSTs
  v
Loopback Go HTTP server
  |-- static dashboard assets
  |-- authenticated /v1 API
  |-- same-origin loopback guard
  v
Core service -> SQLite event store -> Docker runner -> agent adapter
```

The dashboard has no direct SQLite, Docker, filesystem, or model access. All
effects continue through the existing API and core policy boundary.

## Failure Modes Registry

| Failure | Expected behavior | Evidence |
|---|---|---|
| Dashboard opened without token | Authentication panel, no process data | HTTP/UI test |
| Foreign web page calls API | Origin rejected before credential use | API test |
| Operator token attempts approval | Unauthorized | Existing API test |
| Wrong approval token | Approval unchanged, token removed from tab | UI behavior + API test |
| Docker unavailable | Dashboard remains usable; process run reports failure | Live localhost smoke |
| Daemon restart | Browser reconnects; SQLite state remains authoritative | Existing recovery suite |
| Browser refresh | Session credential remains in the tab only | Browser smoke |
| Audit download | Redacted JSON only | Existing audit test |
| Stale PID file | Stop script verifies executable before stopping | Script inspection |

## Test Diagram

```text
GET / --------------------------> embedded asset test
dashboard URL #credentials -----> CLI URL test
same-origin browser GET/POST ----> origin/auth API test
foreign-origin API request ------> rejection API test
process list --------------------> live daemon browser smoke
process inspector/events --------> live seeded-state browser smoke
approval decision ---------------> existing two-principal API tests
replay/audit ---------------------> existing replay and redaction tests
start/stop scripts --------------> local health and PID verification
full daemon/core behavior -------> Go + Python adapter suites
```

## Design Review

| Dimension | Score | Improvement |
|---|---:|---|
| Information hierarchy | 8/10 | Keep process list primary and inspector secondary |
| System status | 9/10 | Health, state pills, pending gates, and usage visible |
| Operator safety | 8/10 | Disable invalid lifecycle actions and confirm cancellation |
| Explainability | 8/10 | Show declared authority beside event history |
| Accessibility | 7/10 | Add keyboard table navigation and richer live-region updates |
| Responsive layout | 8/10 | Collapse inspector below process table on narrow screens |
| Visual identity | 8/10 | Functional local-control-plane aesthetic without decorative noise |

## Developer Journey

| Stage | Target |
|---|---|
| Discover | README explains the managed-process distinction in under 30 seconds |
| Install | Extract archive or build once |
| Start | `scripts\start-localhost.cmd` |
| First proof | Dashboard healthy in under 30 seconds |
| First agent | Run supplied offline manifest |
| Human gate | Approve a brokered write in the dashboard |
| Inspect | See budget, capabilities, and ordered events |
| Verify | Replay state and export a redacted audit bundle |
| Recover | Restart daemon and observe recovery without duplicate action |

Target time to visible dashboard: under 1 minute.
Target time to reviewed agent artifact: under 5 minutes with the offline image.

## DX Scorecard

| Dimension | Score |
|---|---:|
| Installation | 8/10 |
| Time to first visible result | 9/10 |
| CLI naming | 9/10 |
| Error actionability | 7/10 |
| Documentation | 8/10 |
| Debuggability | 9/10 |
| Escape hatches | 8/10 |
| Upgrade confidence | 6/10 |

The next DX priority is `agentos validate`, followed by a versioned state
migration and upgrade-check story.

## Acceptance

- Dashboard loads from the daemon with no Node/npm runtime.
- API data is unavailable without the operator token.
- A matching loopback origin is accepted; a foreign origin is rejected.
- Approval POSTs still reject the operator token.
- A user can start and stop the localhost daemon with one command each.
- A process can be inspected, controlled, replayed, and exported from the UI.
- Go tests, vet, adapter tests when Python is available, and a live browser
  smoke all pass.

## Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|
| 1 | CEO | Prioritize visible localhost proof over distributed features | Mechanical | Completeness | Adoption risk now exceeds kernel-feature risk | Cloud control plane |
| 2 | Design | Use a process table plus sticky inspector | Taste | Explicit over clever | Keeps fleet and single-process context visible together | Wizard-only UI |
| 3 | Engineering | Embed static assets in the Go binary | Mechanical | Pragmatic | No frontend toolchain or deployment drift | Separate web service |
| 4 | Security | Pass credentials through a fragment and keep them in session storage | Taste | Explicit over clever | Avoids query/server logs and durable browser storage | Cookies or localStorage |
| 5 | Security | Preserve a separate approval credential in the UI | Mechanical | Completeness | Operator authority must not become self-approval authority | One shared token |
| 6 | DX | Add reversible localhost start/stop scripts | Mechanical | Bias toward action | Makes the proof repeatable in one command | Manual multi-terminal setup |
| 7 | Engineering | Keep polling for P0 and defer SSE to P1 | Taste | Pragmatic | Smaller change while preserving the API boundary | New streaming protocol now |
