# Decision: Host workspace mutation remains a trusted-host boundary

- Date: 2026-06-11
- Status: accepted for v1

AgentOS rejects mount roots and broker paths that traverse existing symlinks or
escape `AGENTOS_WORKSPACE_ROOT`. Windows and portable Go do not provide one
cross-platform handle-relative implementation for both Docker bind mounts and
atomic replacement. A concurrent privileged host process could still race a
checked path with a symlink or junction. V1 therefore treats concurrent
host-side workspace mutation as outside the hostile-worker boundary.
