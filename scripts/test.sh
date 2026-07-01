#!/usr/bin/env bash
# test.sh — 运行所有测试（含 race detector）
set -euo pipefail

echo "==> go test -race -count=1 ./..."
go test -race -count=1 ./...
echo "==> All tests passed"
