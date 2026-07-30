#!/usr/bin/env bash
# Verifies the local toolchain and installs JS dependencies.
# Usage: ./scripts/bootstrap.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: '$1' is required but was not found on PATH." >&2
    echo "       $2" >&2
    exit 1
  fi
}

echo "==> Checking toolchain"
require node "Install Node.js >= 20: https://nodejs.org"
require pnpm "Enable via corepack: 'corepack enable && corepack prepare pnpm@latest --activate'"
require go "Install Go: https://go.dev/doc/install"

node_major="$(node -p 'process.versions.node.split(".")[0]')"
if [ "$node_major" -lt 20 ]; then
  echo "error: Node.js >= 20 is required (found $(node -v))." >&2
  exit 1
fi

echo "==> Installing JS dependencies"
pnpm install --frozen-lockfile=false

echo "==> Downloading Go dependencies"
go mod download

echo "==> Done. Try: pnpm build && go build ./..."
