# Review Guidelines — systemd-mcp

Inherits from org-wide [REVIEW.md](https://github.com/hairglasses-studio/.github/blob/main/REVIEW.md).

## Additional Focus
- **D-Bus safety**: Validate unit names before passing to systemd D-Bus interface
- **Privilege escalation**: Never allow start/stop/enable of arbitrary services without validation
- **Error messages**: Include unit name and operation in all error returns
- **Timeout handling**: D-Bus calls must have timeouts — never block indefinitely
