# Phase 4 Task 1: Certificate Rotation

## Task Description
Implement automatic certificate rotation for edge agents before expiry.

## Background
Currently, agents get certificates with 24h TTL (configurable via `AGENT_CERT_TTL`). When certificates expire, agents must re-enroll, which is disruptive. Automatic rotation should happen seamlessly before expiry.

## Requirements

### 1. Automatic Rotation Before Expiry

**Rotation Trigger:**
- When certificate has <20% of lifetime remaining
- Or when <6 hours until expiry (whichever is longer)
- Example: For 24h cert, rotate at ~4.8h remaining (20%)

**Rotation Process:**
```
1. Check certificate expiry
2. If nearing expiry:
   a. Generate new CSR
   b. Call /v1/renew with refresh token
   c. Store new certificate
   d. Test new cert with heartbeat
   e. If successful, use new cert
   f. If failed, retry with backoff
3. If rotation fails, alert and continue with old cert
```

### 2. Implementation

**File:** `internal/edge/mtls/rotation.go`

```go
package mtls

type CertRotator struct {
    state      *state.Store
    client     *enroll.Client
    threshold  time.Duration  // 20% of TTL or 6h
    checkInterval time.Duration  // Check every 15 minutes
}

func NewRotator(store *state.Store, client *enroll.Client) *CertRotator {
    return &CertRotator{
        state:         store,
        client:        client,
        threshold:     6 * time.Hour,  // Minimum threshold
        checkInterval: 15 * time.Minute,
    }
}

func (r *CertRotator) Start(ctx context.Context) {
    ticker := time.NewTicker(r.checkInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := r.checkAndRotate(ctx); err != nil {
                log.Printf("certificate rotation failed: %v", err)
            }
        }
    }
}

func (r *CertRotator) checkAndRotate(ctx context.Context) error {
    identity, err := r.state.LoadIdentity()
    if err != nil {
        return err
    }
    
    // Load current certificate
    certPath := filepath.Join(r.state.GetPKIDir(), "client-cert.pem")
    certPEM, err := os.ReadFile(certPath)
    if err != nil {
        return err
    }
    
    // Parse certificate
    block, _ := pem.Decode(certPEM)
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return err
    }
    
    // Check if rotation needed
    remaining := time.Until(cert.NotAfter)
    ttl := cert.NotAfter.Sub(cert.NotBefore)
    threshold := max(r.threshold, ttl*20/100)  // 20% or 6h
    
    if remaining > threshold {
        return nil  // No rotation needed
    }
    
    log.Printf("Certificate expires in %v, rotating...", remaining)
    
    // Generate new CSR
    keyPath := filepath.Join(r.state.GetPKIDir(), "client-key.pem")
    csrPEM, err := GenerateCSR(keyPath)
    if err != nil {
        return fmt.Errorf("generate CSR: %w", err)
    }
    
    // Request renewal
    resp, err := r.client.RenewCertificate(ctx, identity.AgentID, csrPEM, identity.RefreshToken)
    if err != nil {
        return fmt.Errorf("renew certificate: %w", err)
    }
    
    // Store new certificate
    newCertPath := filepath.Join(r.state.GetPKIDir(), "client-cert.pem.new")
    if err := os.WriteFile(newCertPath, []byte(resp.ClientCertificatePEM), 0600); err != nil {
        return fmt.Errorf("write new cert: %w", err)
    }
    
    // Test new certificate
    if err := r.testCertificate(ctx, newCertPath, keyPath); err != nil {
        os.Remove(newCertPath)
        return fmt.Errorf("test new cert: %w", err)
    }
    
    // Atomically replace old certificate
    if err := os.Rename(newCertPath, certPath); err != nil {
        return fmt.Errorf("replace cert: %w", err)
    }
    
    log.Printf("Certificate rotated successfully, new expiry: %s", resp.ExpiresAt)
    return nil
}

func (r *CertRotator) testCertificate(ctx context.Context, certPath, keyPath string) error {
    // Try a test heartbeat with new certificate
    cert, err := tls.LoadX509KeyPair(certPath, keyPath)
    if err != nil {
        return err
    }
    
    // Create test client with new cert
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                Certificates: []tls.Certificate{cert},
            },
        },
        Timeout: 10 * time.Second,
    }
    
    // Send test heartbeat
    req, err := http.NewRequestWithContext(ctx, "POST", r.controlPlaneURL+"/v1/heartbeat", nil)
    if err != nil {
        return err
    }
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("test heartbeat failed: %d", resp.StatusCode)
    }
    
    return nil
}
```

### 3. Integration with Agent

**Modify:** `cmd/edge/main.go` in `run` command

```go
func runAgent(cfg Config) error {
    // ... existing setup ...
    
    // Start certificate rotator
    rotator := mtls.NewRotator(stateStore, client)
    go rotator.Start(ctx)
    
    // ... rest of agent loop ...
}
```

### 4. Backend Support

**Already implemented** in Phase 3:
- `POST /v1/renew` endpoint
- `UpdateAgentCertificate` DB method

**Enhancement needed:** Track certificate history

Add to `internal/controlplane/db/store.go`:
```go
type CertificateHistory struct {
    ID          string    `json:"id"`
    AgentID     string    `json:"agent_id"`
    Serial      string    `json:"serial"`
    IssuedAt    time.Time `json:"issued_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

ListCertificateHistory(ctx context.Context, agentID string) ([]CertificateHistory, error)
```

### 5. CLI Commands

Add `renew` command (already done in Phase 3) for manual rotation.

Add to `status` command output:
```
Certificate:
  Serial:     1234567890abcdef
  Issued:     2024-01-14 10:30:00 UTC
  Expires:    2024-01-15 10:30:00 UTC (23h remaining)
  Rotates in: 18h (at 20% remaining)
```

## Testing

**Unit tests:** `internal/edge/mtls/rotation_test.go`
- Test threshold calculation
- Test rotation trigger
- Test failed rotation handling
- Test certificate parsing

**Integration test:**
- Start agent with short-lived cert (5 minutes)
- Verify automatic rotation happens
- Verify no disruption to service

## Definition of Done
- [ ] Automatic rotation implemented
- [ ] Rotation threshold configurable
- [ ] Failed rotation alerts logged
- [ ] Manual rotation still works
- [ ] Tests pass

## Estimated Effort
6-8 hours
