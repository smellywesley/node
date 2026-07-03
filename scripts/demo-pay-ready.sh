#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADDRESS="${ADDRESS:-127.0.0.1:17479}"
KEEP_RUNNING="${KEEP_RUNNING:-0}"
AGENTOS="$ROOT/bin/agentos"
STATE_HOME="$ROOT/work/pay-ready-home"
WORKSPACE="$ROOT/work/pay-ready-workspace"
INTERNAL_DIR="$WORKSPACE/internal"
WEB_DIR="$WORKSPACE/web"
OUTPUTS="$ROOT/outputs"
AUDIT_PATH="$OUTPUTS/pay-ready-audit.json"
DAEMON_OUT="$STATE_HOME/daemon.out"
DAEMON_ERR="$STATE_HOME/daemon.err"
DAEMON_PID=""

step() {
  echo "[pay-ready] $*"
}

die() {
  echo "problem: $*" >&2
  exit 1
}

assert_under_root() {
  local child
  child="$(python3 - "$ROOT" "$1" <<'PY'
import os, sys
root = os.path.realpath(sys.argv[1])
child = os.path.realpath(sys.argv[2])
if not child.startswith(root + os.sep):
    raise SystemExit(1)
print(child)
PY
)" || die "refusing to touch path outside repo workspace: $1"
  printf '%s\n' "$child" >/dev/null
}

json_get() {
  python3 - "$1" <<'PY'
import json, sys
data = json.load(sys.stdin)
for part in sys.argv[1].split("."):
    data = data[part]
print(data)
PY
}

json_pending_approval() {
  python3 - "$1" <<'PY'
import json, sys
process_id = sys.argv[1]
data = json.load(sys.stdin)
if isinstance(data, dict):
    data = [data]
for item in data:
    if item.get("process_id") == process_id and item.get("status") == "pending":
        print(item.get("id", ""))
        break
PY
}

agentos() {
  "$AGENTOS" "$@"
}

cleanup() {
  if [[ "$KEEP_RUNNING" != "1" && -n "$DAEMON_PID" ]] && kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill "$DAEMON_PID" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" >/dev/null 2>&1 || true
  elif [[ "$KEEP_RUNNING" == "1" ]]; then
    step "daemon left running on $ADDRESS"
  fi
}
trap cleanup EXIT

command -v python3 >/dev/null 2>&1 || die "python3 is required for JSON checks."
command -v docker >/dev/null 2>&1 || {
  echo "problem: Docker is not installed or not on PATH."
  echo "cause: AgentOS runs agent workers in OCI-compatible containers."
  echo "fix: Install Docker Desktop or another Docker-compatible engine, start it, then rerun this demo."
  exit 2
}
docker version --format '{{.Server.Version}}' >/dev/null 2>&1 || {
  echo "problem: Docker is installed but the engine is not reachable."
  echo "cause: Docker Desktop or the container engine is stopped or still starting."
  echo "fix: Start Docker, wait until it is running, then rerun ./bin/agentos doctor --support."
  exit 2
}

step "building AgentOS binary"
OUTPUT=bin/agentos "$ROOT/scripts/build.sh"

step "running security audit"
"$ROOT/scripts/security-audit.sh"

step "building pay-ready worker image"
docker build -f "$ROOT/examples/pay-ready/Dockerfile" -t agentos/pay-ready-demo:local "$ROOT"

mkdir -p "$STATE_HOME" "$INTERNAL_DIR" "$WEB_DIR" "$OUTPUTS"
for path in "$INTERNAL_DIR/backend_fix.txt" "$WEB_DIR/app.js" "$AUDIT_PATH"; do
  assert_under_root "$path"
  rm -f "$path"
done

export AGENTOS_HOME="$STATE_HOME"
export AGENTOS_ADDR="$ADDRESS"
if [[ -z "${AGENTOS_APPROVER_TOKEN:-}" ]]; then
  export AGENTOS_APPROVER_TOKEN
  AGENTOS_APPROVER_TOKEN="$(python3 - <<'PY'
import secrets
print(secrets.token_hex(64))
PY
)"
fi

step "validating manifest"
agentos validate "$ROOT/examples/pay-ready/agent-process.yaml"

step "starting isolated daemon on $ADDRESS"
agentos serve --addr "$ADDRESS" >"$DAEMON_OUT" 2>"$DAEMON_ERR" &
DAEMON_PID="$!"
sleep 2

step "creating managed process"
run_output="$(agentos run "$ROOT/examples/pay-ready/agent-process.yaml")"
process_id="$(printf '%s' "$run_output" | json_get id)"
[[ -n "$process_id" ]] || die "run response did not include a process id: $run_output"
step "process id $process_id"

approval_id=""
for _ in $(seq 1 60); do
  sleep 0.5
  approvals="$(agentos approvals)"
  approval_id="$(printf '%s' "$approvals" | json_pending_approval "$process_id")"
  [[ -n "$approval_id" ]] && break
done
[[ -n "$approval_id" ]] || die "no approval was created for the backend write."

step "approving backend-only write"
agentos approve "$approval_id" "pay-ready demo backend-only approval" >/dev/null

state=""
inspection=""
for _ in $(seq 1 80); do
  sleep 0.5
  inspection="$(agentos inspect "$process_id")"
  state="$(printf '%s' "$inspection" | json_get state)"
  [[ "$state" == "succeeded" || "$state" == "failed" || "$state" == "cancelled" ]] && break
done
[[ "$state" == "succeeded" ]] || die "process finished in unexpected state '$state'."

step "exporting replay and audit evidence"
replay="$(agentos replay "$process_id")"
agentos audit "$process_id" "$AUDIT_PATH" >/dev/null
logs="$(agentos logs "$process_id")"

python3 - "$inspection" "$replay" "$logs" "$INTERNAL_DIR/backend_fix.txt" "$WEB_DIR/app.js" <<'PY'
import json, os, sys
inspection = json.loads(sys.argv[1])
replay = json.loads(sys.argv[2])
logs = json.loads(sys.argv[3])
allowed_file = sys.argv[4]
forbidden_file = sys.argv[5]
if replay.get("state") != "succeeded":
    raise SystemExit("Replay did not reconstruct succeeded state.")
usage = inspection.get("usage") or {}
if usage.get("tokens", 0) <= 0 or usage.get("cost_usd", 0) <= 0:
    raise SystemExit("Usage accounting did not produce nonzero tokens and cost.")
if not any(item.get("type") == "budget.usage_updated" for item in logs):
    raise SystemExit("Missing budget.usage_updated event.")
if not any(item.get("type") == "tool.denied" for item in logs):
    raise SystemExit("Missing tool.denied event.")
if not os.path.exists(allowed_file):
    raise SystemExit("Approved backend artifact was not created.")
if os.path.exists(forbidden_file):
    raise SystemExit("Forbidden frontend artifact was created.")
PY

tokens="$(printf '%s' "$inspection" | json_get usage.tokens)"
cost="$(printf '%s' "$inspection" | json_get usage.cost_usd)"
denied_count="$(printf '%s' "$logs" | python3 -c 'import json,sys; print(sum(1 for item in json.load(sys.stdin) if item.get("type") == "tool.denied"))')"

echo ""
echo "Pay-ready demo passed"
echo "Process: $process_id"
echo "State: $state"
echo "Usage: $tokens tokens, \$$cost"
echo "Denied actions: $denied_count"
echo "Approved artifact: $INTERNAL_DIR/backend_fix.txt"
echo "Audit bundle: $AUDIT_PATH"
