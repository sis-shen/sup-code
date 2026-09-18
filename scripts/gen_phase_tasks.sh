#!/usr/bin/env bash
# gen_phase_tasks.sh — 生成阶段任务看板与验收报告骨架（Linux/macOS 包装）
# 真正的生成逻辑在 gen_phase_tasks.py（UTF-8 安全）。
# Usage: bash scripts/gen_phase_tasks.sh [--force]
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
if command -v python3 >/dev/null 2>&1; then
  PY=python3
elif command -v python >/dev/null 2>&1; then
  PY=python
else
  echo "python not found; please install Python 3" >&2
  exit 1
fi
exec "$PY" "$HERE/gen_phase_tasks.py" "$@"
