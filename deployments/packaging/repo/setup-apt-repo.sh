#!/bin/bash
# Setup APT repository structure
# - Create pool/ dists/ structure
# - Generate Packages.gz
# - Generate Release file
# - Sign with GPG

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

REPO_DIR="${PROJECT_ROOT}/dist/apt-repo"
DISTRO="stable"
COMPONENT="main"
ARCHS="amd64 arm64"

# GPG key configuration
GPG_KEY_ID="${GPG_KEY_ID:-}"
GPG_PASSPHRASE="${GPG_PASSPHRASE:-}"

echo "Setting up APT repository structure..."

# Create repository structure
mkdir -p "${REPO_DIR}/pool/${COMPONENT}"
mkdir -p "${REPO_DIR}/dists/${DISTRO}/${COMPONENT}/binary-amd64"
mkdir -p "${REPO_DIR}/dists/${DISTRO}/${COMPONENT}/binary-arm64"

# Copy .deb packages
echo "Copying .deb packages..."
if [ -d "${PROJECT_ROOT}/dist" ]; then
    find "${PROJECT_ROOT}/dist" -name "*.deb" -exec cp {} "${REPO_DIR}/pool/${COMPONENT}/" \;
fi

# Generate Packages files for each architecture
generate_packages() {
    local arch="$1"
    local output_dir="${REPO_DIR}/dists/${DISTRO}/${COMPONENT}/binary-${arch}"
    
    echo "Generating Packages for ${arch}..."
    
    # Clear existing Packages file
    > "${output_dir}/Packages"
    
    # Process each .deb package
    for deb in "${REPO_DIR}/pool/${COMPONENT}"/*.deb; do
        if [ -f "$deb" ]; then
            # Check if package matches architecture
            pkg_arch=$(dpkg-deb -f "$deb" Architecture)
            if [ "$pkg_arch" = "$arch" ]; then
                dpkg-deb -I "$deb" >> "${output_dir}/Packages"
                echo "Filename: pool/${COMPONENT}/$(basename "$deb")" >> "${output_dir}/Packages"
                echo "Size: $(stat -c%s "$deb")" >> "${output_dir}/Packages"
                echo "MD5sum: $(md5sum "$deb" | cut -d' ' -f1)" >> "${output_dir}/Packages"
                echo "SHA256: $(sha256sum "$deb" | cut -d' ' -f1)" >> "${output_dir}/Packages"
                echo "" >> "${output_dir}/Packages"
            fi
        fi
    done
    
    # Compress Packages file
    gzip -k -f "${output_dir}/Packages" > "${output_dir}/Packages.gz"
}

# Generate Packages for each architecture
for arch in $ARCHS; do
    generate_packages "$arch"
done

# Generate Release file
generate_release() {
    local output_dir="${REPO_DIR}/dists/${DISTRO}"
    
    echo "Generating Release file..."
    
    cat > "${output_dir}/Release" << EOF
Origin: n-kudo
Label: n-kudo APT Repository
Suite: ${DISTRO}
Codename: ${DISTRO}
Version: 1.0
Architectures: ${ARCHS}
Components: ${COMPONENT}
Description: APT repository for n-kudo edge agent
Date: $(date -Ru)
EOF
    
    # Add checksums for each architecture
    echo "MD5Sum:" >> "${output_dir}/Release"
    for arch in $ARCHS; do
        packages_file="${COMPONENT}/binary-${arch}/Packages"
        packages_gz="${COMPONENT}/binary-${arch}/Packages.gz"
        
        if [ -f "${output_dir}/${packages_file}" ]; then
            size=$(stat -c%s "${output_dir}/${packages_file}")
            md5=$(md5sum "${output_dir}/${packages_file}" | cut -d' ' -f1)
            echo " ${md5} ${size} ${packages_file}" >> "${output_dir}/Release"
        fi
        
        if [ -f "${output_dir}/${packages_gz}" ]; then
            size=$(stat -c%s "${output_dir}/${packages_gz}")
            md5=$(md5sum "${output_dir}/${packages_gz}" | cut -d' ' -f1)
            echo " ${md5} ${size} ${packages_gz}" >> "${output_dir}/Release"
        fi
    done
    
    echo "SHA256:" >> "${output_dir}/Release"
    for arch in $ARCHS; do
        packages_file="${COMPONENT}/binary-${arch}/Packages"
        packages_gz="${COMPONENT}/binary-${arch}/Packages.gz"
        
        if [ -f "${output_dir}/${packages_file}" ]; then
            size=$(stat -c%s "${output_dir}/${packages_file}")
            sha256=$(sha256sum "${output_dir}/${packages_file}" | cut -d' ' -f1)
            echo " ${sha256} ${size} ${packages_file}" >> "${output_dir}/Release"
        fi
        
        if [ -f "${output_dir}/${packages_gz}" ]; then
            size=$(stat -c%s "${output_dir}/${packages_gz}")
            sha256=$(sha256sum "${output_dir}/${packages_gz}" | cut -d' ' -f1)
            echo " ${sha256} ${size} ${packages_gz}" >> "${output_dir}/Release"
        fi
    done
}

generate_release

# Sign Release file with GPG
sign_release() {
    local output_dir="${REPO_DIR}/dists/${DISTRO}"
    
    if [ -n "$GPG_KEY_ID" ]; then
        echo "Signing Release file with GPG key ${GPG_KEY_ID}..."
        
        if [ -n "$GPG_PASSPHRASE" ]; then
            # Use passphrase from environment
            gpg --batch --yes --passphrase "$GPG_PASSPHRASE" \
                --pinentry-mode loopback \
                -u "$GPG_KEY_ID" \
                -abs -o "${output_dir}/Release.gpg" "${output_dir}/Release"
            
            gpg --batch --yes --passphrase "$GPG_PASSPHRASE" \
                --pinentry-mode loopback \
                -u "$GPG_KEY_ID" \
                --clearsign -o "${output_dir}/InRelease" "${output_dir}/Release"
        else
            # Interactive signing
            gpg -u "$GPG_KEY_ID" -abs -o "${output_dir}/Release.gpg" "${output_dir}/Release"
            gpg -u "$GPG_KEY_ID" --clearsign -o "${output_dir}/InRelease" "${output_dir}/Release"
        fi
    else
        echo "Warning: GPG_KEY_ID not set. Release file not signed."
        echo "Set GPG_KEY_ID and optionally GPG_PASSPHRASE environment variables to sign."
    fi
}

sign_release

echo ""
echo "APT repository created at: ${REPO_DIR}"
echo ""
echo "To use this repository:"
echo "1. Copy the repository to your web server"
echo "2. Add the GPG key: curl -fsSL https://your-server.com/apt-repo/gpg.key | sudo apt-key add -"
echo "3. Add to /etc/apt/sources.list.d/nkudo.list:"
echo "   deb https://your-server.com/apt-repo stable main"
echo "4. Run: sudo apt-get update && sudo apt-get install nkudo-edge"
