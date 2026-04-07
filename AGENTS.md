# systemd-mcp — Agent Instructions
> Canonical instructions: AGENTS.md


MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

> Canonical source: CLAUDE.md

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
```

## Architecture

Go program with D-Bus primary backend and systemctl/journalctl fallback. One `SystemdModule` registers all 10 tools. D-Bus connections are optional — if unavailable, tools transparently fall back to shell commands.
