# VM Lifecycle Issues Runbook

## Symptoms
- VM creation fails
- VM fails to start
- VM stuck in "PENDING" or "STARTING" state
- Cannot stop or delete VM
- VM has no network connectivity

## Diagnostic Steps

### 1. Check VM Status
```bash
# Check local VM state
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/state.json | jq

# Check Cloud Hypervisor process
ps aux | grep cloud-hypervisor | grep $VM_ID

# Check VM directory
sudo ls -la /var/lib/nkudo-edge/vms/$VM_ID/
```

### 2. Check Cloud Hypervisor Logs
```bash
# View console log
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/console.log

# View stderr/stdout
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/stderr.log
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/stdout.log

# View command log
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/commands.log
```

### 3. Check Cloud Hypervisor API
```bash
# Check if API socket exists
sudo ls -la /var/lib/nkudo-edge/vms/$VM_ID/api.sock

# Query VM status via CH API (if running)
sudo curl -s --unix-socket /var/lib/nkudo-edge/vms/$VM_ID/api.sock \
  http://localhost/api/v1/vm.info | jq
```

### 4. Check Resources
```bash
# Check disk space
df -h /var/lib/nkudo-edge

# Check memory
free -h

# Check CPU load
uptime

# Check KVM support
lsmod | grep kvm
ls -la /dev/kvm
```

### 5. Check Network Configuration
```bash
# Check TAP interface
ip link show | grep tap
ip link show $TAP_NAME

# Check bridge
ip link show br0
brctl show br0

# Check iptables rules
sudo iptables -t nat -L | grep $VM_ID
```

## Common Issues and Solutions

### Issue: "KVM not available"
**Cause**: KVM kernel modules not loaded or `/dev/kvm` permissions.

**Solution**:
```bash
# Load KVM modules
sudo modprobe kvm
sudo modprobe kvm_intel  # or kvm_amd

# Fix permissions
sudo chmod 666 /dev/kvm

# Add user to kvm group
sudo usermod -aG kvm $USER
```

### Issue: "cloud-hypervisor: command not found"
**Cause**: Cloud Hypervisor not installed or not in PATH.

**Solution**:
```bash
# Download and install CH
wget https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v38.0/cloud-hypervisor-static
chmod +x cloud-hypervisor-static
sudo mv cloud-hypervisor-static /usr/local/bin/cloud-hypervisor
```

### Issue: VM creation fails with "Failed to create disk"
**Cause**: Insufficient disk space or permissions.

**Solution**:
```bash
# Check disk space
df -h /var/lib/nkudo-edge/vms

# Check permissions
sudo chown -R root:root /var/lib/nkudo-edge/vms
sudo chmod 755 /var/lib/nkudo-edge/vms

# Clean up old VMs
sudo nkudo-edge apply --plan cleanup-plan.json
```

### Issue: VM starts but has no network
**Cause**: TAP interface or bridge configuration issue.

**Solution**:
```bash
# Create bridge if missing
sudo ip link add name br0 type bridge
sudo ip link set br0 up
sudo ip addr add 192.168.100.1/24 dev br0

# Enable IP forwarding
sudo sysctl -w net.ipv4.ip_forward=1

# Setup NAT
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
sudo iptables -A FORWARD -i br0 -o eth0 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o br0 -m state --state RELATED,ESTABLISHED -j ACCEPT
```

### Issue: "Failed to create cloud-init ISO"
**Cause**: `cloud-localds`, `genisoimage`, or `mkisofs` not found.

**Solution**:
```bash
# Install cloud-image-utils (includes cloud-localds)
sudo apt-get install cloud-image-utils

# Or install genisoimage
sudo apt-get install genisoimage
```

### Issue: VM stuck in "STOPPING" state
**Cause**: Cloud Hypervisor process not responding.

**Solution**:
```bash
# Find and kill CH process
sudo pkill -f "cloud-hypervisor.*$VM_ID"

# Clean up state
sudo rm -f /var/lib/nkudo-edge/vms/$VM_ID/ch.pid

# Update state file
sudo cat /var/lib/nkudo-edge/state/edge-state.json | \
  jq 'del(.microvms["'$VM_ID'"])' > /tmp/state.json && \
  sudo mv /tmp/state.json /var/lib/nkudo-edge/state/edge-state.json
```

## Recovery Procedures

### Force Delete Stuck VM
```bash
VM_ID="your-vm-id"

# Stop CH process
sudo pkill -f "cloud-hypervisor.*$VM_ID" || true

# Remove TAP interface
TAP_NAME=$(sudo cat /var/lib/nkudo-edge/vms/$VM_ID/state.json | jq -r '.tap_name')
sudo ip link del $TAP_NAME 2>/dev/null || true

# Remove VM directory
sudo rm -rf /var/lib/nkudo-edge/vms/$VM_ID

# Update state store
sudo cat /var/lib/nkudo-edge/state/edge-state.json | \
  jq 'del(.microvms["'$VM_ID'"])' > /tmp/state.json && \
  sudo mv /tmp/state.json /var/lib/nkudo-edge/state/edge-state.json

# Report deletion to control-plane
# (via heartbeat with execution update)
```

### Recreate VM from Scratch
```bash
# Delete old VM
# (follow force delete procedure above)

# Submit new plan via API
curl -X POST https://$CONTROL_PLANE/sites/$SITE_ID/plans \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "recreate-'$VM_ID'",
    "actions": [{
      "operation_id": "recreate-1",
      "operation": "CREATE",
      "vm_id": "'$VM_ID'",
      "name": "recovered-vm",
      "vcpu_count": 2,
      "memory_mib": 512
    }]
  }'
```

## Performance Tuning

### Optimize Cloud Hypervisor
```bash
# Use hugepages for better performance
echo 1024 | sudo tee /proc/sys/vm/nr_hugepages

# Enable KSM (Kernel Samepage Merging)
echo 1 | sudo tee /sys/kernel/mm/ksm/run

# CPU governor to performance
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### Resource Limits
```bash
# Check current ulimits
ulimit -a

# Increase limits for cloud-hypervisor
cat >> /etc/security/limits.conf << EOF
root soft nofile 65536
root hard nofile 65536
EOF
```

## Prevention

1. **Monitoring**: Alert on VM failure rate > 5%
2. **Resource Planning**: Ensure sufficient disk/memory for VM workloads
3. **Image Caching**: Pre-download base images to reduce creation time
4. **Health Checks**: Implement VM health probes

## Escalation

If issue persists:
1. Collect all logs: `/var/lib/nkudo-edge/vms/$VM_ID/*.log`
2. Collect agent state: `/var/lib/nkudo-edge/state/edge-state.json`
3. File incident with VM ID, plan ID, and execution ID
