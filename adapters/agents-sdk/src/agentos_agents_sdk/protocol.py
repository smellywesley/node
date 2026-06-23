from __future__ import annotations

import dataclasses
import datetime as dt
import json
from typing import Any, Mapping

PROTOCOL_VERSION = "agent-process-os/v1"


class ProtocolError(ValueError):
    """Raised when an inbound protocol message is invalid."""


def utc_timestamp() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def parse_message(line: str) -> dict[str, Any]:
    try:
        message = json.loads(line)
    except json.JSONDecodeError as exc:
        raise ProtocolError(f"invalid JSON: {exc.msg}") from exc
    if not isinstance(message, dict):
        raise ProtocolError("message must be a JSON object")
    message_type = message.get("type")
    if not isinstance(message_type, str) or not message_type:
        raise ProtocolError("message.type must be a non-empty string")
    return message


def message(message_type: str, **fields: Any) -> dict[str, Any]:
    return {
        "protocol": PROTOCOL_VERSION,
        "type": message_type,
        "timestamp": utc_timestamp(),
        **fields,
    }


def json_safe(value: Any) -> Any:
    if value is None or isinstance(value, (bool, int, float, str)):
        return value
    if dataclasses.is_dataclass(value) and not isinstance(value, type):
        return json_safe(dataclasses.asdict(value))
    if isinstance(value, Mapping):
        return {str(key): json_safe(item) for key, item in value.items()}
    if isinstance(value, (list, tuple, set)):
        return [json_safe(item) for item in value]
    model_dump = getattr(value, "model_dump", None)
    if callable(model_dump):
        return json_safe(model_dump(mode="json"))
    to_dict = getattr(value, "to_dict", None)
    if callable(to_dict):
        return json_safe(to_dict())
    return str(value)


def encode_message(payload: Mapping[str, Any]) -> str:
    return json.dumps(json_safe(payload), separators=(",", ":"), ensure_ascii=True)

