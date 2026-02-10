#!/bin/bash
# Build .deb package from binary
# Usage: ./build-deb.sh v0.1.0 amd64

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Parse arguments
VERSION="${1:-}"
ARCH="${2:-}"

if [ -z "$VERSION" ] || [ -z "$ARCH" ]; then
    echo "Usage: $0 <version> <arch>"
    echo "Example: $0 v0.1.0 amd64"
    echo "Supported architectures: amd64, arm64"
    exit 1
fi

# Strip 'v' prefix from version if present
DEB_VERSION="${VERSION#v}"

# Map Go arch to Debian arch
DEB_ARCH="$ARCH"

# Validate architecture
case "$ARCH" in
    amd64|arm64)
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        echo "Supported architectures: amd64, arm64"
        exit 1
        ;;
esac

echo "Building .deb package for nkudo-edge ${VERSION} (${ARCH})..."

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

# Create package directory structure
PKG_DIR="${BUILD_DIR}/nkudo-edge_${DEB_VERSION}_${DEB_ARCH}"
mkdir -p "${PKG_DIR}/DEBIAN"
mkdir -p "${PKG_DIR}/usr/local/bin"
mkdir -p "${PKG_DIR}/etc/systemd/system"
mkdir -p "${PKG_DIR}/etc/nkudo-edge"
mkdir -p "${PKG_DIR}/var/lib/nkudo-edge"

# Copy control file and substitute variables
sed -e "s/{{VERSION}}/${DEB_VERSION}/g" \
    -e "s/{{ARCH}}/${DEB_ARCH}/g" \
    "${SCRIPT_DIR}/control" > "${PKG_DIR}/DEBIAN/control"

# Copy maintainer scripts
cp "${SCRIPT_DIR}/postinst" "${PKG_DIR}/DEBIAN/"
cp "${SCRIPT_DIR}/prerm" "${PKG_DIR}/DEBIAN/"
cp "${SCRIPT_DIR}/postrm" "${PKG_DIR}/DEBIAN/"
chmod 755 "${PKG_DIR}/DEBIAN/postinst"
chmod 755 "${PKG_DIR}/DEBIAN/prerm"
chmod 755 "${PKG_DIR}/DEBIAN/postrm"

# Copy binary
BINARY_NAME="nkudo-edge-linux-${ARCH}"
if [ -f "${PROJECT_ROOT}/dist/${BINARY_NAME}" ]; then
    cp "${PROJECT_ROOT}/dist/${BINARY_NAME}" "${PKG_DIR}/usr/local/bin/nkudo-edge"
elif [ -f "${PROJECT_ROOT}/bin/edge" ]; then
    cp "${PROJECT_ROOT}/bin/edge" "${PKG_DIR}/usr/local/bin/nkudo-edge"
else
    echo "Error: Binary not found. Please build first:"
    echo "  GOOS=linux GOARCH=${ARCH} go build -o dist/${BINARY_NAME} ./cmd/edge"
    exit 1
fi
chmod 755 "${PKG_DIR}/usr/local/bin/nkudo-edge"

# Copy systemd service file
cp "${PROJECT_ROOT}/deployments/systemd/nkudo-edge.service" "${PKG_DIR}/etc/systemd/system/"
chmod 644 "${PKG_DIR}/etc/systemd/system/nkudo-edge.service"

# Create default environment file
cat > "${PKG_DIR}/etc/nkudo-edge/nkudo-edge.env" << 'EOF'
# n-kudo Edge Agent Configuration
CONTROL_PLANE_URL=https://api.nkudo.io
LOG_LEVEL=info
EOF
chmod 640 "${PKG_DIR}/etc/nkudo-edge/nkudo-edge.env"

# Build the package
dpkg-deb --build "${PKG_DIR}"

# Move package to dist directory
mkdir -p "${PROJECT_ROOT}/dist"
mv "${PKG_DIR}.deb" "${PROJECT_ROOT}/dist/nkudo-edge_${DEB_VERSION}_${DEB_ARCH}.deb"

echo "Package built: dist/nkudo-edge_${DEB_VERSION}_${DEB_ARCH}.deb"

# Generate checksum
cd "${PROJECT_ROOT}/dist"
sha256sum "nkudo-edge_${DEB_VERSION}_${DEB_ARCH}.deb" > "nkudo-edge_${DEB_VERSION}_${DEB_ARCH}.deb.sha256"

echo "Checksum: dist/nkudo-edge_${DEB_VERSION}_${DEB_ARCH}.deb.sha256"
