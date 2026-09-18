#!/usr/bin/env bash
# check_skills.sh — validate every skills/*/skill.json manifest.
# Usage: bash scripts/check_skills.sh ; Exit 0 = PASS, 1 = FAIL
set -euo pipefail

if [ ! -d skills ]; then
  echo "no skills directory, skip"
  exit 0
fi

# Resolve a working Python interpreter. On Windows the Microsoft Store alias
# (…/WindowsApps/python3.exe) exists but exits non-zero when invoked, so probe
# each candidate before trusting it.
PY=""
for candidate in python3 python; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c "import json,re,sys" >/dev/null 2>&1; then
    PY="$candidate"
    break
  fi
done
if [ -z "$PY" ] && command -v py >/dev/null 2>&1 && py -3 -c "import json,re,sys" >/dev/null 2>&1; then
  PY="py -3"
fi
if [ -z "$PY" ]; then
  echo "python interpreter not found; cannot validate skill manifests"
  exit 1
fi

fail=0
count=0

for manifest in skills/*/skill.json; do
  [ -e "$manifest" ] || continue
  count=$((count + 1))
  if ! $PY - "$manifest" <<'PY'
import json
import re
import sys

path = sys.argv[1]
with open(path, encoding="utf-8-sig") as fh:
    m = json.load(fh)

name = m.get("name", "")
version = m.get("version", "")
desc = m.get("description", "")

if not re.fullmatch(r"[a-z0-9-]+", name):
    print(f"  invalid name: {name!r} (must match [a-z0-9-])")
    sys.exit(1)
if not re.fullmatch(r"\d+\.\d+(\.\d+)?([-+].*)?", version):
    print(f"  invalid version: {version!r} (must be semver)")
    sys.exit(1)
if not desc:
    print("  missing description")
    sys.exit(1)
PY
  then
    echo "FAIL: $manifest"
    fail=1
  fi
done

echo "validated $count skill manifest(s)"
exit "$fail"
