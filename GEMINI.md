# systemd-mcp — Gemini CLI Instructions

This repo uses [AGENTS.md](AGENTS.md) as the canonical instruction file. Treat this file as compatibility guidance for Gemini-specific workflows.

MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
make check              # All three above
```

## Architecture

Go program with D-Bus primary backend and systemctl/journalctl fallback. One `SystemdModule` registers all 10 tools. D-Bus connections are optional — if unavailable, tools transparently fall back to shell commands.

## Key Conventions

- All tools default to user scope (`--user`). Set `system: true` for system scope.
- Critical services (sshd, NetworkManager, systemd-*, dbus, polkit) require `confirm: true` for stop/disable.
- Error codes: `handler.CodedErrorResult(handler.ErrInvalidParam, err)` — never `(nil, error)`.
- Thread safety: `sync.RWMutex` with `RLock` for reads, `Lock` for writes.
