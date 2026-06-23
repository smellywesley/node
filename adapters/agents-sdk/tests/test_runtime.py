import asyncio
import io
import json
import os
import unittest
from types import SimpleNamespace
from unittest.mock import patch

from agentos_agents_sdk.protocol import message
from agentos_agents_sdk.runtime import (
    AdapterRuntime,
    _map_sdk_event,
    _validate_agent,
)


class SuccessfulBackend:
    async def run(self, task_input, context, emit):
        await emit(
            message(
                "tool_event",
                event="call",
                tool="word_count",
                call_id="call-1",
                arguments={"text": task_input},
            )
        )
        await asyncio.sleep(0.02)
        await emit(
            message(
                "tool_event",
                event="result",
                call_id="call-1",
                output=4,
            )
        )
        return {"answer": 4, "context": context}


class FailingBackend:
    async def run(self, task_input, context, emit):
        raise RuntimeError("offline test failure")


def decoded_lines(writer):
    return [json.loads(line) for line in writer.getvalue().splitlines()]


class RuntimeTests(unittest.IsolatedAsyncioTestCase):
    async def test_success_emits_lifecycle_heartbeat_and_tool_events(self):
        reader = io.StringIO(
            '{"type":"task","id":"task-1","input":"one two three four",'
            '"context":{"source":"test"}}\n'
            '{"type":"shutdown"}\n'
        )
        writer = io.StringIO()
        runtime = AdapterRuntime(
            SuccessfulBackend(),
            reader=reader,
            writer=writer,
            heartbeat_seconds=0.005,
        )

        await runtime.serve()

        payloads = decoded_lines(writer)
        types = [payload["type"] for payload in payloads]
        self.assertEqual(types[0], "ready")
        self.assertIn("task_started", types)
        self.assertIn("heartbeat", types)
        self.assertEqual(types.count("tool_event"), 2)
        result = next(item for item in payloads if item["type"] == "result")
        self.assertEqual(result["task_id"], "task-1")
        self.assertEqual(result["output"]["answer"], 4)
        self.assertEqual(types[-1], "shutdown_ack")

    async def test_backend_failure_becomes_structured_error(self):
        writer = io.StringIO()
        runtime = AdapterRuntime(
            FailingBackend(),
            reader=io.StringIO('{"type":"task","id":"bad","input":"x"}\n'),
            writer=writer,
            heartbeat_seconds=0,
        )

        await runtime.serve()

        error = next(
            item for item in decoded_lines(writer) if item["type"] == "error"
        )
        self.assertEqual(error["task_id"], "bad")
        self.assertEqual(error["error"]["code"], "agent_run_failed")
        self.assertEqual(error["error"]["kind"], "RuntimeError")

    async def test_invalid_and_ping_messages_are_handled(self):
        writer = io.StringIO()
        runtime = AdapterRuntime(
            SuccessfulBackend(),
            reader=io.StringIO('not-json\n{"type":"ping","id":"p-1"}\n'),
            writer=writer,
            heartbeat_seconds=0,
        )

        await runtime.serve()

        payloads = decoded_lines(writer)
        self.assertEqual(payloads[1]["error"]["code"], "invalid_message")
        self.assertEqual(payloads[2]["type"], "heartbeat")
        self.assertEqual(payloads[2]["reply_to"], "p-1")

    def test_sdk_tool_call_event_mapping(self):
        event = SimpleNamespace(
            type="run_item_stream_event",
            item=SimpleNamespace(
                type="tool_call_item",
                raw_item=SimpleNamespace(
                    name="lookup", call_id="c-7", arguments='{"q":"x"}'
                ),
            ),
        )

        mapped = _map_sdk_event(event)

        self.assertEqual(mapped["type"], "tool_event")
        self.assertEqual(mapped["event"], "call")
        self.assertEqual(mapped["tool"], "lookup")
        self.assertEqual(mapped["call_id"], "c-7")

    def test_direct_sdk_tools_are_default_denied(self):
        tool = SimpleNamespace(name="shell")
        agent = SimpleNamespace(tools=[tool], model=None)
        with self.assertRaisesRegex(RuntimeError, "direct SDK tools are disabled"):
            _validate_agent(agent)

    def test_mcp_handoffs_are_denied_and_manifest_model_is_applied(self):
        with self.assertRaisesRegex(RuntimeError, "MCP servers"):
            _validate_agent(SimpleNamespace(mcp_servers=[object()], handoffs=[], tools=[]))
        with self.assertRaisesRegex(RuntimeError, "handoffs"):
            _validate_agent(SimpleNamespace(mcp_servers=[], handoffs=[object()], tools=[]))

        agent = SimpleNamespace(mcp_servers=[], handoffs=[], tools=[], model=None)
        with patch.dict(os.environ, {"OPENAI_MODEL": "manifest-model"}, clear=False):
            _validate_agent(agent)
        self.assertEqual(agent.model, "manifest-model")

    def test_custom_model_requires_manifest_matching_opt_in(self):
        custom_model = object()
        agent = SimpleNamespace(
            mcp_servers=[], handoffs=[], tools=[], model=custom_model
        )
        with patch.dict(
            os.environ,
            {"OPENAI_MODEL": "offline-deterministic"},
            clear=False,
        ):
            os.environ.pop("AGENTOS_ALLOW_CUSTOM_MODEL", None)
            with self.assertRaisesRegex(RuntimeError, "custom model objects"):
                _validate_agent(agent)

        with patch.dict(
            os.environ,
            {
                "OPENAI_MODEL": "offline-deterministic",
                "AGENTOS_ALLOW_CUSTOM_MODEL": "offline-deterministic",
            },
            clear=False,
        ):
            _validate_agent(agent)


if __name__ == "__main__":
    unittest.main()
