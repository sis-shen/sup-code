#!/usr/bin/env bash
# lint.sh — 运行所有 linter
set -euo pipefail

echo "==> golangci-lint run ./..."
golangci-lint run ./...
echo "==> Done (0 issues)"
