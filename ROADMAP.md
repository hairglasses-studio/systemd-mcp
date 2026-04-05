# Roadmap

## Current State

systemd-mcp provides 10 tools for systemd service and timer management via MCP. Single-file Go binary, shells out to `systemctl` and `journalctl` (no D-Bus dependency). User scope by default, system scope opt-in. Built on mcpkit with stdio transport.

All 10 tools are functional and tested. MIT licensed, README and CLAUDE.md in place.

## Planned

### Phase 1 — Hardening & Observability
- Add integration tests using `mcptest.NewServer()`
- Structured JSON output for `systemd_list_units` and `systemd_list_timers` (machine-parseable)
- Input validation for unit names (reject path traversal, shell metacharacters)
- Rate limiting on start/stop/restart to prevent rapid-fire toggles

### Phase 2 — Timer & Journal Enhancements
- `systemd_create_timer` — create transient timers from MCP (OnCalendar, OnBootSec)
- `systemd_log_search` — grep/filter journal logs by priority, pattern, or time range
- Pagination support for `systemd_logs` (offset + cursor-based)

## Future Considerations
- D-Bus integration as optional backend (eliminates shell overhead, enables event subscriptions)
- `systemd_watch` — streaming unit state changes via SSE transport
- Composed `investigate_unit` tool (status + logs + dependencies in one call)
- Support for `systemd-analyze` blame/critical-chain output
