#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"
export GOWORK=off

case "${SYSTEMD_MCP_SDK:-official}" in
  official)
    exec go run -tags=official_sdk ./cmd/systemd-mcp
    ;;
  legacy)
    exec go run ./cmd/systemd-mcp
    ;;
  *)
    printf 'unsupported SYSTEMD_MCP_SDK=%q (expected official or legacy)\n' "$SYSTEMD_MCP_SDK" >&2
    exit 2
    ;;
esac
