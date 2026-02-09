# Phase 5 Task 4: Multi-tenancy Hardening

## Task Description
Verify tenant isolation, add resource quotas, and implement per-tenant rate limits.

## Requirements

### 1. Tenant Isolation Verification

**File:** `internal/controlplane/tenant/isolation.go`

```go
package tenant

import (
    "context"
    "fmt"
)

// IsolationEnforcer ensures strict tenant separation
type IsolationEnforcer struct {
    repo Repo
}

func NewIsolationEnforcer(repo Repo) *IsolationEnforcer {
    return &IsolationEnforcer{repo: repo}
}

// EnforceTenantAccess checks if a resource belongs to the tenant
func (e *IsolationEnforcer) EnforceTenantAccess(ctx context.Context, tenantID string, resource Resource) error {
    if resource.TenantID != tenantID {
        return ErrTenantIsolationViolation
    }
    return nil
}

// EnforceQueryScope adds tenant filter to all queries
func (e *IsolationEnforcer) EnforceQueryScope(ctx context.Context, tenantID string) context.Context {
    return context.WithValue(ctx, ctxTenantScope{}, tenantID)
}

type Resource struct {
    TenantID string
    Type     string
    ID       string
}

var ErrTenantIsolationViolation = fmt.Errorf("tenant isolation violation")
```

### 2. Resource Quotas

**File:** `internal/controlplane/tenant/quotas.go`

```go
package tenant

import (
    "context"
    "fmt"
)

// QuotaManager tracks and enforces resource limits per tenant
type QuotaManager struct {
    repo Repo
}

type QuotaLimits struct {
    MaxSites           int `json:"max_sites"`
    MaxAgentsPerSite   int `json:"max_agents_per_site"`
    MaxVMsPerAgent     int `json:"max_vms_per_agent"`
    MaxConcurrentPlans int `json:"max_concurrent_plans"`
    MaxAPIKeys         int `json:"max_api_keys"`
    MaxExecutionsDay   int `json:"max_executions_per_day"`
}

type QuotaUsage struct {
    Sites           int `json:"sites"`
    Agents          int `json:"agents"`
    VMs             int `json:"vms"`
    ActivePlans     int `json:"active_plans"`
    APIKeys         int `json:"api_keys"`
    ExecutionsToday int `json:"executions_today"`
}

func (q *QuotaManager) CheckQuota(ctx context.Context, tenantID string, resource string) error {
    limits, err := q.getLimits(ctx, tenantID)
    if err != nil {
        return err
    }
    
    usage, err := q.getUsage(ctx, tenantID)
    if err != nil {
        return err
    }
    
    switch resource {
    case "site":
        if usage.Sites >= limits.MaxSites {
            return fmt.Errorf("site quota exceeded: %d/%d", usage.Sites, limits.MaxSites)
        }
    case "api_key":
        if usage.APIKeys >= limits.MaxAPIKeys {
            return fmt.Errorf("API key quota exceeded: %d/%d", usage.APIKeys, limits.MaxAPIKeys)
        }
    // ... etc
    }
    
    return nil
}
```

### 3. Per-Tenant Rate Limiting

**File:** `internal/controlplane/api/ratelimit.go` (extend)

```go
// TenantRateLimiter extends rate limiting with per-tenant limits
type TenantRateLimiter struct {
    globalLimiter *RateLimiter
    tenantLimits  map[string]TenantLimitConfig
}

type TenantLimitConfig struct {
    RequestsPerSecond float64
    BurstSize         int
}

func (t *TenantRateLimiter) Allow(ctx context.Context, tenantID string) bool {
    // Check tenant-specific limits
    if config, ok := t.tenantLimits[tenantID]; ok {
        return t.checkTenantLimit(tenantID, config)
    }
    
    // Fall back to global limiter
    return t.globalLimiter.Allow(ctx)
}
```

### 4. Tenant Usage Metrics

**File:** `internal/controlplane/api/server.go` (add endpoint)

```go
func (a *App) handleGetTenantUsage(w http.ResponseWriter, r *http.Request) {
    tenantID := r.PathValue("tenantID")
    if !a.tenantAllowed(r.Context(), tenantID) {
        writeError(w, http.StatusForbidden, "tenant mismatch")
        return
    }
    
    usage, err := a.repo.GetTenantUsage(r.Context(), tenantID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to get usage")
        return
    }
    
    limits := a.quotaManager.GetLimits(tenantID)
    
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "usage": usage,
        "limits": limits,
        "quota_usage_percent": calculateQuotaPercent(usage, limits),
    })
}
```

### 5. Database Row-Level Security (Optional)

**File:** `db/migrations/0004_rls.sql`

```sql
-- Enable RLS on tables
ALTER TABLE sites ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE vms ENABLE ROW LEVEL SECURITY;

-- Create policies
CREATE POLICY site_tenant_isolation ON sites
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY agent_tenant_isolation ON agents
    USING (tenant_id = current_setting('app.current_tenant')::UUID);
```

### 6. Tenant Data Export/Deletion (GDPR)

**File:** `internal/controlplane/tenant/gdpr.go`

```go
package tenant

import (
    "context"
    "encoding/json"
)

// ExportTenantData exports all data for a tenant
func (e *IsolationEnforcer) ExportTenantData(ctx context.Context, tenantID string) (*TenantExport, error) {
    export := &TenantExport{
        TenantID:   tenantID,
        ExportedAt: time.Now().UTC(),
    }
    
    // Export all tenant data
    sites, _ := e.repo.ListSites(ctx, tenantID)
    agents, _ := e.repo.ListAgents(ctx, tenantID)
    auditEvents, _ := e.repo.ListAuditEvents(ctx, tenantID, 0)
    
    export.Sites = sites
    export.Agents = agents
    export.AuditEvents = auditEvents
    
    return export, nil
}

// DeleteTenantData hard-deletes all tenant data (GDPR right to erasure)
func (e *IsolationEnforcer) DeleteTenantData(ctx context.Context, tenantID string) error {
    // This is a destructive operation requiring confirmation
    return e.repo.DeleteTenant(ctx, tenantID)
}
```

## Deliverables
1. `internal/controlplane/tenant/isolation.go` - Isolation enforcement
2. `internal/controlplane/tenant/quotas.go` - Resource quotas
3. `internal/controlplane/tenant/gdpr.go` - Data export/deletion
4. Updated `internal/controlplane/api/ratelimit.go` - Per-tenant limits
5. Updated `internal/controlplane/db/postgres.go` - Quota methods
6. Database migration for RLS policies (optional)

## Estimated Effort
6-8 hours
