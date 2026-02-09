#!/usr/bin/env bash
set -euo pipefail

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
DIST_DIR=${DIST_DIR:-./dist}

echo "Building n-kudo edge agent..."
echo "Version: $VERSION"

mkdir -p "$DIST_DIR"

# Build for multiple architectures
platforms=(
    "linux/amd64"
    "linux/arm64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output="$DIST_DIR/nkudo-edge-${GOOS}-${GOARCH}"
    
    echo "Building for $GOOS/$GOARCH..."
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
        go build -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o "$output" \
        ./cmd/edge
    
    # Create tarball
    tar -czf "${output}.tar.gz" -C "$DIST_DIR" "nkudo-edge-${GOOS}-${GOARCH}"
    
    # Generate checksum
    sha256sum "${output}.tar.gz" > "${output}.tar.gz.sha256"
    
    echo "Built: $output"
done

echo "Build complete. Artifacts in $DIST_DIR:"
ls -la "$DIST_DIR/"
