# Roadmap

## Current State

systemd-mcp provides 10 tools for systemd service and timer management via MCP. Single Go binary with D-Bus primary backend and systemctl/journalctl fallback. User scope by default, system scope opt-in. Built on mcpkit with stdio transport.

All 10 tools are functional and tested. MIT licensed, README and CLAUDE.md in place.

## Planned

### Phase 1 — Hardening & Observability
- Add integration tests using `mcptest.NewServer()`
- Add backend capability detection and an environment-aware integration harness
- Structured JSON output for `systemd_list_units` and `systemd_list_timers` (machine-parseable)
- Input validation for unit names (reject path traversal, shell metacharacters)
- Rate limiting on start/stop/restart to prevent rapid-fire toggles

### Phase 2 — Timer & Journal Enhancements
- `systemd_create_timer` — create transient timers from MCP (OnCalendar, OnBootSec)
- `systemd_log_search` — grep/filter journal logs by priority, pattern, or time range
- Pagination support for `systemd_logs` (offset + cursor-based)

## Future Considerations
- D-Bus event subscriptions (unit state change notifications via SSE)
- `systemd_watch` — streaming unit state changes via SSE transport
- Composed `investigate_unit` tool (status + logs + dependencies in one call)
- Support for `systemd-analyze` blame/critical-chain output
---

## Crosspollinate Suggestion: Adopt go-mcp-server pattern

> **Source:** `~/hairglasses-studio/crosspollinate/patterns/go-mcp-server.md`
> **Proposed:** 2026-05-07 (cycle 0, refined cycle 13)
> **How to dismiss:** delete this section. Future crosspollinate cycles will detect the deletion and downgrade the recommendation.

The crosspollinate loop synthesized a canonical pattern for Go MCP servers across the 12-member cluster (hg-mcp, process-mcp, github-runner-mcp, systemd-mcp, tmux-mcp, codexkit, geminikit, jobb, mcp-catalog, terraform-docs, jellyfin-mcp-deluxe, mcpkit) based on context7 docs (mcp-go + official Go SDK + MCP spec) and exemplar code in ralphglasses.

Key recommendations relevant to this repo:

- **Dual-SDK build tags** with separate handler files (`handler_mcpgo.go` vs `handler_officialsdk.go`) — the two SDK signatures differ and cannot share handler bodies.
- **mcp-go error pattern**: validation/business errors → `mcp.NewToolResultError(msg), nil`; system errors → `nil, fmt.Errorf(...)`. Three cases, not one.
- **Deferred-loading tool group registry** instead of eager registration. Keeps cold-start memory bounded.
- **Discovery surfaces are MCP resources**, not tools (`<server>:///catalog/server`).

See the pattern doc for the full `# Adoption checklist` and `# Anti-patterns` sections.

