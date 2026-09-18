#!/usr/bin/env python3
"""Convert a Go coverprofile to LCOV so diff-cover can consume it.

Go's `-coverprofile` format is not LCOV; diff-cover only understands
lcov/Cobertura/Clover/JaCoCo. This script expands each Go cover block into
LCOV `DA:<line>,<hits>` records keyed by repository-relative paths.

Usage: python3 scripts/cover2lcov.py coverage.out > coverage.lcov
"""
import os
import sys

MODULE_PREFIX = "github.com/supcode/supcode/"


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: cover2lcov.py <coverprofile>", file=sys.stderr)
        return 2

    src = sys.argv[1]
    if not os.path.exists(src):
        print(f"coverage file not found: {src}", file=sys.stderr)
        return 1

    # file -> {line: hits}
    files: dict[str, dict[int, int]] = {}

    with open(src, encoding="utf-8") as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("mode:"):
                continue
            # path:startLine.startCol,endLine.endCol numStmts count
            try:
                loc, _stmts, hits = line.rsplit(" ", 2)
                path, positions = loc.rsplit(":", 1)
                start, end = positions.split(",")
                start_line = int(start.split(".")[0])
                end_line = int(end.split(".")[0])
                count = int(hits)
            except ValueError:
                continue

            if path.startswith(MODULE_PREFIX):
                path = path[len(MODULE_PREFIX):]
            path = path.replace("\\", "/")

            bucket = files.setdefault(path, {})
            for ln in range(start_line, end_line + 1):
                if count > bucket.get(ln, -1):
                    bucket[ln] = count

    out = sys.stdout
    for path in sorted(files):
        out.write(f"SF:{path}\n")
        for ln in sorted(files[path]):
            out.write(f"DA:{ln},{files[path][ln]}\n")
        out.write("end_of_record\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
