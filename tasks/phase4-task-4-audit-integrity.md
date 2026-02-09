# Phase 4 Task 4: Audit Log Integrity

## Task Description
Implement audit log integrity verification using cryptographic hashing.

## Background
Audit logs must be tamper-evident. Each audit entry should include a hash of the previous entry, creating a chain of integrity. Any modification to historical entries would break the chain and be detectable.

## Requirements

### 1. Audit Entry Chain

**Modify:** `internal/controlplane/db/store.go`

```go
type AuditEvent struct {
    ID           string          `json:"id"`
    Timestamp    time.Time       `json:"timestamp"`
    TenantID     string          `json:"tenant_id"`
    SiteID       string          `json:"site_id"`
    ActorType    string          `json:"actor_type"`
    ActorID      string          `json:"actor_id"`
    Action       string          `json:"action"`
    ResourceType string          `json:"resource_type"`
    ResourceID   string          `json:"resource_id"`
    RequestID    string          `json:"request_id"`
    SourceIP     string          `json:"source_ip"`
    Metadata     json.RawMessage `json:"metadata,omitempty"`
    
    // Integrity fields
    PrevHash     string          `json:"prev_hash"`      // Hash of previous entry
    EntryHash    string          `json:"entry_hash"`     // Hash of this entry
    ChainValid   bool            `json:"chain_valid"`    // Validity flag
}
```

### 2. Hash Chain Implementation

**File:** `internal/controlplane/audit/chain.go`

```go
package audit

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"
)

// ChainManager manages the audit log hash chain
type ChainManager struct {
    repo AuditRepository
}

type AuditRepository interface {
    GetLastAuditEvent(ctx context.Context) (*AuditEvent, error)
    WriteAuditEvent(ctx context.Context, event *AuditEvent) error
    GetAuditEvent(ctx context.Context, id string) (*AuditEvent, error)
    ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]AuditEvent, error)
}

func NewChainManager(repo AuditRepository) *ChainManager {
    return &ChainManager{repo: repo}
}

// CreateAuditEvent creates a new audit event with proper chaining
func (cm *ChainManager) CreateAuditEvent(ctx context.Context, event *AuditEvent) error {
    // Get previous event for chaining
    prevEvent, err := cm.repo.GetLastAuditEvent(ctx)
    if err != nil {
        return fmt.Errorf("get last audit event: %w", err)
    }
    
    // Set timestamp
    event.Timestamp = time.Now().UTC()
    
    // Set previous hash
    if prevEvent != nil {
        event.PrevHash = prevEvent.EntryHash
    } else {
        // Genesis entry - use fixed value or empty
        event.PrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
    }
    
    // Calculate entry hash
    event.EntryHash = cm.calculateHash(event)
    event.ChainValid = true
    
    // Store event
    return cm.repo.WriteAuditEvent(ctx, event)
}

// calculateHash computes the hash of an audit entry
func (cm *ChainManager) calculateHash(event *AuditEvent) string {
    // Create a copy without the hash fields for hashing
    data := struct {
        ID           string          `json:"id"`
        Timestamp    time.Time       `json:"timestamp"`
        TenantID     string          `json:"tenant_id"`
        SiteID       string          `json:"site_id"`
        ActorType    string          `json:"actor_type"`
        ActorID      string          `json:"actor_id"`
        Action       string          `json:"action"`
        ResourceType string          `json:"resource_type"`
        ResourceID   string          `json:"resource_id"`
        RequestID    string          `json:"request_id"`
        SourceIP     string          `json:"source_ip"`
        Metadata     json.RawMessage `json:"metadata,omitempty"`
        PrevHash     string          `json:"prev_hash"`
    }{
        ID:           event.ID,
        Timestamp:    event.Timestamp,
        TenantID:     event.TenantID,
        SiteID:       event.SiteID,
        ActorType:    event.ActorType,
        ActorID:      event.ActorID,
        Action:       event.Action,
        ResourceType: event.ResourceType,
        ResourceID:   event.ResourceID,
        RequestID:    event.RequestID,
        SourceIP:     event.SourceIP,
        Metadata:     event.Metadata,
        PrevHash:     event.PrevHash,
    }
    
    // Serialize to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return ""
    }
    
    // Calculate SHA256 hash
    hash := sha256.Sum256(jsonData)
    return hex.EncodeToString(hash[:])
}

// VerifyChain verifies the integrity of the entire audit chain
func (cm *ChainManager) VerifyChain(ctx context.Context) (*ChainVerificationResult, error) {
    result := &ChainVerificationResult{
        Valid:      true,
        Total:      0,
        Invalid:    0,
        FirstValid: true,
    }
    
    // Get all audit events
    events, err := cm.repo.ListAuditEvents(ctx, "", 0)  // 0 = no limit
    if err != nil {
        return nil, err
    }
    
    result.Total = len(events)
    
    for i, event := range events {
        // Recalculate hash
        expectedHash := cm.calculateHash(&event)
        
        if expectedHash != event.EntryHash {
            event.ChainValid = false
            result.Valid = false
            result.Invalid++
            
            if i == 0 {
                result.FirstValid = false
            }
            
            // Update the invalid flag in database
            cm.repo.UpdateAuditEventValidity(ctx, event.ID, false)
            continue
        }
        
        // Verify chain link (except for first entry)
        if i > 0 {
            prevEvent := events[i-1]
            if event.PrevHash != prevEvent.EntryHash {
                event.ChainValid = false
                result.Valid = false
                result.Invalid++
                cm.repo.UpdateAuditEventValidity(ctx, event.ID, false)
                continue
            }
        }
        
        event.ChainValid = true
    }
    
    return result, nil
}

// VerifyEvent verifies a single audit event
func (cm *ChainManager) VerifyEvent(ctx context.Context, eventID string) (bool, error) {
    event, err := cm.repo.GetAuditEvent(ctx, eventID)
    if err != nil {
        return false, err
    }
    
    expectedHash := cm.calculateHash(event)
    return expectedHash == event.EntryHash, nil
}

type ChainVerificationResult struct {
    Valid      bool   `json:"valid"`
    Total      int    `json:"total"`
    Invalid    int    `json:"invalid"`
    FirstValid bool   `json:"first_valid"`
}
```

### 3. Database Schema

**Migration:** `db/migrations/0003_audit_integrity.sql`

```sql
-- Add integrity fields to audit_events
ALTER TABLE audit_events ADD COLUMN prev_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN entry_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN chain_valid BOOLEAN NOT NULL DEFAULT true;

-- Create index for chain verification
CREATE INDEX idx_audit_events_chain ON audit_events(created_at, entry_hash);

-- Add unique constraint on entry_hash
CREATE UNIQUE INDEX idx_audit_events_hash ON audit_events(entry_hash);
```

### 4. Integration with Server

**Modify:** `internal/controlplane/api/server.go`

```go
func NewApp(cfg Config, repo store.Repo) (*App, error) {
    // ... existing setup ...
    
    // Setup audit chain manager
    a.auditChain = audit.NewChainManager(repo)
    
    return a, nil
}

func (a *App) WriteAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata json.RawMessage) error {
    event := &audit.AuditEvent{
        ID:           uuid.NewString(),
        TenantID:     tenantID,
        SiteID:       siteID,
        ActorType:    actorType,
        ActorID:      actorID,
        Action:       action,
        ResourceType: resourceType,
        ResourceID:   resourceID,
        RequestID:    requestID,
        SourceIP:     sourceIP,
        Metadata:     metadata,
    }
    
    return a.auditChain.CreateAuditEvent(ctx, event)
}
```

### 5. Admin API Endpoints

**Add to:** `internal/controlplane/api/server.go`

```go
func (a *App) registerRoutes() {
    // ... existing routes ...
    
    // Audit integrity endpoints (admin only)
    a.mux.Handle("POST /admin/audit/verify", a.adminAuth(http.HandlerFunc(a.handleVerifyAuditChain)))
    a.mux.Handle("GET /admin/audit/events", a.adminAuth(http.HandlerFunc(a.handleListAuditEvents)))
}

func (a *App) handleVerifyAuditChain(w http.ResponseWriter, r *http.Request) {
    result, err := a.auditChain.VerifyChain(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to verify chain")
        return
    }
    
    writeJSON(w, http.StatusOK, result)
}

func (a *App) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
    tenantID := r.URL.Query().Get("tenant_id")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 100
    }
    
    events, err := a.repo.ListAuditEvents(r.Context(), tenantID, limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to list events")
        return
    }
    
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "events": events,
        "count":  len(events),
    })
}
```

### 6. Background Verification

**File:** `internal/controlplane/audit/verifier.go`

```go
package audit

import (
    "context"
    "log"
    "time"
)

// BackgroundVerifier runs periodic chain verification
type BackgroundVerifier struct {
    chain   *ChainManager
    interval time.Duration
}

func NewBackgroundVerifier(chain *ChainManager, interval time.Duration) *BackgroundVerifier {
    return &BackgroundVerifier{
        chain:    chain,
        interval: interval,
    }
}

func (bv *BackgroundVerifier) Start(ctx context.Context) {
    ticker := time.NewTicker(bv.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            result, err := bv.chain.VerifyChain(ctx)
            if err != nil {
                log.Printf("audit chain verification error: %v", err)
                continue
            }
            
            if !result.Valid {
                log.Printf("AUDIT ALERT: %d of %d audit entries have invalid chain", 
                    result.Invalid, result.Total)
            }
        }
    }
}
```

### 7. Testing

**Unit tests:** `internal/controlplane/audit/chain_test.go`

```go
func TestChainManager_CreateAuditEvent(t *testing.T) {
    repo := &mockAuditRepo{}
    cm := NewChainManager(repo)
    
    // Create first event (genesis)
    event1 := &AuditEvent{ID: "1", Action: "test"}
    if err := cm.CreateAuditEvent(context.Background(), event1); err != nil {
        t.Fatal(err)
    }
    
    if event1.PrevHash != genesisHash {
        t.Errorf("expected genesis hash, got %s", event1.PrevHash)
    }
    
    if event1.EntryHash == "" {
        t.Error("expected entry hash to be set")
    }
    
    // Create second event
    event2 := &AuditEvent{ID: "2", Action: "test2"}
    if err := cm.CreateAuditEvent(context.Background(), event2); err != nil {
        t.Fatal(err)
    }
    
    if event2.PrevHash != event1.EntryHash {
        t.Error("expected prev_hash to be previous entry's hash")
    }
}

func TestChainManager_VerifyChain(t *testing.T) {
    repo := &mockAuditRepo{}
    cm := NewChainManager(repo)
    
    // Create chain of events
    for i := 0; i < 5; i++ {
        event := &AuditEvent{ID: fmt.Sprintf("%d", i), Action: "test"}
        if err := cm.CreateAuditEvent(context.Background(), event); err != nil {
            t.Fatal(err)
        }
    }
    
    // Verify chain
    result, err := cm.VerifyChain(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    
    if !result.Valid {
        t.Error("expected chain to be valid")
    }
    
    if result.Total != 5 {
        t.Errorf("expected 5 events, got %d", result.Total)
    }
}
```

## Definition of Done
- [ ] Audit entries include hash chain
- [ ] New entries link to previous
- [ ] Chain verification works
- [ ] Background verification runs
- [ ] Admin API for verification
- [ ] Tests pass

## Estimated Effort
6-8 hours
