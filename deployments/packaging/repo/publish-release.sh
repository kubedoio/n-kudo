#!/bin/bash
# Publish packages to GitHub Releases
# - Build deb and rpm for multiple architectures
# - Upload to GitHub Release

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Parse arguments
VERSION="${1:-}"
DRY_RUN="${DRY_RUN:-false}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--dry-run]"
    echo "Example: $0 v0.1.0"
    echo "Environment variables:"
    echo "  DRY_RUN=true    - Build packages without uploading"
    echo "  GITHUB_TOKEN    - GitHub token for uploading (required for non-dry-run)"
    exit 1
fi

# Strip 'v' prefix for consistency
VERSION="${VERSION#v}"
VERSION="v${VERSION}"

ARCHS=("amd64" "arm64")

echo "========================================"
echo "Publishing Release: ${VERSION}"
echo "========================================"
echo ""

# Check prerequisites
check_prerequisites() {
    if [ "$DRY_RUN" != "true" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
        echo "Error: GITHUB_TOKEN environment variable not set"
        exit 1
    fi
    
    # Check for required tools
    if ! command -v gh &> /dev/null && [ "$DRY_RUN" != "true" ]; then
        echo "Error: GitHub CLI (gh) not found"
        echo "Install from: https://cli.github.com/"
        exit 1
    fi
}

# Build binaries
build_binaries() {
    echo "Building binaries..."
    
    cd "$PROJECT_ROOT"
    
    for arch in "${ARCHS[@]}"; do
        echo "  Building for linux/${arch}..."
        
        BINARY_NAME="nkudo-edge-linux-${arch}"
        
        GOOS=linux GOARCH="$arch" CGO_ENABLED=0 \
            go build -trimpath \
            -ldflags "-s -w -X main.version=${VERSION}" \
            -o "dist/${BINARY_NAME}" \
            ./cmd/edge
        
        # Create tarball
        cd dist
        tar czf "${BINARY_NAME}.tar.gz" "${BINARY_NAME}"
        sha256sum "${BINARY_NAME}.tar.gz" > "${BINARY_NAME}.tar.gz.sha256"
        cd "$PROJECT_ROOT"
    done
    
    echo "  Binaries built successfully"
    echo ""
}

# Build .deb packages
build_deb_packages() {
    echo "Building .deb packages..."
    
    for arch in "${ARCHS[@]}"; do
        echo "  Building .deb for ${arch}..."
        "${PROJECT_ROOT}/deployments/packaging/deb/build-deb.sh" "$VERSION" "$arch"
    done
    
    echo "  .deb packages built successfully"
    echo ""
}

# Build .rpm packages
build_rpm_packages() {
    echo "Building .rpm packages..."
    
    for arch in "${ARCHS[@]}"; do
        echo "  Building .rpm for ${arch}..."
        "${PROJECT_ROOT}/deployments/packaging/rpm/build-rpm.sh" "$VERSION" "$arch"
    done
    
    echo "  .rpm packages built successfully"
    echo ""
}

# Create or verify GitHub release
create_release() {
    if [ "$DRY_RUN" = "true" ]; then
        echo "[DRY RUN] Would create GitHub release: ${VERSION}"
        return
    fi
    
    echo "Creating GitHub release..."
    
    # Check if release exists
    if gh release view "$VERSION" --repo "${GITHUB_REPOSITORY:-}" &>/dev/null; then
        echo "  Release ${VERSION} already exists"
    else
        gh release create "$VERSION" \
            --title "Release ${VERSION}" \
            --generate-notes \
            --draft=false \
            --prerelease=false
        echo "  Release created"
    fi
    
    echo ""
}

# Upload packages to GitHub release
upload_packages() {
    if [ "$DRY_RUN" = "true" ]; then
        echo "[DRY RUN] Would upload packages to GitHub release: ${VERSION}"
        ls -la "${PROJECT_ROOT}/dist/"*.deb 2>/dev/null || true
        ls -la "${PROJECT_ROOT}/dist/"*.rpm 2>/dev/null || true
        return
    fi
    
    echo "Uploading packages to GitHub release..."
    
    cd "${PROJECT_ROOT}/dist"
    
    # Upload all packages
    for file in *.deb *.rpm *.sha256 *.tar.gz; do
        if [ -f "$file" ]; then
            echo "  Uploading: $file"
            gh release upload "$VERSION" "$file" --clobber
        fi
    done
    
    echo "  Packages uploaded successfully"
    echo ""
}

# Main execution
main() {
    echo "DRY_RUN: ${DRY_RUN}"
    echo ""
    
    check_prerequisites
    
    # Create dist directory
    mkdir -p "${PROJECT_ROOT}/dist"
    
    # Build packages
    build_binaries
    build_deb_packages
    build_rpm_packages
    
    # Create and upload to GitHub
    create_release
    upload_packages
    
    echo "========================================"
    echo "Release ${VERSION} published successfully!"
    echo "========================================"
    echo ""
    echo "Packages:"
    ls -la "${PROJECT_ROOT}/dist/"*.{deb,rpm} 2>/dev/null || true
    echo ""
    echo "Release URL: https://github.com/${GITHUB_REPOSITORY:-}/releases/tag/${VERSION}"
}

main "$@"
