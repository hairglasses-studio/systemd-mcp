# Roadmap

## Current State

systemd-mcp provides 10 tools for systemd service and timer management via MCP. Single Go binary with D-Bus primary backend and systemctl/journalctl fallback. User scope by default, system scope opt-in. Built on mcpkit with stdio transport.

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
- D-Bus event subscriptions (unit state change notifications via SSE)
- `systemd_watch` — streaming unit state changes via SSE transport
- Composed `investigate_unit` tool (status + logs + dependencies in one call)
- Support for `systemd-analyze` blame/critical-chain output

<!-- whiteclaw-rollout:start -->
## Whiteclaw-Derived Overhaul (2026-04-08)

This tranche applies the highest-value whiteclaw findings that fit this repo's real surface: engineer briefs, bounded skills/runbooks, searchable provenance, scoped MCP packaging, and explicit verification ladders.

### Strategic Focus
- Treat this repo as a public mirror with a real operator-facing contract around systemd state and D-Bus behavior.
- Use whiteclaw patterns to harden contract visibility, scope documentation, and mirror verification rather than inventing extra local tooling.
- Keep the source-of-truth path to `dotfiles/mcp/systemd-mcp` explicit and enforceable.

### Recommended Work
- [ ] [Mirror contract] Keep the canonical-source mapping to `dotfiles/mcp/systemd-mcp` explicit and verifiable.
- [ ] [Contract tests] Add D-Bus vs fallback contract tests for the exported systemd investigation and control surfaces.
- [ ] [Scope docs] Document user-scope vs system-scope behavior and the permissions/runtime differences between them.
- [ ] [Publish verification] Add mirror smoke tests that prove the released artifact matches the canonical source surface.

### Rationale Snapshot
- Tier / lifecycle: `standalone` / `publish-mirror`
- Language profile: `Go`
- Visibility / sensitivity: `PUBLIC` / `public`
- Surface baseline: AGENTS=yes, skills=yes, codex=yes, mcp_manifest=configured, ralph=yes, roadmap=yes
- Whiteclaw transfers in scope: mirror contract, D-Bus vs fallback tests, user/system scope docs, publish verification
- Live repo notes: AGENTS, skills, Codex config, configured .mcp.json, .ralph, 1 workflow(s)

<!-- whiteclaw-rollout:end -->
