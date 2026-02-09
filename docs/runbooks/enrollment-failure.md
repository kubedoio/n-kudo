# Enrollment Failure Runbook

## Symptoms
- Agent fails to enroll with error messages
- HTTP 400/401/403 errors during enrollment
- TLS/certificate errors
- Token expired or invalid errors

## Diagnostic Steps

### 1. Check Enrollment Token
```bash
# Verify token format (should be base64-encoded)
echo "$ENROLLMENT_TOKEN" | base64 -d 2>/dev/null | head -c 100

# Check token hasn't expired
curl -s "https://$CONTROL_PLANE/tenants/$TENANT_ID/enrollment-tokens" \
  -H "X-API-Key: $API_KEY" | jq '.[] | select(.token_hash == "...")'
```

### 2. Verify Network Connectivity
```bash
# Check control-plane reachability
curl -v https://$CONTROL_PLANE/healthz

# Check TLS certificate
openssl s_client -connect $CONTROL_PLANE:443 -servername $CONTROL_PLANE </dev/null
```

### 3. Check Agent Logs
```bash
# View edge agent logs
journalctl -u nkudo-edge -n 100

# Run enrollment with verbose output
nkudo-edge enroll --control-plane https://$CONTROL_PLANE --token $TOKEN -v
```

### 4. Verify System Requirements
```bash
# Check disk space for PKI storage
df -h /var/lib/nkudo-edge

# Check permissions
ls -la /var/lib/nkudo-edge/

# Check hostname resolution
hostname -f
```

## Common Issues and Solutions

### Issue: "invalid or expired enrollment token"
**Cause**: Token is single-use and has been consumed, or has expired.

**Solution**:
1. Generate new token from dashboard/API
2. Check token TTL in tenant settings
3. Verify system clock is synchronized

### Issue: TLS certificate verification failed
**Cause**: Self-signed certificates in dev, or CA trust issues in production.

**Solution**:
```bash
# For development only (not production)
nkudo-edge enroll --insecure-skip-verify ...

# For production: provide CA certificate
nkudo-edge enroll --ca-file /path/to/ca.crt ...
```

### Issue: "failed to write PKI"
**Cause**: Permission denied or disk full.

**Solution**:
```bash
# Fix permissions
sudo mkdir -p /var/lib/nkudo-edge/pki
sudo chown -R root:root /var/lib/nkudo-edge
sudo chmod 700 /var/lib/nkudo-edge/pki

# Check disk space
df -h /var/lib/nkudo-edge
```

### Issue: "CSR generation failed"
**Cause**: Insufficient entropy or crypto libraries missing.

**Solution**:
```bash
# Check entropy
cat /proc/sys/kernel/random/entropy_avail

# Install haveged if needed (low entropy environments)
sudo apt-get install haveged
```

## Recovery Procedures

### Reset Agent and Re-enroll
```bash
# Stop agent
sudo systemctl stop nkudo-edge

# Backup and clear state
sudo mv /var/lib/nkudo-edge /var/lib/nkudo-edge.bak.$(date +%s)
sudo mkdir -p /var/lib/nkudo-edge/{state,pki,vms}

# Re-enroll with new token
sudo nkudo-edge enroll \
  --control-plane https://$CONTROL_PLANE \
  --token $NEW_TOKEN \
  --state-dir /var/lib/nkudo-edge/state \
  --pki-dir /var/lib/nkudo-edge/pki

# Start agent
sudo systemctl start nkudo-edge
```

### Verify Enrollment Success
```bash
# Check agent identity
sudo cat /var/lib/nkudo-edge/state/edge-state.json | jq '.identity'

# Check certificates
sudo openssl x509 -in /var/lib/nkudo-edge/pki/client.crt -text -noout | head -20

# Verify heartbeat
sudo nkudo-edge verify-heartbeat --once
```

## Prevention

1. **Token Management**: Generate tokens with sufficient TTL (15+ minutes)
2. **Clock Sync**: Ensure NTP is configured on all agents
3. **Monitoring**: Alert on enrollment failure rate > 2%
4. **Documentation**: Provide clear enrollment instructions to customers

## Escalation

If issue persists after following this runbook:
1. Collect agent logs: `journalctl -u nkudo-edge > edge-logs.txt`
2. Collect control-plane logs for the time period
3. Check audit events: `SELECT * FROM audit_events WHERE action = 'enroll' AND ...`
4. File incident with collected data
