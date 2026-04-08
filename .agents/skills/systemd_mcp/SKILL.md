---
name: systemd_mcp
description: 'Operate systemd-mcp as the canonical repo for systemd service and timer MCP tooling. Use this when changing D-Bus or systemctl-backed behavior, repo guidance, or contract surfaces here, and reference this repo from broader parity docs instead of duplicating it elsewhere.'
---

# systemd-mcp

Use this repo as the canonical home for MCP tooling around systemd units, timers, journals, and service control.

Focus paths:
- `cmd/`
- `internal/`
- `AGENTS.md`
- `README.md`

Preserve the fallback behavior between D-Bus and shell-backed execution, keep write paths clearly separated from inspection tools, and prefer cross-repo references back to this repo over copied maintenance guidance.
