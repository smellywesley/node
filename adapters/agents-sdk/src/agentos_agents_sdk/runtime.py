from __future__ import annotations

import asyncio
import importlib
import inspect
import logging
import os
import sys
import traceback
import uuid
from collections.abc import AsyncIterator, Awaitable, Callable
from typing import Any, Protocol, TextIO

from .protocol import ProtocolError, encode_message, json_safe, message, parse_message

LOGGER = logging.getLogger(__name__)
Emit = Callable[[dict[str, Any]], Awaitable[None]]


class AgentBackend(Protocol):
    async def run(self, task_input: Any, context: Any, emit: Emit) -> Any: ...


def load_object(path: str) -> Any:
    module_name, separator, attribute = path.partition(":")
    if not separator or not module_name or not attribute:
        raise ValueError("factory must use the form 'module.path:attribute'")
    return getattr(importlib.import_module(module_name), attribute)


async def _resolve_factory(factory: Callable[..., Any]) -> Any:
    value = factory()
    if inspect.isawaitable(value):
        return await value
    return value


class OpenAIAgentsBackend:
    def __init__(self, agent_factory: str | None = None) -> None:
        self.agent_factory = agent_factory or os.getenv("AGENTOS_AGENT_FACTORY")
        self._agent: Any = None

    async def _get_agent(self) -> Any:
        if self._agent is not None:
            return self._agent
        if self.agent_factory:
            self._agent = await _resolve_factory(load_object(self.agent_factory))
            _validate_agent(self._agent)
            return self._agent

        try:
            from agents import Agent
        except ImportError as exc:
            raise RuntimeError(
                "openai-agents is not installed; install the adapter package"
            ) from exc

        self._agent = Agent(
            name=os.getenv("AGENTOS_AGENT_NAME", "AgentOS Agent"),
            instructions=os.getenv(
                "AGENTOS_AGENT_INSTRUCTIONS",
                "Complete the task accurately and return a concise final result.",
            ),
            model=os.getenv("OPENAI_MODEL", "gpt-4.1-mini"),
        )
        _validate_agent(self._agent)
        return self._agent

    async def run(self, task_input: Any, context: Any, emit: Emit) -> Any:
        try:
            from agents import Runner
        except ImportError as exc:
            raise RuntimeError(
                "openai-agents is not installed; install the adapter package"
            ) from exc

        agent = await self._get_agent()
        run_input = _coerce_run_input(task_input, context)
        if os.getenv("AGENTOS_DISABLE_STREAMING") == "1":
            completed = await Runner.run(agent, input=run_input)
            await _emit_usage(completed, emit, None)
            return completed.final_output
        streamed = Runner.run_streamed(agent, input=run_input)
        if inspect.isawaitable(streamed):
            streamed = await streamed

        last_usage: tuple[int, int, int] | None = None
        async for event in streamed.stream_events():
            mapped = _map_sdk_event(event)
            if mapped is not None:
                await emit(mapped)
            last_usage = await _emit_usage(streamed, emit, last_usage)
        await _emit_usage(streamed, emit, last_usage)
        return streamed.final_output


async def _emit_usage(
    streamed: Any,
    emit: Emit,
    previous: tuple[int, int, int] | None,
) -> tuple[int, int, int] | None:
    usage = getattr(getattr(streamed, "context_wrapper", None), "usage", None)
    if usage is None:
        return previous
    input_tokens = int(getattr(usage, "input_tokens", 0) or 0)
    output_tokens = int(getattr(usage, "output_tokens", 0) or 0)
    requests = int(getattr(usage, "requests", 0) or 0)
    current = (input_tokens, output_tokens, requests)
    if current != previous:
        await emit(
            message(
                "usage",
                tokens=input_tokens + output_tokens,
                input_tokens=input_tokens,
                output_tokens=output_tokens,
                requests=requests,
                cost_usd=0,
            )
        )
    return current


def _coerce_run_input(task_input: Any, context: Any) -> Any:
    if context is None:
        return task_input
    if isinstance(task_input, str):
        return (
            f"{task_input}\n\n"
            "AgentOS task context (JSON-compatible data):\n"
            f"{json_safe(context)}"
        )
    return {"input": json_safe(task_input), "context": json_safe(context)}


def _validate_agent(agent: Any) -> None:
    if getattr(agent, "mcp_servers", None):
        raise RuntimeError("MCP servers are not supported by the AgentOS v1 adapter")
    if getattr(agent, "handoffs", None):
        raise RuntimeError("agent handoffs are not supported by the AgentOS v1 adapter")
    if getattr(agent, "tools", None):
        raise RuntimeError(
            "direct SDK tools are disabled in AgentOS v1; use the broker protocol"
        )
    expected_model = os.getenv("OPENAI_MODEL")
    actual_model = getattr(agent, "model", None)
    if expected_model:
        if actual_model is None:
            agent.model = expected_model
        elif isinstance(actual_model, str) and actual_model != expected_model:
            raise RuntimeError(
                f"agent model {actual_model!r} does not match manifest model "
                f"{expected_model!r}"
            )
        elif not isinstance(actual_model, str):
            allowed_custom_model = os.getenv("AGENTOS_ALLOW_CUSTOM_MODEL")
            if allowed_custom_model != expected_model:
                raise RuntimeError(
                    "custom model objects require AGENTOS_ALLOW_CUSTOM_MODEL "
                    "to match the manifest model"
                )


def _map_sdk_event(event: Any) -> dict[str, Any] | None:
    event_type = getattr(event, "type", event.__class__.__name__)
    if event_type == "agent_updated_stream_event":
        agent = getattr(event, "new_agent", None)
        return message(
            "agent_event",
            event="agent_updated",
            agent=getattr(agent, "name", None),
        )

    if event_type == "run_item_stream_event":
        item = getattr(event, "item", None)
        item_type = getattr(item, "type", None)
        raw_item = getattr(item, "raw_item", item)
        if item_type == "tool_call_item":
            return message(
                "tool_event",
                event="call",
                tool=_tool_name(raw_item),
                call_id=_attribute(raw_item, "call_id", "id"),
                arguments=json_safe(_attribute(raw_item, "arguments")),
            )
        if item_type in {"tool_call_output_item", "tool_output_item"}:
            return message(
                "tool_event",
                event="result",
                call_id=_attribute(raw_item, "call_id", "id"),
                output=json_safe(getattr(item, "output", raw_item)),
            )
        if item_type == "message_output_item":
            return message("agent_event", event="message_output")
        return message(
            "agent_event",
            event=str(item_type or getattr(event, "name", "run_item")),
        )

    if event_type == "raw_response_event":
        data = getattr(event, "data", None)
        data_type = getattr(data, "type", "")
        if data_type in {"response.output_text.delta", "output_text_delta"}:
            return message(
                "agent_event",
                event="output_text_delta",
                delta=getattr(data, "delta", ""),
            )
    return None


def _attribute(value: Any, *names: str) -> Any:
    for name in names:
        if isinstance(value, dict) and name in value:
            return value[name]
        if hasattr(value, name):
            return getattr(value, name)
    return None


def _tool_name(value: Any) -> str | None:
    name = _attribute(value, "name")
    if name:
        return str(name)
    function = _attribute(value, "function")
    nested_name = _attribute(function, "name")
    return str(nested_name) if nested_name else None


class AdapterRuntime:
    def __init__(
        self,
        backend: AgentBackend,
        reader: TextIO = sys.stdin,
        writer: TextIO = sys.stdout,
        heartbeat_seconds: float = 15.0,
    ) -> None:
        self.backend = backend
        self.reader = reader
        self.writer = writer
        self.heartbeat_seconds = heartbeat_seconds
        self._write_lock = asyncio.Lock()
        self._shutdown = False

    async def emit(self, payload: dict[str, Any]) -> None:
        async with self._write_lock:
            self.writer.write(encode_message(payload) + "\n")
            self.writer.flush()

    async def serve(self) -> None:
        await self.emit(
            message(
                "ready",
                capabilities=["heartbeat", "tool_events", "structured_results"],
            )
        )
        while not self._shutdown:
            line = await asyncio.to_thread(self.reader.readline)
            if line == "":
                break
            if not line.strip():
                continue
            await self.handle_line(line)

    async def handle_line(self, line: str) -> None:
        try:
            inbound = parse_message(line)
        except ProtocolError as exc:
            await self.emit(
                message(
                    "error",
                    error={"code": "invalid_message", "message": str(exc)},
                )
            )
            return

        message_type = inbound["type"]
        if message_type == "ping":
            await self.emit(message("heartbeat", reply_to=inbound.get("id")))
        elif message_type == "shutdown":
            self._shutdown = True
            await self.emit(message("shutdown_ack"))
        elif message_type == "task":
            await self._handle_task(inbound)
        else:
            await self.emit(
                message(
                    "error",
                    reply_to=inbound.get("id"),
                    error={
                        "code": "unsupported_message",
                        "message": f"unsupported message type: {message_type}",
                    },
                )
            )

    async def _handle_task(self, inbound: dict[str, Any]) -> None:
        task_id = str(inbound.get("id") or uuid.uuid4())
        if "input" not in inbound:
            await self.emit(
                message(
                    "error",
                    task_id=task_id,
                    error={
                        "code": "invalid_task",
                        "message": "task.input is required",
                    },
                )
            )
            return

        await self.emit(message("task_started", task_id=task_id))
        stop_heartbeat = asyncio.Event()
        heartbeat = asyncio.create_task(
            self._heartbeat_loop(task_id, stop_heartbeat)
        )

        async def task_emit(payload: dict[str, Any]) -> None:
            payload.setdefault("task_id", task_id)
            await self.emit(payload)

        try:
            output = await self.backend.run(
                inbound["input"], inbound.get("context"), task_emit
            )
            artifact = os.getenv("AGENTOS_RESULT_ARTIFACT")
            if artifact:
                artifact_content = str(output)
                if len(artifact_content.encode("utf-8")) > 8 * 1024 * 1024:
                    raise RuntimeError("artifact content exceeds the 8 MiB broker limit")
                key = f"result-artifact:{task_id}:{artifact}"
                await self.emit(
                    message(
                        "tool_request",
                        task_id=task_id,
                        idempotency_key=key,
                        action="fs.write",
                        resource=artifact,
                        payload={"content": artifact_content},
                    )
                )
                response_line = await asyncio.to_thread(self.reader.readline)
                if response_line == "":
                    raise RuntimeError("tool broker closed before artifact completion")
                response = parse_message(response_line)
                if (
                    response.get("type") != "tool_result"
                    or response.get("idempotency_key") != key
                    or response.get("status") != "completed"
                ):
                    raise RuntimeError(
                        f"artifact tool failed: {response.get('error') or response}"
                    )
                output = {
                    "artifact": artifact,
                    "model_output": json_safe(output),
                    "tool_output": json_safe(response.get("output")),
                }
            await self.emit(
                message("result", task_id=task_id, output=json_safe(output))
            )
        except Exception as exc:
            if os.getenv("AGENTOS_DEBUG_ERRORS") == "1":
                LOGGER.exception("task %s failed", task_id)
            else:
                LOGGER.debug("task %s failed: %s", task_id, exc)
            error: dict[str, Any] = {
                "code": "agent_run_failed",
                "message": str(exc) or exc.__class__.__name__,
                "kind": exc.__class__.__name__,
                "retryable": False,
            }
            if os.getenv("AGENTOS_DEBUG_ERRORS") == "1":
                error["traceback"] = traceback.format_exc()
            await self.emit(message("error", task_id=task_id, error=error))
        finally:
            stop_heartbeat.set()
            await heartbeat

    async def _heartbeat_loop(
        self, task_id: str, stop: asyncio.Event
    ) -> None:
        if self.heartbeat_seconds <= 0:
            return
        while True:
            try:
                await asyncio.wait_for(
                    stop.wait(), timeout=self.heartbeat_seconds
                )
                return
            except TimeoutError:
                await self.emit(message("heartbeat", task_id=task_id))
