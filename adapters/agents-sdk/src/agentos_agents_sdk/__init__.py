"""Agent Process OS adapter for the OpenAI Agents SDK."""

from .protocol import PROTOCOL_VERSION
from .runtime import AdapterRuntime, OpenAIAgentsBackend

__all__ = ["AdapterRuntime", "OpenAIAgentsBackend", "PROTOCOL_VERSION"]

