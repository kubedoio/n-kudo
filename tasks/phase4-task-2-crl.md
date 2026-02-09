# Phase 4 Task 2: Certificate Revocation List (CRL)

## Task Description
Implement Certificate Revocation List (CRL) support for agent certificate validation.

## Background
When an agent is unenrolled or compromised, its certificate should be immediately invalidated. CRL provides a way to check if a certificate has been revoked before accepting it.

## Requirements

### 1. CRL Generation and Distribution

**File:** `internal/controlplane/pki/crl.go`

```go
package pki

import (
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "math/big"
    "time"
)

type CRLManager struct {
    ca      *InternalCA
    revoked map[string]time.Time  // serial -> revoked_at
}

func NewCRLManager(ca *InternalCA) *CRLManager {
    return &CRLManager{
        ca:      ca,
        revoked: make(map[string]time.Time),
    }
}

func (m *CRLManager) Revoke(serial string) error {
    m.revoked[serial] = time.Now().UTC()
    return m.generateCRL()
}

func (m *CRLManager) generateCRL() error {
    now := time.Now().UTC()
    
    var revokedCerts []pkix.RevokedCertificate
    for serial, revokedAt := range m.revoked {
        serialNum := new(big.Int)
        serialNum.SetString(serial, 16)
        
        revokedCerts = append(revokedCerts, pkix.RevokedCertificate{
            SerialNumber:   serialNum,
            RevocationTime: revokedAt,
        })
    }
    
    crlTemplate := &x509.RevocationList{
        SignatureAlgorithm: x509.SHA256WithRSA,
        RevokedCertificates: revokedCerts,
        Number:             big.NewInt(int64(now.Unix())),
        ThisUpdate:         now,
        NextUpdate:         now.Add(24 * time.Hour),
    }
    
    crlBytes, err := x509.CreateRevocationList(
        rand.Reader,
        crlTemplate,
        m.ca.cert,
        m.ca.key,
    )
    if err != nil {
        return err
    }
    
    // Store CRL
    m.crlDER = crlBytes
    return nil
}

func (m *CRLManager) GetCRL() []byte {
    return m.crlDER
}

func (m *CRLManager) IsRevoked(serial string) bool {
    _, ok := m.revoked[serial]
    return ok
}
```

### 2. CRL Endpoint

**Add to:** `internal/controlplane/api/server.go`

```go
func (a *App) registerRoutes() {
    // ... existing routes ...
    
    // CRL distribution point
    a.mux.HandleFunc("GET /v1/crl", a.handleGetCRL)
    a.mux.HandleFunc("GET /v1/crl.pem", a.handleGetCRLPEM)
}

func (a *App) handleGetCRL(w http.ResponseWriter, r *http.Request) {
    crl := a.crlManager.GetCRL()
    w.Header().Set("Content-Type", "application/pkix-crl")
    w.Write(crl)
}

func (a *App) handleGetCRLPEM(w http.ResponseWriter, r *http.Request) {
    crl := a.crlManager.GetCRL()
    pemBlock := &pem.Block{
        Type:  "X509 CRL",
        Bytes: crl,
    }
    w.Header().Set("Content-Type", "application/x-pem-file")
    pem.Encode(w, pemBlock)
}
```

### 3. Certificate Validation with CRL

**Modify:** `internal/controlplane/api/server.go` in `agentMTLSAuth`

```go
func (a *App) agentMTLSAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
            writeError(w, http.StatusUnauthorized, "missing client certificate")
            return
        }
        
        cert := r.TLS.PeerCertificates[0]
        
        // Check CRL
        if a.crlManager.IsRevoked(cert.SerialNumber.String()) {
            writeError(w, http.StatusUnauthorized, "certificate revoked")
            return
        }
        
        // ... rest of existing validation ...
    })
}
```

### 4. Revocation on Unenroll

**Modify:** `internal/controlplane/api/server.go` in `handleUnenroll`

```go
func (a *App) handleUnenroll(w http.ResponseWriter, r *http.Request) {
    agent := r.Context().Value(ctxAgent{}).(store.Agent)
    
    // Revoke certificate
    if err := a.crlManager.Revoke(agent.CertSerial); err != nil {
        log.Printf("failed to revoke certificate: %v", err)
        writeError(w, http.StatusInternalServerError, "failed to revoke certificate")
        return
    }
    
    // Mark agent as unenrolled in database
    if err := a.repo.UnenrollAgent(r.Context(), agent.ID); err != nil {
        writeError(w, http.StatusInternalServerError, "failed to unenroll agent")
        return
    }
    
    // Log audit event
    _ = a.repo.WriteAudit(r.Context(), agent.TenantID, agent.SiteID, "AGENT", agent.ID, 
        "agent.unenroll", "agent", agent.ID, requestID(r), sourceIP(r), nil)
    
    w.WriteHeader(http.StatusNoContent)
}
```

### 5. CRL in Agent Certificate

When issuing certificates, include CRL distribution point:

**Modify:** `internal/controlplane/pki/ca.go` in `SignAgentCSR`

```go
func (ca *InternalCA) SignAgentCSR(csrPEM []byte, agentID, tenantID, siteID string, ttl time.Duration) ([]byte, string, error) {
    // ... existing code ...
    
    template := &x509.Certificate{
        // ... existing fields ...
        
        CRLDistributionPoints: []string{
            "https://control-plane:8443/v1/crl.pem",
        },
    }
    
    // ... rest of signing ...
}
```

### 6. CRL Persistence

CRL should persist across control-plane restarts.

**Database table:**
```sql
CREATE TABLE crl_entries (
    serial TEXT PRIMARY KEY,
    revoked_at TIMESTAMPTZ NOT NULL,
    reason TEXT,
    agent_id UUID REFERENCES agents(id)
);

CREATE INDEX idx_crl_entries_revoked_at ON crl_entries(revoked_at);
```

**DB methods:**
```go
func (p *PostgresRepo) RevokeCertificate(ctx context.Context, serial string, agentID string, reason string) error
func (p *PostgresRepo) IsCertificateRevoked(ctx context.Context, serial string) (bool, error)
func (p *PostgresRepo) ListRevokedCertificates(ctx context.Context) ([]CRLEntry, error)
```

## Testing

**Unit tests:** `internal/controlplane/pki/crl_test.go`
- Test CRL generation
- Test certificate revocation
- Test CRL validation
- Test serialization/deserialization

**Integration tests:**
- Test revoked cert rejected
- Test unenroll triggers revocation
- Test CRL endpoint returns valid CRL

## Definition of Done
- [ ] CRL generation implemented
- [ ] CRL endpoint exposed
- [ ] Certificate validation checks CRL
- [ ] Unenroll revokes certificate
- [ ] CRL persists in database
- [ ] Tests pass

## Estimated Effort
6-8 hours
