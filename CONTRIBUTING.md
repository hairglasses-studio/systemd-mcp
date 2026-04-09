# Contributing to systemd-mcp

Thank you for your interest in contributing to systemd-mcp. This document
covers everything you need to get started.

## Reporting Bugs

Open a [GitHub Issue](https://github.com/hairglasses-studio/systemd-mcp/issues)
with:

- Go version, OS, and systemd version
- Minimal reproduction steps
- Expected vs. actual behavior
- Relevant error output or logs

## Suggesting Features

Open a [GitHub Issue](https://github.com/hairglasses-studio/systemd-mcp/issues)
with the `enhancement` label. Describe the use case and which systemd operations
are involved. If suggesting a new MCP tool, include the proposed tool name and
parameter schema.

## Submitting Pull Requests

1. Fork the repository and clone your fork.
2. Create a branch from `main`: `git checkout -b feat/my-change`
3. Make your changes, following the code style below.
4. Run the narrowest valid check suite for your host:
   - `make build && make vet && make test-unit`
   - `make test-live` only when your host exposes a usable live systemd environment
5. Commit with a clear, descriptive message.
6. Push your branch and open a PR against `main`.

Keep PRs focused. One logical change per PR is easier to review than a combined
refactor-plus-feature.

## Development Setup

**Requirements:** Go 1.26.1+, Linux with systemd

```bash
git clone https://github.com/hairglasses-studio/systemd-mcp
cd systemd-mcp
make build    # go build -o systemd-mcp ./...
make test     # deterministic default tier (live tests skip unless enabled)
make test-live
make vet      # go vet ./...
```

## Code Style

- Format with `gofmt` (or `goimports`).
- Pass `go vet ./...` with no warnings.
- Follow existing patterns in the codebase.
- New tools must implement the `ToolModule` interface (`Name()`, `Description()`, `Tools()`).
- Use `handler.TypedHandler` for new tool handlers.
- Return errors via `handler.CodedErrorResult` -- never `(nil, error)`.
- Protect shared state with `sync.RWMutex` (`RLock` for reads, `Lock` for writes).

## Testing Requirements

- All existing tests must pass before submitting a PR.
- Add tests for new features and bug fixes.
- Run the default deterministic tier with race detection: `go test ./... -count=1 -race`
- Run live integration coverage with `SYSTEMD_MCP_LIVE=1 go test ./... -count=1 -race` only when the host can satisfy the requested scope.
- Integration tests use `mcptest.NewServer()` plus explicit capability probes; unit tests use stdlib `testing`.

### Environment Matrix

| Tier | Requires |
|------|----------|
| Default `go test` | Go toolchain only; live tests will skip when disabled or unsupported |
| Live user-scope | `SYSTEMD_MCP_LIVE=1`, Linux, `systemctl`, reachable user manager |
| Live system-scope | `SYSTEMD_MCP_LIVE=1`, Linux, `systemctl`, reachable system manager, appropriate permissions; intended for manual or dedicated-runner execution |

If your host has `systemctl` installed but `systemctl --user` cannot reach a user manager, that is an environment limitation, not a unit-test failure. The runtime capability report at `systemd://runtime/capabilities` is the source of truth for what the current host can actually support.

## Branch Naming

Use `type/short-description`:

| Prefix | Use for |
|--------|---------|
| `feat/` | New features |
| `fix/` | Bug fixes |
| `docs/` | Documentation changes |
| `test/` | Test additions or improvements |
| `refactor/` | Code restructuring without behavior changes |
| `chore/` | Dependency updates, CI, tooling |

Examples: `feat/systemd-reload`, `fix/journal-filter-crash`, `docs/timer-examples`

## Commit Messages

Use conventional-style prefixes:

```
feat: add systemd_reload tool
fix: handle missing unit file gracefully
docs: document timer listing behavior
test: add tests for journal log filtering
```

## Review Timeline

| Priority | First response |
|----------|---------------|
| P0 (security/production) | 4 hours |
| P1 (blocking bug) | 24 hours |
| P2 (feature/enhancement) | 3 days |
| P3 (docs/chore) | 7 days |

## Getting Help

- **GitHub Issues:** Bug reports and feature requests
- **CLAUDE.md** in the repo root: Architecture overview and key patterns
- **README.md**: Quick start guide and tool reference

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).
