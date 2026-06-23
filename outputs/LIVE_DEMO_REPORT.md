# AgentOS Live Demo Report

Date: 2026-06-23
Host: local Windows workstation
Daemon address: 127.0.0.1:7479

## What Ran

A local AgentOS daemon was started with `scripts/start-localhost.cmd` after fixing a Windows process-local `Path`/`PATH` duplication issue in the launcher.

Docker Desktop was started, the smoke worker image was built, and `examples/smoke/agent-process.yaml` was submitted through the daemon.

## Evidence

- Docker image: `agentos/protocol-smoke:local`
- Manifest validation: passed
- Process ID: `4518d4c2-d8f1-461a-8ac9-04c2d1e10e8a`
- Final state: `succeeded`
- Replay state: `succeeded`
- Event sequence observed:
  - `process.created`
  - `process.queued`
  - `process.running`
  - `worker.stdout` ready frame
  - `worker.stdout` task started frame
  - `worker.stdout` result frame
  - `process.succeeded`

## Dashboard

The localhost dashboard opened successfully earlier and showed the daemon online, process counters, process table, approval panel, and inspector. During post-run screenshot capture, the gstack browser helper crashed repeatedly, so dashboard screenshot evidence was not captured in this pass. CLI/API evidence confirmed the live process lifecycle and deterministic replay.

## Notes

No credential-bearing dashboard URLs or operator tokens are stored in this report.
