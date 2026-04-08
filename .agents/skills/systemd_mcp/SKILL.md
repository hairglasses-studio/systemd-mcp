---
name: systemd_mcp
description: 'Operate the systemd-mcp server for service and timer management. Use this when changing D-Bus or systemctl-backed MCP tooling in this repo, not for generic workstation work that belongs in dotfiles.'
---

# systemd-mcp

Use this repo for MCP tooling around systemd units, timers, journals, and service control.

Focus paths:
- `cmd/`
- `internal/`
- `AGENTS.md`
- `README.md`

Preserve the fallback behavior between D-Bus and shell-backed execution, and keep write paths clearly separated from inspection tools.
