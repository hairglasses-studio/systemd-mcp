# systemd-mcp

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

MCP server for systemd service and timer management. Gives AI assistants like Claude Code the ability to manage Linux services, inspect logs, and debug failed units.

Built with [mcpkit](https://github.com/hairglasses-studio/mcpkit) using stdio transport.

## Install

```bash
go install github.com/hairglasses-studio/systemd-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/hairglasses-studio/systemd-mcp
cd systemd-mcp
go build -o systemd-mcp .
```

## Configure

Add to your Claude Code MCP config (`.mcp.json`):

```json
{
  "mcpServers": {
    "systemd": {
      "command": "systemd-mcp"
    }
  }
}
```

## Tools

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

## Scope

All tools default to **user scope** (`--user`). Set `system: true` for system-wide services (requires appropriate permissions).

```
"Can you check the status of the docker service?"
→ systemd_status(unit: "docker", system: true)

"Show me any failed user services"
→ systemd_failed()

"Get the last 50 lines of logs for my shader-rotate timer"
→ systemd_logs(unit: "shader-rotate", lines: 50)
```

## Architecture

Single Go binary. Shells out to `systemctl` and `journalctl` — no D-Bus dependency. Uses mcpkit's `TypedHandler` generics for type-safe parameter handling and structured error codes.

## License

MIT
