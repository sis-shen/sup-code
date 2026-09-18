#!/usr/bin/env bash
# check_task_checkboxes.sh — verify phase task boards expose checkbox items.
# If no phase boards exist yet, the check is a no-op (passes).
# Usage: bash scripts/check_task_checkboxes.sh ; Exit 0 = PASS, 1 = FAIL
set -euo pipefail

fail=0
found=0

for readme in tasks/phase-*/README.md; do
  [ -e "$readme" ] || continue
  found=$((found + 1))
  if ! grep -qE '^[[:space:]]*-[[:space:]]*\[[ xX]\]' "$readme"; then
    echo "FAIL: $readme has no checkbox items"
    fail=1
  fi
done

if [ "$fail" -eq 0 ]; then
  echo "task checklists OK ($found board(s))"
fi
exit "$fail"
