# Phase 5 Task 2: Performance Optimization

## Task Description
Optimize database queries, add caching layer, and implement connection pooling.

## Requirements

### 1. Database Query Optimization

**File:** `internal/controlplane/db/postgres.go`

Add query metrics and optimize slow queries:

```go
// QueryMetrics tracks database query performance
type QueryMetrics struct {
    Query     string        `json:"query"`
    Duration  time.Duration `json:"duration"`
    Rows      int           `json:"rows"`
}

// Add indexes for common queries
const createIndexesSQL = `
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agents_tenant_site ON agents(tenant_id, site_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_heartbeats_agent_time ON heartbeats(agent_id, ingested_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_executions_plan ON executions(plan_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_events_tenant_time ON audit_events(tenant_id, occurred_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_logs_execution ON execution_logs(execution_id, emitted_at DESC);
`
```

### 2. Connection Pooling

**File:** `internal/controlplane/db/postgres.go`

```go
func NewPostgres(connStr string) (*PostgresRepo, error) {
    config, err := pgxpool.ParseConfig(connStr)
    if err != nil {
        return nil, err
    }
    
    // Connection pool tuning
    config.MaxConns = int32(getEnvInt("DB_MAX_CONNECTIONS", 25))
    config.MinConns = int32(getEnvInt("DB_MIN_CONNECTIONS", 5))
    config.MaxConnLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
    config.MaxConnIdleTime = getEnvDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)
    config.HealthCheckPeriod = getEnvDuration("DB_HEALTH_CHECK_PERIOD", 5*time.Minute)
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    // ...
}
```

### 3. Caching Layer

**File:** `internal/controlplane/cache/cache.go`

```go
package cache

import (
    "context"
    "time"
    
    "github.com/patrickmn/go-cache"
)

// Cache provides a simple in-memory cache
type Cache struct {
    local *cache.Cache
}

func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
    return &Cache{
        local: cache.New(defaultExpiration, cleanupInterval),
    }
}

func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool) {
    return c.local.Get(key)
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
    c.local.Set(key, value, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) {
    c.local.Delete(key)
}

func (c *Cache) Flush(ctx context.Context) {
    c.local.Flush()
}
```

### 4. API Key Caching

**File:** `internal/controlplane/api/server.go`

Cache API key validations to reduce DB load:

```go
func (a *App) apiKeyAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
        
        // Check cache first
        cacheKey := "apikey:" + hashString(apiKey)
        if cached, ok := a.cache.Get(r.Context(), cacheKey); ok {
            ctx := context.WithValue(r.Context(), ctxTenantID{}, cached.(string))
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        
        // Validate against DB
        validation, err := a.repo.ValidateAPIKey(r.Context(), hashString(apiKey))
        if err != nil {
            writeError(w, http.StatusUnauthorized, "invalid api key")
            return
        }
        
        // Cache for 5 minutes
        a.cache.Set(r.Context(), cacheKey, validation.TenantID, 5*time.Minute)
        
        ctx := context.WithValue(r.Context(), ctxTenantID{}, validation.TenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 5. Query Timeout Contexts

Add context timeouts to all database operations:

```go
func (p *PostgresRepo) GetAgentByID(ctx context.Context, agentID string) (Agent, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // ... existing query
}
```

### 6. Performance Metrics

**File:** `internal/controlplane/metrics/metrics.go`

```go
var (
    DBQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "nkudo_db_query_duration_seconds",
        Help: "Database query duration",
    }, []string{"query", "table"})
    
    DBConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "nkudo_db_connections_active",
        Help: "Active database connections",
    })
    
    CacheHitRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "nkudo_cache_hit_rate",
        Help: "Cache hit rate",
    }, []string{"cache_name"})
)
```

## Deliverables
1. `internal/controlplane/db/indexes.sql` - Database indexes
2. `internal/controlplane/cache/cache.go` - Caching layer
3. `internal/controlplane/metrics/db.go` - DB metrics
4. Updated `internal/controlplane/db/postgres.go` - Connection pooling
5. Updated `internal/controlplane/api/server.go` - API key caching
6. Performance benchmark tests

## Dependencies
```bash
go get github.com/patrickmn/go-cache
go get github.com/prometheus/client_golang/prometheus
```

## Estimated Effort
6-8 hours
