import json
import sys


def emit(message_type, **fields):
    print(
        json.dumps(
            {
                "protocol": "agent-process-os/v1",
                "type": message_type,
                **fields,
            },
            separators=(",", ":"),
        ),
        flush=True,
    )


emit("ready")
task = json.loads(sys.stdin.readline())
task_id = task["id"]
emit("task_started", task_id=task_id)
emit(
    "tool_request",
    task_id=task_id,
    idempotency_key="write-reviewed-artifact",
    action="fs.write",
    resource="/workspace/reviewed.txt",
    payload={"content": "approved by Agent Process OS\n"},
)
result = json.loads(sys.stdin.readline())
if result.get("type") != "tool_result" or result.get("status") != "completed":
    emit("error", task_id=task_id, error=result)
else:
    emit("result", task_id=task_id, output=result.get("output"))
