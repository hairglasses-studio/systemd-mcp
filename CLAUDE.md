# systemd-mcp

This repo uses [AGENTS.md](AGENTS.md) as the canonical instruction file.

MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

## Tools (10)

| Tool | Description |
|------|-------------|
| `systemd_status` | Show unit status (active state, PID, memory, CPU) |
| `systemd_start` | Start a unit |
| `systemd_stop` | Stop a unit |
| `systemd_restart` | Restart a unit |
| `systemd_enable` | Enable a unit for boot/login |
| `systemd_disable` | Disable a unit from boot/login |
| `systemd_logs` | Fetch journal logs (configurable lines, since filter) |
| `systemd_list_units` | List units with optional state filter |
| `systemd_list_timers` | List active timers with trigger times |
| `systemd_failed` | List failed units |

All tools default to **user scope** (`--user`). Set `system: true` for system scope.

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
SYSTEMD_MCP_LIVE=1 go test ./... -count=1
```

## Architecture

Go program with D-Bus primary backend, explicit runtime capability probes for user and system scope, and structured fallback to systemctl/journalctl only when those backends are actually usable.

## Scope Convention

The `system` bool field (default `false`) controls scope:
- `false` (default) = `--user` scope (user services like makima, shader-rotate)
- `true` = system scope (requires appropriate permissions)
