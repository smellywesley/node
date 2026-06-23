import json
import socket
import sys
import urllib.error
import urllib.request


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
for line in sys.stdin:
    request = json.loads(line)
    task_id = request.get("id")
    allowed_proxy = False
    blocked_host = False
    direct_blocked = False
    try:
        with urllib.request.urlopen("https://example.com", timeout=10) as response:
            allowed_proxy = response.status == 200
    except Exception:
        allowed_proxy = False
    try:
        urllib.request.urlopen("https://openai.com", timeout=5)
    except urllib.error.HTTPError as error:
        blocked_host = error.code == 403
    except Exception:
        blocked_host = True
    try:
        connection = socket.create_connection(("1.1.1.1", 443), timeout=3)
        connection.close()
    except OSError:
        direct_blocked = True
    if not (allowed_proxy and blocked_host and direct_blocked):
        emit(
            "error",
            task_id=task_id,
            error={
                "allowed_proxy": allowed_proxy,
                "blocked_host": blocked_host,
                "direct_blocked": direct_blocked,
            },
        )
    else:
        emit(
            "result",
            task_id=task_id,
            output={
                "allowed_proxy": allowed_proxy,
                "blocked_host": blocked_host,
                "direct_blocked": direct_blocked,
            },
        )
