# AgentOS Benchmark Report

Date: 2026-06-23
Target: http://127.0.0.1:7479/
Mode: local dashboard plus operational smoke budgets

## Result

PASS. The localhost dashboard is comfortably inside the demo budgets, and the operational CLI checks are below the target thresholds.

## Dashboard Measurements

| Metric | Budget | Actual | Status |
|---|---:|---:|---|
| gstack browse total load | < 2500 ms | 10 ms | PASS |
| Total JS | < 500 KB | 19,103 bytes | PASS |
| Total CSS | < 100 KB | 12,909 bytes | PASS |
| Total transfer | < 2 MB | 40,199 bytes | PASS |
| Static requests | < 50 | 3 | PASS |
| Health request | < 500 ms | 21.86 ms | PASS |

## Operational Measurements

| Operation | Budget | Actual | Status |
|---|---:|---:|---|
| `agentos validate examples\smoke\agent-process.yaml` | < 2000 ms | 106.41 ms | PASS |
| `agentos ps` | < 500 ms | 27.47 ms | PASS |
| `agentos doctor` | < 2000 ms | 139.45 ms | PASS |

## Demo Readiness Evidence

- Dashboard renders a guided demo path instead of a blank empty state.
- Smoke worker image was built locally.
- Smoke process `01fb054b-02c9-4505-90bd-a817c9804b43` reached `succeeded`.
- Replay for that process returned `succeeded`.
- Operator token was rotated after prior exposure, and the new token is not printed in this report.
- Security audit passed with zero forbidden tracked paths and zero high-confidence secret findings.
- Release archive has 119 entries and zero forbidden entries.

## Notes

The gstack browser `eval` command treated inline JavaScript as a file path in this environment, so detailed Performance API resource extraction was replaced with deterministic HTTP resource-size measurements plus `browse perf` timing. This keeps the benchmark repeatable and avoids storing credentials.