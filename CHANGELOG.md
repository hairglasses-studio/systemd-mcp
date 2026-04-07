# Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Changed
- D-Bus primary backend with systemctl/journalctl fallback (documentation corrected)
- Expanded .gitignore, SECURITY.md, CONTRIBUTING.md, Makefile
- Added Go Report Card, pkg.go.dev, CI badges to README
- Consolidated goreleaser configs, fixed golangci-lint config
- Fixed errcheck lint issues in test files

## [1.0.0] - 2026-04-04

### Added
- 10 MCP tools for systemd service and timer management
- User scope by default, system scope opt-in
- Critical service safety guards (confirm required for stop/disable)
- D-Bus backend with systemctl fallback
- Structured logging via slog
- MIT license and documentation
