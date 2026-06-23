import json
import os
import sys
import time


def emit(message_type, **fields):
    print(
        json.dumps(
            {
                "protocol": "agent-process-os/v1",
                "type": message_type,
                "timestamp": time.time(),
                **fields,
            },
            separators=(",", ":"),
        ),
        flush=True,
    )


emit("ready", capabilities=["structured_results"])
for line in sys.stdin:
    request = json.loads(line)
    if request.get("type") != "task":
        emit("error", error={"code": "unsupported_message"})
        continue
    text = str(request.get("input", ""))
    context = request.get("context") or {}
    emit("task_started", task_id=request.get("id"))
    delay = float(os.getenv("AGENTOS_TEST_DELAY_SECONDS", "0"))
    if delay and not context.get("checkpoint"):
        emit("checkpoint", task_id=request.get("id"), checkpoint={"phase": "ready"})
        time.sleep(delay)
    emit(
        "result",
        task_id=request.get("id"),
        output={
            "echo": text,
            "word_count": len(text.split()),
            "resumed": bool(context.get("checkpoint")),
        },
    )
