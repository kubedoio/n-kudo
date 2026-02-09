# NetBird Troubleshooting Runbook

## Symptoms
- Agent shows "NetBird disconnected" in heartbeat
- Cannot reach VMs via NetBird mesh
- NetBird status probe failures
- VMs have no network connectivity

## Diagnostic Steps

### 1. Check NetBird Service Status
```bash
# Check if netbird service is running
sudo systemctl status netbird

# Check netbird process
ps aux | grep netbird

# Check netbird logs
sudo journalctl -u netbird -n 100
```

### 2. Verify NetBird CLI
```bash
# Check netbird version
netbird version

# Check netbird status
netbird status

# Check netbird peers
netbird status --detail
```

### 3. Check Network Connectivity
```bash
# Test control plane connectivity
ping -c 3 api.netbird.io

# Check for firewall rules
sudo iptables -L | grep netbird
sudo iptables -L -t nat | grep netbird

# Check routing table
ip route | grep netbird
```

### 4. Check Agent NetBird Configuration
```bash
# Check agent config
cat /etc/nkudo/edge.conf | grep netbird

# Check agent logs for NetBird events
journalctl -u nkudo-edge | grep -i netbird
```

## Common Issues and Solutions

### Issue: NetBird service not running
**Cause**: Service not enabled or crashed.

**Solution**:
```bash
# Start and enable service
sudo systemctl start netbird
sudo systemctl enable netbird

# Check for crashes
sudo journalctl -u netbird --since "1 hour ago"
```

### Issue: "netbird: command not found"
**Cause**: NetBird not installed on host.

**Solution**:
```bash
# Install NetBird
curl -fsSL https://pkgs.netbird.io/install.sh | sh

# Or use agent auto-install feature
nkudo-edge run --netbird-auto-join --netbird-install-cmd "curl -fsSL https://pkgs.netbird.io/install.sh | sh"
```

### Issue: Setup key invalid or expired
**Cause**: Setup key revoked or expired in NetBird management.

**Solution**:
1. Generate new setup key from NetBird dashboard
2. Update agent configuration
3. Re-enroll or restart agent with new key

### Issue: Cannot reach VMs via NetBird IP
**Cause**: Firewall blocking WireGuard port, or routing issue.

**Solution**:
```bash
# Check WireGuard port (default 51820)
sudo ss -tunlp | grep 51820

# Check if port is allowed
sudo iptables -L INPUT -v -n | grep 51820

# Allow WireGuard through firewall
sudo iptables -I INPUT -p udp --dport 51820 -j ACCEPT
sudo iptables -I INPUT -i wt0 -j ACCEPT
```

### Issue: NetBird probe failures
**Cause**: Control plane not accessible via NetBird mesh.

**Solution**:
```bash
# Test with different probe type
nkudo-edge run --netbird-probe-type ping --netbird-probe-target 100.64.0.1

# Disable probe if needed (not recommended for production)
nkudo-edge run --netbird-probe-type none
```

## Advanced Diagnostics

### Check WireGuard Interface
```bash
# List WireGuard interfaces
sudo wg show

# Show interface details
ip addr show wt0
ip link show wt0
```

### Check NetBird Routes
```bash
# Show NetBird routes
netbird status --detail | grep -A5 "Routes"

# Check IP forwarding
sysctl net.ipv4.ip_forward
sysctl net.ipv6.conf.all.forwarding
```

### DNS Resolution
```bash
# Check if NetBird DNS is configured
cat /etc/resolv.conf | grep netbird

# Test internal DNS resolution
nslookup <peer-name>.netbird.selfhosted
```

## Recovery Procedures

### Reset NetBird and Re-join
```bash
# Stop services
sudo systemctl stop netbird
sudo systemctl stop nkudo-edge

# Clear NetBird state
sudo rm -rf /var/lib/netbird/

# Restart services
sudo systemctl start netbird
sudo systemctl start nkudo-edge

# Or manually re-join
netbird up --setup-key <new-setup-key>
```

### Force NetBird Re-evaluation
```bash
# Restart agent to trigger fresh NetBird evaluation
sudo systemctl restart nkudo-edge

# Check new status
journalctl -u nkudo-edge -f | grep -i netbird
```

## Monitoring Checklist

- [ ] NetBird service running: `systemctl is-active netbird`
- [ ] Connected to control plane: `netbird status | grep Connected`
- [ ] Has IP address: `ip addr show wt0 | grep inet`
- [ ] Can reach peers: `ping -c 3 <peer-netbird-ip>`
- [ ] Agent reports connected: Check heartbeat in dashboard

## Related Runbooks

- [Enrollment Failure](./enrollment-failure.md)
- [VM Network Issues](./vm-network-issues.md)

## External Resources

- [NetBird Documentation](https://docs.netbird.io/)
- [WireGuard Troubleshooting](https://www.wireguard.com/netns/)
- NetBird Support: support@netbird.io
