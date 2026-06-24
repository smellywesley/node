import json
import sys


PROTOCOL = "agent-process-os/v1"


def emit(message_type, **fields):
    print(
        json.dumps(
            {
                "protocol": PROTOCOL,
                "type": message_type,
                **fields,
            },
            separators=(",", ":"),
        ),
        flush=True,
    )


def read_message(task_id):
    line = sys.stdin.readline()
    if not line:
        emit("error", task_id=task_id, error="daemon closed the worker channel")
        sys.exit(1)
    return json.loads(line)


def request_tool(task_id, key, action, resource, payload):
    emit(
        "tool_request",
        task_id=task_id,
        idempotency_key=key,
        action=action,
        resource=resource,
        payload=payload,
    )
    result = read_message(task_id)
    if result.get("type") != "tool_result" or result.get("idempotency_key") != key:
        emit("error", task_id=task_id, error={"unexpected_tool_result": result})
        sys.exit(1)
    return result


emit("ready")
task = json.loads(sys.stdin.readline())
task_id = task["id"]
task_text = task.get("input", "")

emit("task_started", task_id=task_id)

# This frame exercises AgentOS budget accounting. Cost is calculated by the
# daemon from the manifest pricing so dashboard and audit numbers use one path.
emit(
    "usage",
    task_id=task_id,
    tokens=1800,
    input_tokens=1200,
    output_tokens=600,
    requests=1,
    provider="demo-metered",
)

allowed = request_tool(
    task_id,
    "write-backend-fix",
    "fs.write",
    "/workspace/internal/backend_fix.txt",
    {
        "content": (
            "AgentOS pay-ready demo artifact\n\n"
            "Task: "
            + task_text.strip()
            + "\n\n"
            "This backend-scoped write was approved and brokered by AgentOS.\n"
        )
    },
)
if allowed.get("status") != "completed":
    emit("error", task_id=task_id, error={"allowed_write_failed": allowed})
    sys.exit(1)

forbidden = request_tool(
    task_id,
    "write-forbidden-frontend",
    "fs.write",
    "/workspace/web/app.js",
    {
        "content": (
            "console.log('This frontend write should never be created by the demo.');\n"
        )
    },
)
if forbidden.get("status") != "failed":
    emit("error", task_id=task_id, error={"forbidden_write_was_not_denied": forbidden})
    sys.exit(1)

emit(
    "result",
    task_id=task_id,
    output={
        "allowed_write": allowed.get("status"),
        "forbidden_write": "denied",
        "tokens": 1800,
        "cost_usd": 0.003,
        "message": "AgentOS approved the backend write, denied the frontend write, and accounted for usage.",
    },
)