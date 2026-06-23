# OpenAI Agents SDK adapter example

Build the example from the repository root:

```sh
docker build -f examples/agents-sdk/Dockerfile \
  -t agentos/openai-agents-word-count:local .
```

Start the daemon, then run the manifest:

```sh
export OPENAI_API_KEY=...
agentos serve
agentos run examples/agents-sdk/agent-process.yaml
```

`OPENAI_API_KEY` is injected only because the manifest declares it. The
container receives no network at all unless `network_destinations` is
non-empty. AgentOS then creates an isolated per-process network and routes HTTPS
through an exact-host allowlist proxy.

For a direct protocol smoke test:

```sh
docker run --rm -i -e OPENAI_API_KEY \
  agentos/openai-agents-word-count:local <<'EOF'
{"type":"task","id":"demo-1","input":"How many words are in: Agent processes are portable."}
{"type":"shutdown"}
EOF
```

The process writes one JSON object per stdout line. A typical run emits
`ready`, `task_started`, tool `call` and `result` events, a final `result`,
and `shutdown_ack`. Container logs are written to stderr.

You can use a different SDK agent without changing the adapter. Put an
importable callable in the image and set:

```yaml
implementation:
  env:
    AGENTOS_AGENT_FACTORY: your_package.agent:create_agent
```

The callable may return an `Agent` directly or return it from an async
function.

V1 disables direct SDK tools, MCP servers, and handoffs. Consequential effects
must be requested through the AgentOS broker so policy, approval, idempotency,
and audit rules remain authoritative.
