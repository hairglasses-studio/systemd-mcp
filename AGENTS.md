# systemd-mcp — Agent Instructions

MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

> Canonical source: CLAUDE.md

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
```

## Architecture

Single-file monolithic Go program (`main.go`). One `SystemdModule` registers all 10 tools. Shells out to `systemctl` and `journalctl` — no D-Bus dependency.

