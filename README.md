# systemd-mcp

> **Mirror** -- Canonical development lives in [hairglasses-studio/dotfiles](https://github.com/hairglasses-studio/dotfiles) at `mcp/systemd-mcp/`. This repo is a publish mirror kept in parity for `go install` and MCP registry discovery.

[![Go Reference](https://pkg.go.dev/badge/github.com/hairglasses-studio/systemd-mcp.svg)](https://pkg.go.dev/github.com/hairglasses-studio/systemd-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/hairglasses-studio/systemd-mcp)](https://goreportcard.com/report/github.com/hairglasses-studio/systemd-mcp)
[![CI](https://github.com/hairglasses-studio/systemd-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/hairglasses-studio/systemd-mcp/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Glama](https://glama.ai/mcp/servers/hairglasses-studio/systemd-mcp/badges/score.svg)](https://glama.ai/mcp/servers/hairglasses-studio/systemd-mcp)

MCP server for systemd service and timer management. Gives AI assistants like Codex or Claude Code the ability to manage Linux services, inspect logs, and debug failed units.

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

Add to your MCP client config (for example Codex or Claude Code):

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

## Runtime Requirements

- Linux with `systemd`
- `systemctl` for unit inspection and write operations
- `journalctl` for log reads
- A reachable user systemd manager for default user-scope operations
- Appropriate permissions for system-scope operations

`systemd-mcp` prefers D-Bus, but fallback is backend-aware rather than guaranteed success. A host can have a session bus without a usable user manager, or `systemctl` installed without `systemctl --user` being usable in the current environment. The runtime capability report is available at `systemd://runtime/capabilities`.

## Test Tiers

Default `go test` runs deterministic unit coverage plus live-integration tests that skip unless `SYSTEMD_MCP_LIVE=1` and the required host capability is actually present.

```bash
go test ./... -count=1
SYSTEMD_MCP_LIVE=1 go test ./... -count=1
make test-unit
make test-live
```

The canonical dotfiles CI runs user-scope live coverage in a dedicated `systemd-live` job and keeps the system-scope lane as an opt-in `workflow_dispatch` path because host permissions vary.

## Architecture

Single Go binary. Uses D-Bus as the primary backend, explicit runtime capability probes for user and system scope, and structured fallback to `systemctl` and `journalctl` only when those backends are actually usable. Uses mcpkit's `TypedHandler` generics for type-safe parameter handling and structured error codes.

## License

MIT
