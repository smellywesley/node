# Uninstall And Reset

AgentOS v1 is local-first. Removing it means stopping daemons, deleting the extracted package or source checkout, and deleting any state directories you intentionally created.

## Stop Localhost Demo

```powershell
.\scripts\stop-localhost.cmd
```

## Default State

By default, daemon state lives under:

```text
%USERPROFILE%\.agentos
```

If you set `AGENTOS_HOME`, delete that dedicated state directory instead. Never point `AGENTOS_HOME` at the user profile root or a filesystem root.

## Demo State

The localhost script uses:

```text
work\localhost\state
```

The state directory contains the operator token and SQLite process database. Delete it only when you no longer need process history, replay, or audit export.

## Docker Artifacts

AgentOS workers use Docker images and temporary containers. Containers are removed automatically when runs complete. You may remove local example images manually with Docker if desired.

## Browser Credentials

The dashboard stores credentials in browser `sessionStorage` for the launched tab. Close the tab to discard them.
