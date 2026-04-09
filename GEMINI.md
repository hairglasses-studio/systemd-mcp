# systemd-mcp — Gemini CLI Instructions

This repo uses [AGENTS.md](AGENTS.md) as the canonical instruction file. Treat this file as compatibility guidance for Gemini-specific workflows.

MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
SYSTEMD_MCP_LIVE=1 go test ./... -count=1
make check
make test-live
```

## Architecture

Go program with D-Bus primary backend, explicit runtime capability probes for user and system scope, and systemctl/journalctl fallback only when those backends are actually usable.

## Key Conventions

- All tools default to user scope (`--user`). Set `system: true` for system scope.
- Critical services (sshd, NetworkManager, systemd-*, dbus, polkit) require `confirm: true` for stop/disable.
- Default `go test` is the deterministic tier. Live systemd integration requires `SYSTEMD_MCP_LIVE=1` and a host that actually exposes the requested scope.
- Thread safety: `sync.RWMutex` with `RLock` for reads, `Lock` for writes.
