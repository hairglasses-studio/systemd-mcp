# systemd-mcp — Agent Instructions
> Canonical instructions: AGENTS.md


MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

> Canonical source: CLAUDE.md

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
SYSTEMD_MCP_LIVE=1 go test ./... -count=1
```

## Architecture

Go program with D-Bus primary backend, explicit per-scope capability probing, and systemctl/journalctl fallback only when the alternate backend is usable. Default `go test` is the deterministic tier; live integration coverage is opt-in via `SYSTEMD_MCP_LIVE=1` and still skips when the host cannot satisfy the requested scope.
