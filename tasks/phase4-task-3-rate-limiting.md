# Phase 4 Task 3: Rate Limiting

## Task Description
Implement rate limiting for API endpoints to prevent abuse.

## Requirements

### 1. Rate Limits by Endpoint

| Endpoint | Rate Limit | Burst |
|----------|------------|-------|
| `POST /enroll` | 10/minute | 20 |
| `POST /v1/enroll` | 10/minute | 20 |
| `POST /agents/heartbeat` | 60/minute | 120 |
| `POST /v1/heartbeat` | 60/minute | 120 |
| `POST /tenants` | 5/minute | 10 |
| `POST /tenants/{id}/api-keys` | 10/minute | 20 |
| All other endpoints | 100/minute | 200 |

### 2. Rate Limiting Implementation

**File:** `internal/controlplane/api/ratelimit.go`

```go
package api

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "strings"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// RateLimiter manages rate limits per client
type RateLimiter struct {
    mu       sync.RWMutex
    limiters map[string]*rate.Limiter
    config   RateLimitConfig
}

type RateLimitConfig struct {
    DefaultRate  rate.Limit    // Requests per second
    DefaultBurst int           // Burst size
    
    // Per-endpoint overrides
    EndpointLimits map[string]EndpointLimit
}

type EndpointLimit struct {
    Rate  rate.Limit
    Burst int
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        config:   config,
    }
}

// getClientID extracts client identifier from request
// Uses IP address for agent endpoints, API key hash for tenant endpoints
func (rl *RateLimiter) getClientID(r *http.Request) string {
    // For agent endpoints, use IP + agent ID if available
    if isAgentEndpoint(r.URL.Path) {
        ip := getClientIP(r)
        if agentID := r.Header.Get("X-Agent-ID"); agentID != "" {
            return fmt.Sprintf("agent:%s:%s", ip, agentID)
        }
        return fmt.Sprintf("ip:%s", ip)
    }
    
    // For API endpoints, use API key hash
    if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
        return "apikey:" + hashString(apiKey)
    }
    
    // For admin endpoints, use admin key hash
    if adminKey := r.Header.Get("X-Admin-Key"); adminKey != "" {
        return "admin:" + hashString(adminKey)
    }
    
    // Fallback to IP
    return "ip:" + getClientIP(r)
}

func (rl *RateLimiter) getLimiter(clientID string, endpoint string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    key := clientID + ":" + endpoint
    
    limiter, exists := rl.limiters[key]
    if !exists {
        // Get rate for this endpoint
        endpointLimit, ok := rl.config.EndpointLimits[endpoint]
        if !ok {
            endpointLimit = EndpointLimit{
                Rate:  rl.config.DefaultRate,
                Burst: rl.config.DefaultBurst,
            }
        }
        
        limiter = rate.NewLimiter(endpointLimit.Rate, endpointLimit.Burst)
        rl.limiters[key] = limiter
    }
    
    return limiter
}

func (rl *RateLimiter) Allow(r *http.Request) bool {
    clientID := rl.getClientID(r)
    endpoint := r.URL.Path
    
    limiter := rl.getLimiter(clientID, endpoint)
    return limiter.Allow()
}

// Middleware returns HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !rl.Allow(r) {
            w.Header().Set("Content-Type", "application/json")
            w.Header().Set("Retry-After", "60")
            w.WriteHeader(http.StatusTooManyRequests)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "rate limit exceeded",
            })
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

// Cleanup removes old limiters (call periodically)
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
    // In production, track last access time and remove stale entries
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header first
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // Take the first IP if multiple
        if idx := strings.Index(xff, ","); idx != -1 {
            xff = xff[:idx]
        }
        return strings.TrimSpace(xff)
    }
    
    // Check X-Real-IP
    if xri := r.Header.Get("X-Real-Ip"); xri != "" {
        return xri
    }
    
    // Fall back to remote address
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}

func isAgentEndpoint(path string) bool {
    return strings.HasPrefix(path, "/agents/") ||
           strings.HasPrefix(path, "/v1/") && 
           (strings.Contains(path, "heartbeat") || 
            strings.Contains(path, "logs") ||
            strings.Contains(path, "enroll"))
}
```

### 3. Integration with Server

**Modify:** `internal/controlplane/api/server.go`

```go
func NewApp(cfg Config, repo store.Repo) (*App, error) {
    // ... existing setup ...
    
    // Setup rate limiter
    rateLimitConfig := RateLimitConfig{
        DefaultRate:  rate.Limit(100.0/60.0),  // 100 per minute
        DefaultBurst: 200,
        EndpointLimits: map[string]EndpointLimit{
            "/enroll":               {Rate: rate.Limit(10.0/60.0), Burst: 20},
            "/v1/enroll":            {Rate: rate.Limit(10.0/60.0), Burst: 20},
            "/agents/heartbeat":     {Rate: rate.Limit(60.0/60.0), Burst: 120},
            "/v1/heartbeat":         {Rate: rate.Limit(60.0/60.0), Burst: 120},
            "/tenants":              {Rate: rate.Limit(5.0/60.0), Burst: 10},
            "/tenants/{tenantID}/api-keys": {Rate: rate.Limit(10.0/60.0), Burst: 20},
        },
    }
    a.rateLimiter = NewRateLimiter(rateLimitConfig)
    
    a.registerRoutes()
    return a, nil
}

func (a *App) Handler() http.Handler {
    // Apply rate limiting to all requests
    return a.withRequestLogging(a.rateLimiter.Middleware(a.mux))
}
```

### 4. Environment Configuration

Add to `internal/controlplane/api/config.go`:

```go
type Config struct {
    // ... existing fields ...
    
    // Rate limiting
    RateLimitEnabled    bool          `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
    RateLimitDefaultRPS float64       `env:"RATE_LIMIT_DEFAULT_RPS" envDefault:"1.67"`  // 100/minute
    RateLimitDefaultBurst int         `env:"RATE_LIMIT_DEFAULT_BURST" envDefault:"200"`
}
```

### 5. Metrics

Add rate limiting metrics:

```go
var (
    rateLimitHits = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_rate_limit_hits_total",
        Help: "Total rate limit hits",
    }, []string{"endpoint"})
    
    rateLimitBlocks = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_rate_limit_blocks_total",
        Help: "Total requests blocked by rate limit",
    }, []string{"endpoint"})
)
```

### 6. Testing

**Unit tests:** `internal/controlplane/api/ratelimit_test.go`

```go
func TestRateLimiter_Allow(t *testing.T) {
    config := RateLimitConfig{
        DefaultRate:  rate.Limit(1),  // 1 per second
        DefaultBurst: 2,
    }
    rl := NewRateLimiter(config)
    
    // Create request
    req := httptest.NewRequest("GET", "/test", nil)
    req.RemoteAddr = "192.168.1.1:12345"
    
    // First 2 requests should pass (burst)
    if !rl.Allow(req) {
        t.Error("first request should be allowed")
    }
    if !rl.Allow(req) {
        t.Error("second request should be allowed (burst)")
    }
    
    // Third request should be blocked
    if rl.Allow(req) {
        t.Error("third request should be blocked")
    }
}

func TestRateLimiter_Middleware(t *testing.T) {
    config := RateLimitConfig{
        DefaultRate:  0,  // Block all
        DefaultBurst: 0,
    }
    rl := NewRateLimiter(config)
    
    handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    
    req := httptest.NewRequest("GET", "/test", nil)
    rr := httptest.NewRecorder()
    
    handler.ServeHTTP(rr, req)
    
    if rr.Code != http.StatusTooManyRequests {
        t.Errorf("expected 429, got %d", rr.Code)
    }
}
```

## Dependencies

```bash
go get golang.org/x/time/rate
```

## Definition of Done
- [ ] Rate limiting implemented per endpoint
- [ ] IP-based limiting for agents
- [ ] API key-based limiting for tenant endpoints
- [ ] Configurable limits via environment
- [ ] 429 response with Retry-After header
- [ ] Tests pass

## Estimated Effort
4-6 hours
