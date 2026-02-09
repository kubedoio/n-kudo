# Agent Troubleshooting Runbook

Comprehensive troubleshooting guide for n-kudo edge agent issues.

## Overview

This runbook covers common agent issues including:
- Enrollment problems
- Certificate expiry and renewal
- Heartbeat failures
- VM lifecycle issues
- Network connectivity problems

## Quick Diagnostics

### Check Agent Status

```bash
# Check if agent is running
sudo systemctl status nkudo-edge

# View recent logs
sudo journalctl -u nkudo-edge -n 100

# Follow logs in real-time
sudo journalctl -u nkudo-edge -f

# Check agent state
sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq

# Verify certificates
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -dates
```

---

## 1. Enrollment Issues

### Symptoms
- Agent fails to enroll
- "invalid or expired enrollment token" error
- TLS/certificate errors during enrollment
- "failed to write PKI" error

### Diagnostic Steps

```bash
# 1. Verify enrollment token
echo "$NKUDO_ENROLL_TOKEN" | base64 -d 2>/dev/null | head -c 100

# 2. Check token validity via API
curl -s -H "X-Admin-Key: $ADMIN_KEY" \
  https://$CONTROL_PLANE/tenants/$TENANT_ID/enrollment-tokens | jq

# 3. Test control-plane connectivity
curl -v https://$CONTROL_PLANE/healthz

# 4. Check disk space for PKI storage
df -h /var/lib/nkudo-edge

# 5. Verify directory permissions
ls -la /var/lib/nkudo-edge/
ls -la /var/lib/nkudo-edge/pki/
```

### Common Issues and Solutions

#### Invalid or Expired Token

```bash
# Generate new token
curl -X POST https://$CONTROL_PLANE/tenants/$TENANT_ID/enrollment-tokens \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"ttl_minutes": 30}'

# Re-enroll with new token
sudo nkudo-edge enroll \
  --control-plane https://$CONTROL_PLANE \
  --token $NEW_TOKEN \
  --state-dir /var/lib/nkudo-edge/state \
  --pki-dir /var/lib/nkudo-edge/pki
```

#### TLS Certificate Verification Failed

```bash
# For development only - skip verification
sudo nkudo-edge enroll --insecure-skip-verify ...

# For production - provide CA certificate
sudo nkudo-edge enroll \
  --control-plane https://$CONTROL_PLANE \
  --token $TOKEN \
  --ca-file /etc/nkudo/ca.crt
```

#### Permission Denied

```bash
# Fix permissions
sudo mkdir -p /var/lib/nkudo-edge/{state,pki,vms}
sudo chown -R root:root /var/lib/nkudo-edge
sudo chmod 700 /var/lib/nkudo-edge/pki
sudo chmod 755 /var/lib/nkudo-edge/state
sudo chmod 755 /var/lib/nkudo-edge/vms
```

---

## 2. Certificate Expiry and Renewal

### Symptoms
- Agent shows "certificate expired" errors
- mTLS connection failures
- Agent appears offline in dashboard
- "unable to authenticate" errors

### Diagnostic Steps

```bash
# Check certificate expiry
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -dates

# Check certificate subject and issuer
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -text | head -20

# Verify certificate against CA
sudo openssl verify -CAfile /var/lib/nkudo-edge/pki/ca.crt \
  /var/lib/nkudo-edge/pki/client.crt

# Check refresh token validity
sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq '.identity.refresh_token'
```

### Manual Certificate Renewal

```bash
# Stop agent
sudo systemctl stop nkudo-edge

# Backup current PKI
sudo cp -r /var/lib/nkudo-edge/pki /var/lib/nkudo-edge/pki.bak.$(date +%s)

# Generate new CSR
sudo openssl req -new -key /var/lib/nkudo-edge/pki/client.key \
  -out /tmp/client.csr \
  -subj "/CN=$(hostname -f)/O=n-kudo-agent"

# Submit CSR for signing (requires API access)
curl -X POST https://$CONTROL_PLANE/v1/refresh \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq -r '.identity.refresh_token')" \
  --data-binary @/tmp/client.csr \
  -o /tmp/client.crt

# Verify new certificate
sudo openssl x509 -in /tmp/client.crt -noout -dates

# Install new certificate
sudo mv /tmp/client.crt /var/lib/nkudo-edge/pki/client.crt
sudo chmod 600 /var/lib/nkudo-edge/pki/client.crt

# Start agent
sudo systemctl start nkudo-edge
```

### Force Re-enrollment

```bash
# Stop agent
sudo systemctl stop nkudo-edge

# Backup and clear state
sudo mv /var/lib/nkudo-edge /var/lib/nkudo-edge.bak.$(date +%s)
sudo mkdir -p /var/lib/nkudo-edge/{state,pki,vms}

# Generate new enrollment token from dashboard/API
# Then re-enroll
sudo nkudo-edge enroll \
  --control-plane https://$CONTROL_PLANE \
  --token $NEW_TOKEN \
  --state-dir /var/lib/nkudo-edge/state \
  --pki-dir /var/lib/nkudo-edge/pki

# Start agent
sudo systemctl start nkudo-edge
```

---

## 3. Heartbeat Failures

### Symptoms
- Agent shows as offline in dashboard
- "heartbeat failed" errors in logs
- No recent host facts updates
- Plans not being received by agent

### Diagnostic Steps

```bash
# Check network connectivity
curl -v --cacert /var/lib/nkudo-edge/pki/ca.crt \
  --cert /var/lib/nkudo-edge/pki/client.crt \
  --key /var/lib/nkudo-edge/pki/client.key \
  https://$CONTROL_PLANE/healthz

# Check agent identity
sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq '.identity'

# Verify heartbeat manually
sudo nkudo-edge verify-heartbeat --once

# Check system time (must be synchronized)
timedatectl status

# Check for firewall blocking connections
sudo iptables -L -n | grep 8443
sudo ss -tlnp | grep 8443
```

### Common Issues and Solutions

#### Clock Skew

```bash
# Check time synchronization
timedatectl status

# Force time sync
sudo systemctl restart systemd-timesyncd
sudo timedatectl set-ntp true

# Or use chrony
sudo chronyc tracking
sudo chronyc makestep
```

#### Network Connectivity

```bash
# Test connection with mTLS
curl -v --cacert /var/lib/nkudo-edge/pki/ca.crt \
  --cert /var/lib/nkudo-edge/pki/client.crt \
  --key /var/lib/nkudo-edge/pki/client.key \
  https://$CONTROL_PLANE/agents/heartbeat \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "test", "timestamp": "'$(date -Iseconds)'"}'

# Check DNS resolution
nslookup $(echo $CONTROL_PLANE | sed 's|https://||')

# Trace route
traceroute $(echo $CONTROL_PLANE | sed 's|https://||')
```

#### Certificate Issues

```bash
# Verify certificate is not expired
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -checkend 0

# Check if certificate is valid for 24h
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -checkend 86400

# If expired, see Certificate Renewal section above
```

#### Firewall/Proxy Issues

```bash
# Check if proxy is configured
echo $HTTP_PROXY $HTTPS_PROXY

# Test direct connection
sudo nkudo-edge verify-heartbeat --once --control-plane-direct

# Configure proxy for agent (if needed)
# Edit /etc/systemd/system/nkudo-edge.service.d/proxy.conf
[Service]
Environment="HTTP_PROXY=http://proxy:8080"
Environment="HTTPS_PROXY=http://proxy:8080"
Environment="NO_PROXY=localhost,127.0.0.1"

sudo systemctl daemon-reload
sudo systemctl restart nkudo-edge
```

---

## 4. VM Lifecycle Issues

### Symptoms
- VM creation/start/stop/delete fails
- VMs stuck in pending state
- Cloud Hypervisor errors
- No network connectivity in VMs

### Diagnostic Steps

```bash
# Check VM state
sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq '.microvms'

# Check Cloud Hypervisor logs
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/console.log
sudo cat /var/lib/nkudo-edge/vms/$VM_ID/stderr.log

# Check CH process
ps aux | grep cloud-hypervisor

# Check KVM support
lsmod | grep kvm
ls -la /dev/kvm

# Check resources
free -h
df -h /var/lib/nkudo-edge
```

### Common Issues

See [VM Lifecycle Issues Runbook](./vm-lifecycle-issues.md) for detailed troubleshooting.

Quick fixes:

```bash
# Fix KVM permissions
sudo chmod 666 /dev/kvm
sudo usermod -aG kvm $USER

# Restart stuck VM
sudo pkill -f "cloud-hypervisor.*$VM_ID"
sudo rm -rf /var/lib/nkudo-edge/vms/$VM_ID

# Clear VM state
sudo cat /var/lib/nkudo-edge/state/edge-state.json | \
  jq 'del(.microvms["'$VM_ID'"])' > /tmp/state.json && \
  sudo mv /tmp/state.json /var/lib/nkudo-edge/state/edge-state.json
```

---

## 5. NetBird Connectivity Issues

### Symptoms
- NetBird status shows disconnected
- Cannot reach VMs via NetBird network
- NetBird auto-join fails

### Diagnostic Steps

```bash
# Check NetBird status
sudo netbird status

# Check NetBird service
sudo systemctl status netbird

# Check NetBird logs
sudo journalctl -u netbird -n 100

# Verify NetBird interface
ip addr show wt0

# Check routing
ip route | grep netbird
```

See [NetBird Troubleshooting Runbook](./netbird-troubleshooting.md) for detailed steps.

---

## 6. General System Issues

### Disk Space Issues

```bash
# Check disk usage
df -h /var/lib/nkudo-edge

# Find large files
sudo du -h /var/lib/nkudo-edge | sort -rh | head -20

# Clean up old logs
sudo find /var/log -name "*.log.*" -mtime +7 -delete

# Clean up old VM logs
sudo find /var/lib/nkudo-edge/vms -name "*.log" -mtime +7 -delete
```

### Memory Issues

```bash
# Check memory usage
free -h
vmstat 1 5

# Check for OOM kills
sudo dmesg | grep -i "out of memory"
sudo journalctl -k | grep -i "oom"

# Check agent memory usage
ps aux | grep nkudo-edge
sudo cat /proc/$(pgrep nkudo-edge)/status | grep -E 'VmRSS|VmSize'
```

### High CPU Usage

```bash
# Check CPU usage
top -p $(pgrep nkudo-edge)

# Profile agent (if debug build available)
sudo perf top -p $(pgrep nkudo-edge)

# Check for busy loops in logs
sudo journalctl -u nkudo-edge | grep -i "retry\|loop\|timeout"
```

---

## Recovery Procedures

### Complete Agent Reset

```bash
# 1. Stop agent
sudo systemctl stop nkudo-edge

# 2. Backup state
sudo tar czf /opt/nkudo-edge-backup-$(date +%Y%m%d).tar.gz /var/lib/nkudo-edge

# 3. Clear state directories
sudo rm -rf /var/lib/nkudo-edge/state/*
sudo rm -rf /var/lib/nkudo-edge/pki/*

# 4. Generate new enrollment token from dashboard

# 5. Re-enroll
sudo nkudo-edge enroll \
  --control-plane https://$CONTROL_PLANE \
  --token $NEW_TOKEN

# 6. Start agent
sudo systemctl start nkudo-edge

# 7. Verify
sudo nkudo-edge verify-heartbeat --once
```

### Disaster Recovery

```bash
# Restore from backup
sudo systemctl stop nkudo-edge
sudo rm -rf /var/lib/nkudo-edge
sudo tar xzf /opt/nkudo-edge-backup-YYYYMMDD.tar.gz -C /
sudo systemctl start nkudo-edge

# Verify certificates are still valid
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -checkend 0

# If certificates expired, force re-enrollment (see above)
```

---

## Debugging and Logs

### Enable Debug Logging

```bash
# Edit service file
sudo systemctl edit nkudo-edge

# Add:
[Service]
Environment="LOG_LEVEL=debug"

# Restart
sudo systemctl daemon-reload
sudo systemctl restart nkudo-edge

# View debug logs
sudo journalctl -u nkudo-edge -f
```

### Collect Diagnostic Information

```bash
#!/bin/bash
# collect-diagnostics.sh

OUTPUT_DIR="/tmp/nkudo-diagnostics-$(date +%Y%m%d-%H%M%S)"
mkdir -p $OUTPUT_DIR

# System info
uname -a > $OUTPUT_DIR/system.txt
cat /proc/cpuinfo > $OUTPUT_DIR/cpuinfo.txt
free -h > $OUTPUT_DIR/memory.txt
df -h > $OUTPUT_DIR/disk.txt

# Service status
sudo systemctl status nkudo-edge > $OUTPUT_DIR/service-status.txt

# Logs
sudo journalctl -u nkudo-edge --since "24 hours ago" > $OUTPUT_DIR/logs.txt

# State (redacted)
sudo cat /var/lib/nkudo-edge/state/edge-state.json | \
  jq '.identity.refresh_token = "REDACTED"' > $OUTPUT_DIR/state.json

# Certificates (metadata only)
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -text > $OUTPUT_DIR/cert-info.txt

# Network
ip addr > $OUTPUT_DIR/network.txt
ip route >> $OUTPUT_DIR/network.txt

# Package into tarball
tar czf $OUTPUT_DIR.tar.gz -C /tmp $(basename $OUTPUT_DIR)
echo "Diagnostics collected: $OUTPUT_DIR.tar.gz"
```

---

## Prevention

### Monitoring Setup

```bash
# Monitor agent status
#!/bin/bash
# /usr/local/bin/check-nkudo-agent.sh

if ! systemctl is-active --quiet nkudo-edge; then
  echo "ALERT: nkudo-edge is not running"
  # Send alert via your monitoring system
fi

# Check certificate expiry (alert 7 days before)
EXPIRY=$(sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
NOW_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

if [ $DAYS_UNTIL_EXPIRY -lt 7 ]; then
  echo "ALERT: Certificate expires in $DAYS_UNTIL_EXPIRY days"
fi
```

Add to crontab:
```bash
*/5 * * * * /usr/local/bin/check-nkudo-agent.sh
```

### Best Practices

1. **Regular Updates**: Keep agent binary updated
2. **Certificate Monitoring**: Monitor certificate expiry
3. **Log Rotation**: Configure logrotate for agent logs
4. **Backup**: Regular backups of `/var/lib/nkudo-edge`
5. **Resource Monitoring**: Monitor disk, memory, CPU usage
6. **Network Monitoring**: Alert on connectivity issues

---

## Escalation

If issue persists after following this runbook:

1. Collect diagnostics: `sudo ./collect-diagnostics.sh`
2. Gather control-plane logs for the time period
3. Document steps taken and results
4. File incident with:
   - Agent ID and Site ID
   - Diagnostic tarball
   - Control-plane logs
   - Timeline of events
