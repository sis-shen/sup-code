#!/usr/bin/env bash
# bench.sh ? SupCode Phase 1 ??????
# ?? SRS ? 5.1 ??????
# ????: bash scripts/bench.sh
# ??: go, /usr/bin/time (Linux/macOS)

set -euo pipefail

BINARY="supcode"
TMPDIR=$(mktemp -d /tmp/supcode_bench.XXXXXX 2>/dev/null || mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "=== SupCode Phase 1 ?????? ==="
echo "??: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# ?? 1. ?? ?????????????????????????????????????????????????
echo "--- 1. ?? ---"
STARTUP_LOG="$TMPDIR/startup.txt"

go build -o "$BINARY" ./cmd/supcode
echo "  Build: OK"

# ?? 2. ????? ??????????????????????????????????????????
echo ""
echo "--- 2. ????? (--help) ---"

if command -v /usr/bin/time &>/dev/null; then
    for i in 1 2 3; do
        /usr/bin/time -f "%e" "./$BINARY" --help > /dev/null 2>> "$STARTUP_LOG"
    done
    avg=$(awk '{s+=$1} END {printf "%.3f", s/NR}' "$STARTUP_LOG")
    echo "  Run 1-3: $(paste -s -d ', ' "$STARTUP_LOG")s"
    echo "  Average: ${avg}s  (target: < 0.5s)"
    if [ "$(echo "$avg < 0.5" | bc -l 2>/dev/null || echo 0)" = "1" ]; then
        echo "  Result: PASS"
    else
        echo "  Result: CHECK (threshold 0.5s)"
    fi
else
    echo "  /usr/bin/time not available. Install 'time' package or use manual timing."
    for i in 1 2 3; do
        start=$(date +%s%N)
        "./$BINARY" --help > /dev/null 2>&1
        end=$(date +%s%N)
        elapsed=$(echo "scale=3; ($end - $start) / 1000000000" | bc)
        echo "$elapsed" >> "$STARTUP_LOG"
    done
    avg=$(awk '{s+=$1} END {printf "%.3f", s/NR}' "$STARTUP_LOG")
    echo "  Average: ${avg}s  (target: < 0.5s)"
fi

# ?? 3. ????? ??????????????????????????????????????????
echo ""
echo "--- 3. ????? ---"
SIZE=$(ls -lh "$BINARY" 2>/dev/null | awk '{print $5}')
echo "  Binary size: $SIZE  (target: < 30MB)"
if [ -n "$SIZE" ]; then
    bytes=$(ls -l "$BINARY" | awk '{print $5}')
    if [ "$bytes" -lt 31457280 ]; then
        echo "  Result: PASS"
    else
        echo "  Result: CHECK (exceeds 30MB)"
    fi
fi

# ?? 4. ???? (? Linux/macOS) ???????????????????????????
echo ""
echo "--- 4. ???? ---"
case "$(uname -s)" in
    Linux)
        # ?? supcode, ?? 500ms, ?? RSS, ?? kill
        "./$BINARY" --help > /dev/null 2>&1 &
        PID=$!
        sleep 0.5
        if [ -d "/proc/$PID" ]; then
            RSS=$(awk '/VmRSS/ {print $2}' "/proc/$PID/status" 2>/dev/null || echo "N/A")
            echo "  RSS: ${RSS}kB  (target: < 100MB)"
        fi
        kill "$PID" 2>/dev/null || true
        ;;
    Darwin)
        "./$BINARY" --help > /dev/null 2>&1 &
        PID=$!
        sleep 0.5
        RSS=$(ps -o rss= -p "$PID" 2>/dev/null || echo "N/A")
        echo "  RSS: ${RSS}kB  (target: < 100MB)"
        kill "$PID" 2>/dev/null || true
        ;;
    *)
        echo "  Memory measurement not supported on $(uname -s)."
        echo "  On Windows, use: Measure-Command { .\supcode.exe --help }"
        echo "  and check Task Manager for memory."
        ;;
esac

echo ""
echo "=== ?????? ==="
