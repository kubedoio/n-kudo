#!/usr/bin/env bash
set -euo pipefail

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
DIST_DIR=${DIST_DIR:-./dist}

echo "Building n-kudo control-plane..."
echo "Version: $VERSION"

mkdir -p "$DIST_DIR"

# Build native binary
echo "Building control-plane binary..."
go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "$DIST_DIR/nkudo-control-plane" \
    ./cmd/control-plane

echo "Build complete. Artifacts in $DIST_DIR:"
ls -la "$DIST_DIR/"
