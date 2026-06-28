#!/bin/sh
# build.sh - build the plus3 CLI binary.
#
# Requires: Go (>= 1.25) on PATH.
# Produces: ./plus3
#
# The version is taken from the root VERSION file and injected at build time via
# -ldflags, so the binary reports the correct version even without running
# syncver first.

set -e
ROOT=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$ROOT"

VER=$(tr -d ' \t\r\n' < VERSION)
LDFLAGS="-X github.com/ha1tch/plus3/internal/version.Version=$VER"

echo "Building plus3 $VER ..."
go build -ldflags "$LDFLAGS" -o plus3 ./cmd

echo "Done. ./plus3 (version $(./plus3 --version | awk '{print $NF}'))"
