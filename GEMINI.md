# systemd-mcp — Gemini CLI Instructions

MCP server for systemd service and timer management. Built on mcpkit (stdio transport).

## Build & Test

```bash
go build -o systemd-mcp ./...
go vet ./...
go test ./... -count=1
```

## Architecture


