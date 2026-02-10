# Package Installation

This document describes how to install the n-kudo edge agent using package managers (APT/YUM) or manual package installation.

## Table of Contents

- [APT Repository (Debian/Ubuntu)](#apt-repository-debianubuntu)
- [YUM Repository (RHEL/CentOS/Fedora)](#yum-repository-rhelcentosfedora)
- [Manual Package Installation](#manual-package-installation)
- [Post-Installation](#post-installation)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)

## APT Repository (Debian/Ubuntu)

### Quick Install

```bash
# Add the GPG key
curl -fsSL https://packages.nkudo.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/nkudo-archive-keyring.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/nkudo-archive-keyring.gpg] https://packages.nkudo.io/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/nkudo.list

# Update package list and install
sudo apt-get update
sudo apt-get install nkudo-edge
```

### Supported Versions

| Distribution | Versions |
|--------------|----------|
| Ubuntu | 20.04 (Focal), 22.04 (Jammy), 24.04 (Noble) |
| Debian | 11 (Bullseye), 12 (Bookworm) |

### Supported Architectures

- `amd64` (x86_64)
- `arm64` (aarch64)

### Manual .deb Installation

If you prefer not to use the repository, you can download and install the .deb package directly:

```bash
# Download the package (replace VERSION and ARCH as needed)
VERSION="v0.1.0"
ARCH="amd64"
curl -LO "https://github.com/kubedoio/n-kudo/releases/download/${VERSION}/nkudo-edge_${VERSION}_${ARCH}.deb"

# Install the package
sudo dpkg -i "nkudo-edge_${VERSION}_${ARCH}.deb"

# Fix any missing dependencies
sudo apt-get install -f
```

## YUM Repository (RHEL/CentOS/Fedora)

### Quick Install

```bash
# Add the repository
sudo tee /etc/yum.repos.d/nkudo.repo << 'EOF'
[nkudo]
name=n-kudo Edge Agent Repository
baseurl=https://packages.nkudo.io/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.nkudo.io/gpg.key
repo_gpgcheck=1
metadata_expire=300
EOF

# Import GPG key
sudo rpm --import https://packages.nkudo.io/gpg.key

# Install the package
sudo dnf install nkudo-edge
# or for older systems:
# sudo yum install nkudo-edge
```

### Supported Versions

| Distribution | Versions |
|--------------|----------|
| RHEL | 8, 9 |
| CentOS Stream | 8, 9 |
| Fedora | 39, 40 |
| Rocky Linux | 8, 9 |
| AlmaLinux | 8, 9 |

### Supported Architectures

- `x86_64` (amd64)
- `aarch64` (arm64)

### Manual .rpm Installation

If you prefer not to use the repository, you can download and install the .rpm package directly:

```bash
# Download the package (replace VERSION and ARCH as needed)
VERSION="v0.1.0"
ARCH="x86_64"
curl -LO "https://github.com/kubedoio/n-kudo/releases/download/${VERSION}/nkudo-edge-${VERSION}-1.${ARCH}.rpm"

# Install the package
sudo rpm -i "nkudo-edge-${VERSION}-1.${ARCH}.rpm"
```

## Manual Package Installation

### From GitHub Releases

1. Visit the [GitHub Releases](https://github.com/kubedoio/n-kudo/releases) page
2. Download the appropriate package for your system:
   - Debian/Ubuntu: `.deb` file
   - RHEL/CentOS/Fedora: `.rpm` file
3. Install using the commands above

### Verification

You can verify the package integrity using the provided SHA256 checksums:

```bash
# Download the checksum file
curl -LO "https://github.com/kubedoio/n-kudo/releases/download/${VERSION}/nkudo-edge_${VERSION}_${ARCH}.deb.sha256"

# Verify the package
sha256sum -c "nkudo-edge_${VERSION}_${ARCH}.deb.sha256"
```

## Post-Installation

### Configuration

After installation, you need to configure the edge agent:

1. Edit the configuration file:
   ```bash
   sudo nano /etc/nkudo-edge/nkudo-edge.env
   ```

2. Set your control plane URL:
   ```bash
   CONTROL_PLANE_URL=https://your-control-plane.example.com
   ```

3. Start and enable the service:
   ```bash
   sudo systemctl start nkudo-edge
   sudo systemctl enable nkudo-edge
   ```

### Enrollment

Before the edge agent can connect to the control plane, you need to enroll it:

1. Obtain an enrollment token from your control plane
2. Run the enrollment command:
   ```bash
   sudo nkudo-edge enroll --token <your-enrollment-token>
   ```

3. Restart the service:
   ```bash
   sudo systemctl restart nkudo-edge
   ```

### Service Management

```bash
# Check service status
sudo systemctl status nkudo-edge

# View logs
sudo journalctl -u nkudo-edge -f

# Stop the service
sudo systemctl stop nkudo-edge

# Start the service
sudo systemctl start nkudo-edge

# Restart the service
sudo systemctl restart nkudo-edge
```

### File Locations

| File/Directory | Description |
|----------------|-------------|
| `/usr/local/bin/nkudo-edge` | Binary executable |
| `/etc/systemd/system/nkudo-edge.service` | Systemd service file |
| `/etc/nkudo-edge/nkudo-edge.env` | Environment configuration |
| `/var/lib/nkudo-edge/` | Data directory |
| `/var/lib/nkudo-edge/state/` | State storage |
| `/var/lib/nkudo-edge/pki/` | PKI certificates |
| `/var/lib/nkudo-edge/vms/` | VM data |

## Upgrading

### APT Upgrade

```bash
sudo apt-get update
sudo apt-get upgrade nkudo-edge
```

### YUM/DNF Upgrade

```bash
# For dnf-based systems
sudo dnf upgrade nkudo-edge

# For yum-based systems
sudo yum upgrade nkudo-edge
```

The service will be automatically restarted after upgrade.

## Uninstallation

### APT Remove

```bash
# Remove the package (keeps configuration and data)
sudo apt-get remove nkudo-edge

# Remove the package and purge all data
sudo apt-get purge nkudo-edge

# Remove the repository
echo "Remove /etc/apt/sources.list.d/nkudo.list manually if desired"
```

### YUM/DNF Remove

```bash
# Remove the package
sudo dnf remove nkudo-edge
# or
sudo yum remove nkudo-edge

# Remove the repository
sudo rm /etc/yum.repos.d/nkudo.repo
```

### Data Cleanup

After removal, you may want to clean up remaining data:

```bash
# Remove data directory
sudo rm -rf /var/lib/nkudo-edge

# Remove configuration directory
sudo rm -rf /etc/nkudo-edge
```

## Troubleshooting

### Service fails to start

1. Check the logs:
   ```bash
   sudo journalctl -u nkudo-edge -n 100
   ```

2. Verify configuration:
   ```bash
   cat /etc/nkudo-edge/nkudo-edge.env
   ```

3. Check binary:
   ```bash
   /usr/local/bin/nkudo-edge --version
   ```

### Permission issues

The edge agent runs as the `nkudo-edge` user. Ensure proper permissions:

```bash
sudo chown -R nkudo-edge:nkudo-edge /var/lib/nkudo-edge
sudo chmod 750 /var/lib/nkudo-edge
```

### Network connectivity

Ensure the edge agent can reach the control plane:

```bash
# From the configuration file
source /etc/nkudo-edge/nkudo-edge.env
curl -v "${CONTROL_PLANE_URL}/health"
```

## Building Packages Locally

If you need to build packages from source:

```bash
# Build DEB package
./deployments/packaging/deb/build-deb.sh v0.1.0 amd64

# Build RPM package (requires rpmbuild)
./deployments/packaging/rpm/build-rpm.sh v0.1.0 amd64
```

For more information on local builds, see the [Development Guide](../development/building.md).
