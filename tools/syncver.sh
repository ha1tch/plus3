#!/bin/sh
# syncver.sh - keep internal/version/version.go in sync with the canonical root
# VERSION file.
#
# Reads VERSION (a single "MAJOR.MINOR.PATCH" line) and rewrites the Version
# string in internal/version/version.go.
#
# Mirrors olu's syncver.sh (VERSION -> version.go). No sed/awk: the rewrite is
# done in Python (multiline-safe, guarded).
#
# Usage:  sh tools/syncver.sh            # sync version.go from VERSION
#         sh tools/syncver.sh 0.2.0      # set VERSION to 0.2.0, then sync

set -e
ROOT=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
VERFILE="$ROOT/VERSION"
VERGO="$ROOT/internal/version/version.go"

if [ -n "$1" ]; then
    printf '%s\n' "$1" > "$VERFILE"
fi

VER=$(tr -d ' \t\r\n' < "$VERFILE")
case "$VER" in
    *.*.*) : ;;
    *) echo "syncver: VERSION '$VER' is not MAJOR.MINOR.PATCH" >&2; exit 1 ;;
esac

VERGO="$VERGO" VER="$VER" python3 - <<'PY'
import os, re, sys

path = os.environ["VERGO"]
ver  = os.environ["VER"]

with open(path) as f:
    src = f.read()
orig = src

new, n = re.subn(r'(var Version = ")[0-9]+\.[0-9]+\.[0-9]+(")',
                 rf'\g<1>{ver}\g<2>', src, count=1)
if n != 1:
    sys.stderr.write("syncver: could not find Version string in version.go\n")
    sys.exit(1)

if new == orig:
    print(f"syncver: already at {ver}, no change")
else:
    with open(path, "w") as f:
        f.write(new)
    print(f"syncver: version.go synced to {ver}")
PY
