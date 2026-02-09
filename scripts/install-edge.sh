#!/usr/bin/env bash
set -euo pipefail

# n-kudo edge agent installer script
# Usage: curl -sSL https://get.nkudo.io | sudo bash

REPO="kubedoio/n-kudo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
CONFIG_DIR="${CONFIG_DIR:-/etc/nkudo}"
DATA_DIR="${DATA_DIR:-/var/lib/nkudo-edge}"

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        *)
            echo "Unsupported OS: $os" >&2
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install binary
install_binary() {
    local version="${1:-latest}"
    local os
    local arch
    local binary_name
    local download_url
    local tmp_dir
    
    os=$(detect_os)
    arch=$(detect_arch)
    binary_name="nkudo-edge-${os}-${arch}"
    
    if [ "$version" = "latest" ]; then
        version=$(get_latest_version)
    fi
    
    echo "Installing n-kudo edge agent ${version} for ${os}/${arch}..."
    
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}.tar.gz"
    
    echo "Downloading from ${download_url}..."
    curl -sSL -o "$tmp_dir/${binary_name}.tar.gz" "$download_url"
    
    echo "Extracting..."
    tar -xzf "$tmp_dir/${binary_name}.tar.gz" -C "$tmp_dir"
    
    echo "Installing binary to ${INSTALL_DIR}..."
    install -m 755 "$tmp_dir/${binary_name}" "${INSTALL_DIR}/nkudo-edge"
    
    echo "Binary installed successfully!"
}

# Create systemd service
install_systemd_service() {
    echo "Creating systemd service..."
    
    mkdir -p "$DATA_DIR"/{state,pki,vms}
    
    cat > "${SYSTEMD_DIR}/nkudo-edge.service" << 'EOF'
[Unit]
Description=n-kudo Edge Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nkudo-edge run --config /etc/nkudo/edge.conf
Restart=always
RestartSec=10
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    echo "Systemd service installed."
    echo "Start with: sudo systemctl start nkudo-edge"
    echo "Enable auto-start: sudo systemctl enable nkudo-edge"
}

# Create default config
create_config() {
    echo "Creating default configuration..."
    mkdir -p "$CONFIG_DIR"
    
    cat > "${CONFIG_DIR}/edge.conf" << EOF
# n-kudo Edge Agent Configuration
CONTROL_PLANE_URL=https://api.nkudo.io
STATE_DIR=${DATA_DIR}/state
PKI_DIR=${DATA_DIR}/pki
RUNTIME_DIR=${DATA_DIR}/vms
HEARTBEAT_INTERVAL=15s
EOF
    
    echo "Configuration created at ${CONFIG_DIR}/edge.conf"
    echo "Please edit it with your control-plane URL and enrollment token."
}

# Main installation
main() {
    local version="${1:-latest}"
    
    echo "========================================"
    echo "n-kudo Edge Agent Installer"
    echo "========================================"
    
    # Check for root
    if [ "$EUID" -ne 0 ]; then
        echo "Please run as root (use sudo)"
        exit 1
    fi
    
    install_binary "$version"
    install_systemd_service
    create_config
    
    echo ""
    echo "========================================"
    echo "Installation complete!"
    echo "========================================"
    echo ""
    echo "Next steps:"
    echo "1. Edit ${CONFIG_DIR}/edge.conf with your settings"
    echo "2. Obtain an enrollment token from your control-plane"
    echo "3. Run: nkudo-edge enroll --token <your-token>"
    echo "4. Start the service: systemctl start nkudo-edge"
    echo ""
    echo "Check status: systemctl status nkudo-edge"
    echo "View logs: journalctl -u nkudo-edge -f"
}

main "$@"
