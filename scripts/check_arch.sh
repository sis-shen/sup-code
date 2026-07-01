#!/usr/bin/env bash
# check_arch.sh — Architecture compliance checker
# Verifies package import direction rules from Architecture Design §6.
# Usage: bash scripts/check_arch.sh
# Exit: 0 = PASS, 1 = FAIL

set -euo pipefail

MODULE="github.com/supcode/supcode"
errors=0
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

check_no_import() {
    local pkg="$1"
    local forbidden="$2"
    local pattern="\"${MODULE}/${forbidden}\""
    local matched
    matched=$(grep -r "$pattern" "$pkg" --include="*.go" 2>/dev/null || true)
    if [ -n "$matched" ]; then
        echo -e "${RED}VIOLATION${NC}: ${pkg} imports ${forbidden}"
        echo "$matched"
        errors=$((errors + 1))
    fi
}

echo "=== SupCode Architecture Check ==="
echo ""

echo "[Rule 1] internal/tools -> internal/agent (DENY)"
check_no_import "internal/tools" "internal/agent"

echo "[Rule 2] internal/contextmgr -> internal/tools (DENY)"
check_no_import "internal/contextmgr" "internal/tools"

echo "[Rule 3] internal/permission -> internal/agent (DENY)"
check_no_import "internal/permission" "internal/agent"

echo "[Rule 4] pkg/ -> internal/ (DENY)"
check_no_import "pkg" "internal/"

echo "[Rule 5] internal/tui -> internal/tools (DENY)"
check_no_import "internal/tui" "internal/tools"

echo "[Rule 6] internal/cli -> internal/tools (DENY)"
check_no_import "internal/cli" "internal/tools"

echo ""
echo "=== Result ==="
if [ "$errors" -gt 0 ]; then
    echo -e "${RED}FAILED${NC}: $errors architecture violation(s) found"
    exit 1
fi
echo -e "${GREEN}PASS${NC}: All architecture rules satisfied"
exit 0