# Audit Log Chain Integrity

This package provides cryptographic chain integrity verification for audit logs,
similar to blockchain technology but designed for audit trail integrity.

## Overview

The audit chain ensures that once an audit event is recorded, it cannot be
tampered with without detection. Each event contains:

- **PrevHash**: Hash of the previous event in the chain
- **EntryHash**: Hash of the current event's data
- **ChainValid**: Flag indicating whether this event passes integrity checks

## Architecture

### ChainManager

The `ChainManager` is the core component that:
- Creates new audit events with proper chain linking
- Calculates SHA256 hashes of event data
- Verifies chain integrity (full and single event)
- Provides chain information

### BackgroundVerifier

The `BackgroundVerifier` runs periodic integrity checks:
- Configurable verification interval
- Logs warnings if chain integrity issues are detected
- Supports on-demand verification

## Hash Calculation

Event hashes are calculated using SHA256 over a JSON representation of the event
data, excluding the `EntryHash` field itself (since that's what we're calculating).

The hash includes:
- Event ID (if set)
- Tenant ID
- Site ID
- Actor information (type, user ID, agent ID)
- Action details
- Resource information
- Metadata
- Timestamp
- Previous hash
- Chain validity flag

### Genesis Hash

The first event in the chain uses a special genesis hash:
```
0000000000000000000000000000000000000000000000000000000000000000
```

## Usage

### Creating an Audit Event

```go
import (
    "github.com/kubedoio/n-kudo/internal/controlplane/audit"
    store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

chainManager := audit.NewChainManager(repo)

input := store.AuditEventInput{
    TenantID:     "tenant-uuid",
    SiteID:       "site-uuid",
    ActorType:    "USER",
    ActorID:      "user-uuid",
    Action:       "vm.create",
    ResourceType: "vm",
    ResourceID:   "vm-uuid",
    RequestID:    "req-id",
    SourceIP:     "192.168.1.1",
    Metadata:     []byte(`{"vcpu": 2, "memory": 4096}`),
}

event, err := chainManager.CreateAuditEvent(ctx, input)
```

### Verifying the Chain

```go
// Full chain verification
result, err := chainManager.VerifyChain(ctx)
if err != nil {
    log.Fatal(err)
}

if !result.Valid {
    log.Printf("Chain integrity compromised: %d invalid events", result.Invalid)
}
```

### Background Verification

```go
verifier := audit.NewBackgroundVerifier(chainManager, 5*time.Minute)

// Start in background
go verifier.Start(ctx)

// Stop when done
defer verifier.Stop()
```

## API Endpoints

### POST /admin/audit/verify

Verifies the entire audit log chain.

**Response:**
```json
{
  "valid": true,
  "total": 150,
  "invalid": 0,
  "first_valid": 1
}
```

### GET /admin/audit/events

Lists audit events with chain status.

**Query Parameters:**
- `tenant_id`: Filter by tenant (optional)
- `limit`: Maximum events to return (default: 100, max: 1000)

**Response:**
```json
{
  "events": [
    {
      "id": 1,
      "tenant_id": "tenant-uuid",
      "actor_type": "USER",
      "action": "vm.create",
      "prev_hash": "0000...",
      "entry_hash": "a1b2c3...",
      "chain_valid": true
    }
  ]
}
```

### GET /admin/audit/chain-info

Returns information about the current state of the audit chain.

**Response:**
```json
{
  "total_events": 150,
  "genesis_hash": "0000000000000000000000000000000000000000000000000000000000000000",
  "last_event_id": 150,
  "last_entry_hash": "a1b2c3...",
  "chain_head_valid": true
}
```

## Database Schema

The `audit_events` table includes chain integrity columns:

```sql
CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID,
  actor_type TEXT NOT NULL,
  actor_user_id UUID,
  actor_agent_id UUID,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  request_id TEXT,
  source_ip INET,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- Chain integrity columns
  prev_hash TEXT NOT NULL DEFAULT '0000...',
  entry_hash TEXT NOT NULL DEFAULT '',
  chain_valid BOOLEAN NOT NULL DEFAULT TRUE
);
```

## Security Considerations

1. **Immutable Chain**: Once written, events should not be modified
2. **Hash Verification**: Regular verification detects tampering
3. **Database Security**: Protect database access to prevent direct manipulation
4. **Backup Integrity**: Chain verification can detect backup restoration issues

## Testing

Run the audit chain tests:

```bash
go test -v ./internal/controlplane/audit/...
```
