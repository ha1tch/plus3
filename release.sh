#!/bin/sh
# release.sh - plus3 release automation.
#
# Adapted from olu's release.sh, preserving its discipline for a Go CLI:
#   validate -> sync version -> build -> verify -> consistency -> clean ->
#   guarded zip (explicit allowlist, binary magic-byte sniff, size ceiling).
#
# Usage:  sh release.sh <version> [--source]
#   <version>   MAJOR.MINOR.PATCH; must match a CHANGELOG entry.
#   --source    package the full source tree (no size ceiling) instead of the
#               lean release artefact.
#
# Produces:  plus3-v<version>-checkpoint.zip   (lean: source, no built binary)
#       or:  plus3-v<version>-source.zip       (with --source)

set -e

ROOT=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$ROOT"

VERSION="$1"
MODE="checkpoint"
shift 2>/dev/null || true
for arg in "$@"; do
    case "$arg" in
        --source) MODE="source" ;;
        *) echo "release: unknown option '$arg'" >&2; exit 1 ;;
    esac
done

SIZE_CEILING=2000000      # 2 MB ceiling for checkpoint zips

die() { echo "release: $1" >&2; exit 1; }

# --- 1. validate -----------------------------------------------------------
[ -n "$VERSION" ] || die "usage: sh release.sh <version> [--source]"
case "$VERSION" in
    *.*.*) : ;;
    *) die "version '$VERSION' is not MAJOR.MINOR.PATCH" ;;
esac
grep -q "^## \[$VERSION\]" CHANGELOG.md \
    || die "no CHANGELOG.md entry for [$VERSION]"

# --- 2. sync version -------------------------------------------------------
echo "==> syncing version to $VERSION"
sh tools/syncver.sh "$VERSION"

# --- 3. build --------------------------------------------------------------
echo "==> building (go build with version injected)"
command -v go >/dev/null 2>&1 || die "go not on PATH"
LDFLAGS="-X github.com/ha1tch/plus3/internal/version.Version=$VERSION"
go build -ldflags "$LDFLAGS" -o plus3 ./cmd || die "build failed"

# --- 4. verify -------------------------------------------------------------
echo "==> verifying (build + vet of shipping packages)"
go build ./... >/dev/null 2>&1 || die "shipping packages do not build"
go vet ./internal/... ./pkg/diskimg/ ./cmd/... >/dev/null 2>&1 \
    || echo "    note: go vet reported issues (non-fatal)"
[ -f plus3 ] || die "plus3 binary missing after build"

# --- 5. consistency --------------------------------------------------------
FILEVER=$(tr -d ' \t\r\n' < VERSION)
[ "$FILEVER" = "$VERSION" ] || die "VERSION file ($FILEVER) != $VERSION"
BINVER=$(./plus3 --version | awk '{print $NF}')
[ "$BINVER" = "$VERSION" ] || die "binary version ($BINVER) != $VERSION"
echo "    version consistency OK ($VERSION)"

# --- 6. clean transient artefacts -----------------------------------------
echo "==> cleaning transient build artefacts"
rm -f plus3 /tmp/plus3_release_manifest.txt

# --- 7. assemble file list (explicit allowlist) ---------------------------
echo "==> building file allowlist"
ALLOW="VERSION CHANGELOG.md LICENSE NOTICE README.md build.sh release.sh go.mod go.sum"
ALLOW="$ALLOW cmd internal pkg tools doc .github"

MANIFEST=/tmp/plus3_release_manifest.txt
: > "$MANIFEST"
for path in $ALLOW; do
    if [ -d "$path" ]; then
        find "$path" -type f \
            ! -path '*/__pycache__/*' ! -name '*.pyc' \
            >> "$MANIFEST"
    elif [ -f "$path" ]; then
        echo "$path" >> "$MANIFEST"
    fi
done

# --- 8. guard: binary magic-byte sniff ------------------------------------
echo "==> guarding manifest (no stray binaries)"
while IFS= read -r f; do
    magic=$(dd if="$f" bs=1 count=4 2>/dev/null | od -An -tx1 | tr -d ' \n')
    case "$magic" in
        7f454c46|4d5a*|504b0304|1f8b*)
            die "binary artefact slipped into manifest: $f ($magic)" ;;
    esac
done < "$MANIFEST"

# --- 9. guarded zip -------------------------------------------------------
if [ "$MODE" = "checkpoint" ]; then
    OUT="plus3-v$VERSION-checkpoint.zip"
else
    OUT="plus3-v$VERSION-source.zip"
fi
rm -f "$OUT"
echo "==> packaging $OUT"
zip -q "$OUT" -@ < "$MANIFEST"

SZ=$(wc -c < "$OUT")
if [ "$MODE" = "checkpoint" ] && [ "$SZ" -gt "$SIZE_CEILING" ]; then
    die "$OUT is ${SZ}B, exceeds ceiling ${SIZE_CEILING}B"
fi

echo ""
echo "Release $VERSION packaged: $OUT (${SZ} bytes, mode=$MODE)"
