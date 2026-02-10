#!/bin/bash
# Build .rpm package from binary
# Usage: ./build-rpm.sh v0.1.0 amd64

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Parse arguments
VERSION="${1:-}"
ARCH="${2:-}"

if [ -z "$VERSION" ] || [ -z "$ARCH" ]; then
    echo "Usage: $0 <version> <arch>"
    echo "Example: $0 v0.1.0 amd64"
    echo "Supported architectures: amd64 (x86_64), arm64 (aarch64)"
    exit 1
fi

# Strip 'v' prefix from version if present
RPM_VERSION="${VERSION#v}"

# Map Go arch to RPM arch
case "$ARCH" in
    amd64)
        RPM_ARCH="x86_64"
        ;;
    arm64)
        RPM_ARCH="aarch64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        echo "Supported architectures: amd64, arm64"
        exit 1
        ;;
esac

echo "Building .rpm package for nkudo-edge ${VERSION} (${ARCH} -> ${RPM_ARCH})..."

# Check for rpmbuild
if ! command -v rpmbuild &> /dev/null; then
    echo "Error: rpmbuild not found. Please install rpm-build package:"
    echo "  Fedora/RHEL: sudo dnf install rpm-build"
    echo "  CentOS: sudo yum install rpm-build"
    exit 1
fi

# Create RPM build directory structure
RPM_BUILD_DIR="${HOME}/rpmbuild"
mkdir -p "${RPM_BUILD_DIR}"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Create source tarball
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

SOURCE_DIR="${BUILD_DIR}/nkudo-edge-${RPM_VERSION}"
mkdir -p "${SOURCE_DIR}"

# Copy binary
BINARY_NAME="nkudo-edge-linux-${ARCH}"
if [ -f "${PROJECT_ROOT}/dist/${BINARY_NAME}" ]; then
    cp "${PROJECT_ROOT}/dist/${BINARY_NAME}" "${SOURCE_DIR}/nkudo-edge"
elif [ -f "${PROJECT_ROOT}/bin/edge" ]; then
    cp "${PROJECT_ROOT}/bin/edge" "${SOURCE_DIR}/nkudo-edge"
else
    echo "Error: Binary not found. Please build first:"
    echo "  GOOS=linux GOARCH=${ARCH} go build -o dist/${BINARY_NAME} ./cmd/edge"
    exit 1
fi

# Copy systemd service file
cp "${PROJECT_ROOT}/deployments/systemd/nkudo-edge.service" "${SOURCE_DIR}/"

# Copy default environment file
cat > "${SOURCE_DIR}/nkudo-edge.env" << 'EOF'
# n-kudo Edge Agent Configuration
CONTROL_PLANE_URL=https://api.nkudo.io
LOG_LEVEL=info
EOF

# Create a minimal LICENSE file if not exists
if [ -f "${PROJECT_ROOT}/LICENSE" ]; then
    cp "${PROJECT_ROOT}/LICENSE" "${SOURCE_DIR}/"
else
    echo "Apache-2.0" > "${SOURCE_DIR}/LICENSE"
fi

# Create a minimal README
if [ -f "${PROJECT_ROOT}/README.md" ]; then
    cp "${PROJECT_ROOT}/README.md" "${SOURCE_DIR}/"
else
    echo "n-kudo Edge Agent" > "${SOURCE_DIR}/README.md"
fi

# Create tarball
cd "${BUILD_DIR}"
tar czf "${RPM_BUILD_DIR}/SOURCES/nkudo-edge-${RPM_VERSION}.tar.gz" "nkudo-edge-${RPM_VERSION}"

# Copy and process spec file
sed -e "s/{{VERSION}}/${RPM_VERSION}/g" \
    "${SCRIPT_DIR}/nkudo-edge.spec" > "${RPM_BUILD_DIR}/SPECS/nkudo-edge.spec"

# Build RPM
cd "${RPM_BUILD_DIR}"
rpmbuild --define "_topdir ${RPM_BUILD_DIR}" \
         --define "_build_name_fmt %%{NAME}-%%{VERSION}-%%{RELEASE}.%%{ARCH}.rpm" \
         -bb SPECS/nkudo-edge.spec

# Copy built RPM to dist directory
mkdir -p "${PROJECT_ROOT}/dist"
RPM_FILE="nkudo-edge-${RPM_VERSION}-1.${RPM_ARCH}.rpm"
cp "${RPM_BUILD_DIR}/RPMS/${RPM_ARCH}/${RPM_FILE}" "${PROJECT_ROOT}/dist/"

echo "Package built: dist/${RPM_FILE}"

# Generate checksum
cd "${PROJECT_ROOT}/dist"
sha256sum "${RPM_FILE}" > "${RPM_FILE}.sha256"

echo "Checksum: dist/${RPM_FILE}.sha256"
