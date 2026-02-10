#!/bin/bash
# Setup YUM repository structure
# - Create repo metadata
# - Sign packages
# - Generate repodata/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

REPO_DIR="${PROJECT_ROOT}/dist/yum-repo"
ARCHS="x86_64 aarch64"

# GPG key configuration
GPG_KEY_ID="${GPG_KEY_ID:-}"
GPG_PASSPHRASE="${GPG_PASSPHRASE:-}"

echo "Setting up YUM repository structure..."

# Create repository structure for each architecture
for arch in $ARCHS; do
    mkdir -p "${REPO_DIR}/${arch}"
done

# Copy .rpm packages
echo "Copying .rpm packages..."
if [ -d "${PROJECT_ROOT}/dist" ]; then
    for rpm in "${PROJECT_ROOT}/dist"/*.rpm; do
        if [ -f "$rpm" ]; then
            # Determine architecture from filename
            rpm_arch=$(rpm -qp --queryformat '%{ARCH}' "$rpm" 2>/dev/null || echo "")
            
            case "$rpm_arch" in
                x86_64)
                    cp "$rpm" "${REPO_DIR}/x86_64/"
                    ;;
                aarch64)
                    cp "$rpm" "${REPO_DIR}/aarch64/"
                    ;;
                noarch)
                    # Copy to all architectures
                    for arch in $ARCHS; do
                        cp "$rpm" "${REPO_DIR}/${arch}/"
                    done
                    ;;
                *)
                    echo "Warning: Unknown architecture for $rpm, skipping"
                    ;;
            esac
        fi
    done
fi

# Sign packages
sign_packages() {
    if [ -n "$GPG_KEY_ID" ]; then
        echo "Signing packages with GPG key ${GPG_KEY_ID}..."
        
        for arch in $ARCHS; do
            for rpm in "${REPO_DIR}/${arch}"/*.rpm; do
                if [ -f "$rpm" ] && [ ! -f "${rpm}.asc" ]; then
                    echo "Signing: $(basename "$rpm")"
                    
                    if [ -n "$GPG_PASSPHRASE" ]; then
                        # Use expect for automated signing
                        rpm --define "_gpg_name ${GPG_KEY_ID}" \
                            --define "_signature gpg" \
                            --addsign "$rpm" << EOF
${GPG_PASSPHRASE}
EOF
                    else
                        rpm --define "_gpg_name ${GPG_KEY_ID}" \
                            --define "_signature gpg" \
                            --addsign "$rpm"
                    fi
                fi
            done
        done
        
        # Export public key
        gpg --armor --export "$GPG_KEY_ID" > "${REPO_DIR}/RPM-GPG-KEY-nkudo"
    else
        echo "Warning: GPG_KEY_ID not set. Packages not signed."
        echo "Set GPG_KEY_ID and optionally GPG_PASSPHRASE environment variables to sign."
    fi
}

sign_packages

# Generate repository metadata
generate_metadata() {
    echo "Generating repository metadata..."
    
    for arch in $ARCHS; do
        if [ -d "${REPO_DIR}/${arch}" ] && [ "$(ls -A "${REPO_DIR}/${arch}"/*.rpm 2>/dev/null)" ]; then
            echo "Creating repodata for ${arch}..."
            
            if command -v createrepo &> /dev/null; then
                createrepo "${REPO_DIR}/${arch}"
            elif command -v createrepo_c &> /dev/null; then
                createrepo_c "${REPO_DIR}/${arch}"
            else
                echo "Warning: createrepo not found. Skipping metadata generation."
                echo "Install with: sudo dnf install createrepo  or  sudo yum install createrepo"
                return
            fi
            
            # Sign repodata if GPG key is available
            if [ -n "$GPG_KEY_ID" ]; then
                if [ -f "${REPO_DIR}/${arch}/repodata/repomd.xml" ]; then
                    gpg --detach-sign --armor \
                        -u "$GPG_KEY_ID" \
                        -o "${REPO_DIR}/${arch}/repodata/repomd.xml.asc" \
                        "${REPO_DIR}/${arch}/repodata/repomd.xml"
                fi
            fi
        fi
    done
}

generate_metadata

# Create .repo file
create_repo_file() {
    cat > "${REPO_DIR}/nkudo.repo" << 'EOF'
[nkudo]
name=n-kudo Edge Agent Repository
baseurl=https://your-server.com/yum-repo/$basearch
enabled=1
gpgcheck=1
gpgkey=https://your-server.com/yum-repo/RPM-GPG-KEY-nkudo
repo_gpgcheck=1
metadata_expire=300

[nkudo-source]
name=n-kudo Edge Agent Repository - Source
baseurl=https://your-server.com/yum-repo/source
enabled=0
gpgcheck=1
gpgkey=https://your-server.com/yum-repo/RPM-GPG-KEY-nkudo
EOF
}

create_repo_file

echo ""
echo "YUM repository created at: ${REPO_DIR}"
echo ""
echo "Repository structure:"
for arch in $ARCHS; do
    if [ -d "${REPO_DIR}/${arch}" ]; then
        echo "  ${arch}/"
        echo "    Packages: $(ls "${REPO_DIR}/${arch}"/*.rpm 2>/dev/null | wc -l)"
    fi
done
echo ""
echo "To use this repository:"
echo "1. Copy the repository to your web server"
echo "2. Import GPG key: sudo rpm --import https://your-server.com/yum-repo/RPM-GPG-KEY-nkudo"
echo "3. Create /etc/yum.repos.d/nkudo.repo with:"
cat "${REPO_DIR}/nkudo.repo"
echo "4. Run: sudo dnf install nkudo-edge  or  sudo yum install nkudo-edge"
