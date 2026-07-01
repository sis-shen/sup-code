#!/usr/bin/env bash
# build.sh — 全平台交叉编译
set -euo pipefail

echo "==> Cross-compiling..."

GOOS=linux   GOARCH=amd64 go build -o build/supcode-linux-amd64   ./cmd/supcode
GOOS=darwin  GOARCH=amd64 go build -o build/supcode-darwin-amd64  ./cmd/supcode
GOOS=darwin  GOARCH=arm64 go build -o build/supcode-darwin-arm64  ./cmd/supcode
GOOS=windows GOARCH=amd64 go build -o build/supcode-windows-amd64.exe ./cmd/supcode

echo "==> Built:"
ls -lh build/
