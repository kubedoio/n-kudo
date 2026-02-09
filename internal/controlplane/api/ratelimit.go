package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiting configuration for different endpoints
type RateLimitConfig struct {
	// Default rate limit (requests per second) and burst size
	DefaultRate  float64
	DefaultBurst int

	// Per-endpoint rate limits (endpoint path -> limit config)
	// Endpoint paths should be normalized (e.g., "/enroll", "/v1/enroll")
	EndpointRates map[string]RateLimit
}

// RateLimit holds the rate and burst for a specific endpoint
type RateLimit struct {
	Rate  float64 // requests per second
	Burst int     // maximum burst size
}

// clientLimiterKey uniquely identifies a limiter for a client+endpoint combination
type clientLimiterKey struct {
	ClientID string
	Endpoint string
}

// RateLimiter provides per-client, per-endpoint rate limiting
type RateLimiter struct {
	config   RateLimitConfig
	limiters map[clientLimiterKey]*rate.Limiter
	mu       sync.RWMutex

	// Cleanup tracking
	lastUsed map[clientLimiterKey]time.Time
	cleanupMu sync.Mutex

	// Metrics counters
	hitsTotal   int64
	blocksTotal int64
	muMetrics   sync.RWMutex
}

// NewRateLimiter creates a new RateLimiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	// Set defaults if not provided
	if config.DefaultRate <= 0 {
		config.DefaultRate = 100.0 / 60.0 // 100 per minute = 1.67 per second
	}
	if config.DefaultBurst <= 0 {
		config.DefaultBurst = 200
	}
	if config.EndpointRates == nil {
		config.EndpointRates = make(map[string]RateLimit)
	}

	rl := &RateLimiter{
		config:   config,
		limiters: make(map[clientLimiterKey]*rate.Limiter),
		lastUsed: make(map[clientLimiterKey]time.Time),
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// DefaultRateLimitConfig returns the default rate limit configuration
// with the specific limits defined in the requirements
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		DefaultRate:  100.0 / 60.0, // 100 per minute
		DefaultBurst: 200,
		EndpointRates: map[string]RateLimit{
			// Enrollment endpoints: 10/minute, burst 20
			"/enroll":     {Rate: 10.0 / 60.0, Burst: 20},
			"/v1/enroll":  {Rate: 10.0 / 60.0, Burst: 20},
			// Heartbeat endpoints: 60/minute, burst 120
			"/agents/heartbeat": {Rate: 60.0 / 60.0, Burst: 120},
			"/v1/heartbeat":     {Rate: 60.0 / 60.0, Burst: 120},
			// Tenant creation: 5/minute, burst 10
			"/tenants": {Rate: 5.0 / 60.0, Burst: 10},
			// API key creation: 10/minute, burst 20
			"/tenants/{id}/api-keys": {Rate: 10.0 / 60.0, Burst: 20},
		},
	}
}

// getClientID extracts a client identifier from the request
// For agent endpoints (no API key), uses IP address
// For tenant endpoints (with API key), uses hashed API key
func (rl *RateLimiter) getClientID(r *http.Request) string {
	// Check for API key first (tenant endpoints)
	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey != "" {
		// Hash the API key for privacy and consistent length
		h := sha256.Sum256([]byte(apiKey))
		return "key:" + hex.EncodeToString(h[:8]) // Use first 8 bytes of hash
	}

	// Check for admin key (admin endpoints)
	adminKey := strings.TrimSpace(r.Header.Get("X-Admin-Key"))
	if adminKey != "" {
		h := sha256.Sum256([]byte(adminKey))
		return "admin:" + hex.EncodeToString(h[:8])
	}

	// Fall back to IP address for agent endpoints
	return "ip:" + rl.getClientIP(r)
}

// getClientIP extracts the real client IP from the request
// Checks X-Forwarded-For, X-Real-IP headers, then RemoteAddr
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header (common for nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port in some cases
		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr
		}
		return "unknown"
	}
	return host
}

// normalizeEndpoint normalizes the endpoint path for rate limiting
// It handles path patterns and versioned endpoints
func (rl *RateLimiter) normalizeEndpoint(path string) string {
	// Remove query string
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Check for exact match first
	if _, ok := rl.config.EndpointRates[path]; ok {
		return path
	}

	// Check for path patterns (simple matching)
	// For /tenants/{id}/api-keys pattern
	if strings.HasPrefix(path, "/tenants/") && strings.Contains(path, "/api-keys") {
		return "/tenants/{id}/api-keys"
	}

	return path
}

// getLimiter gets or creates a rate limiter for the given client and endpoint
func (rl *RateLimiter) getLimiter(clientID, endpoint string) *rate.Limiter {
	key := clientLimiterKey{ClientID: clientID, Endpoint: endpoint}

	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		rl.cleanupMu.Lock()
		rl.lastUsed[key] = time.Now()
		rl.cleanupMu.Unlock()
		return limiter
	}

	// Create new limiter
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = rl.limiters[key]; exists {
		return limiter
	}

	// Get rate limit for this endpoint
	var limit RateLimit
	normalizedEndpoint := rl.normalizeEndpoint(endpoint)
	if l, ok := rl.config.EndpointRates[normalizedEndpoint]; ok {
		limit = l
	} else {
		limit = RateLimit{
			Rate:  rl.config.DefaultRate,
			Burst: rl.config.DefaultBurst,
		}
	}

	limiter = rate.NewLimiter(rate.Limit(limit.Rate), limit.Burst)
	rl.limiters[key] = limiter

	rl.cleanupMu.Lock()
	rl.lastUsed[key] = time.Now()
	rl.cleanupMu.Unlock()

	return limiter
}

// Allow checks if the request is allowed by the rate limiter
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(r *http.Request) bool {
	clientID := rl.getClientID(r)
	endpoint := r.URL.Path

	limiter := rl.getLimiter(clientID, endpoint)

	allowed := limiter.Allow()

	rl.muMetrics.Lock()
	if allowed {
		rl.hitsTotal++
	} else {
		rl.blocksTotal++
	}
	rl.muMetrics.Unlock()

	return allowed
}

// Middleware returns an HTTP middleware that applies rate limiting
// Returns 429 Too Many Requests if rate limit is exceeded
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(r) {
				endpoint := r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error": "rate limit exceeded for endpoint %s"}`, endpoint)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetMetrics returns the current rate limiting metrics
func (rl *RateLimiter) GetMetrics() (hitsTotal, blocksTotal int64) {
	rl.muMetrics.RLock()
	defer rl.muMetrics.RUnlock()
	return rl.hitsTotal, rl.blocksTotal
}

// cleanupLoop runs periodically to remove stale limiters
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanupStaleLimiters()
	}
}

// cleanupStaleLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) cleanupStaleLimiters() {
	staleThreshold := time.Now().Add(-10 * time.Minute)

	rl.cleanupMu.Lock()
	var toDelete []clientLimiterKey
	for key, lastUsed := range rl.lastUsed {
		if lastUsed.Before(staleThreshold) {
			toDelete = append(toDelete, key)
		}
	}
	rl.cleanupMu.Unlock()

	if len(toDelete) == 0 {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for _, key := range toDelete {
		delete(rl.limiters, key)
		delete(rl.lastUsed, key)
	}
}

// RateLimitConfigFromEnv creates a RateLimitConfig from environment variables
// Uses the defaults defined in the requirements
func RateLimitConfigFromEnv() RateLimitConfig {
	// For now, return the default configuration
	// In the future, this could read from environment variables
	return DefaultRateLimitConfig()
}

// TenantRateLimiter wraps RateLimiter with per-tenant rate limiting
type TenantRateLimiter struct {
	*RateLimiter
	tenantLimits map[string]RateLimitConfig // tenantID -> config
	tenantMu     sync.RWMutex
}

// NewTenantRateLimiter creates a new TenantRateLimiter with default limits
func NewTenantRateLimiter(config RateLimitConfig) *TenantRateLimiter {
	return &TenantRateLimiter{
		RateLimiter:  NewRateLimiter(config),
		tenantLimits: make(map[string]RateLimitConfig),
	}
}

// SetTenantLimits sets custom rate limits for a specific tenant
func (trl *TenantRateLimiter) SetTenantLimits(tenantID string, config RateLimitConfig) {
	trl.tenantMu.Lock()
	defer trl.tenantMu.Unlock()
	trl.tenantLimits[tenantID] = config
}

// GetTenantLimits returns the rate limit config for a tenant
// Returns the default config if no custom limits are set
func (trl *TenantRateLimiter) GetTenantLimits(tenantID string) RateLimitConfig {
	trl.tenantMu.RLock()
	defer trl.tenantMu.RUnlock()

	if config, ok := trl.tenantLimits[tenantID]; ok {
		return config
	}
	return trl.config
}

// AllowTenant checks if the request is allowed for the given tenant
// First checks tenant-specific limits, then falls back to default limits
func (trl *TenantRateLimiter) AllowTenant(r *http.Request, tenantID string) bool {
	// Get tenant-specific limits
	tenantConfig := trl.GetTenantLimits(tenantID)

	// Create a temporary limiter with tenant config if different from default
	clientID := trl.getClientID(r)
	endpoint := r.URL.Path

	// Use tenant-specific prefix for client ID
	tenantClientID := "tenant:" + tenantID + ":" + clientID

	limiter := trl.getLimiterWithConfig(tenantClientID, endpoint, tenantConfig)

	allowed := limiter.Allow()

	trl.muMetrics.Lock()
	if allowed {
		trl.hitsTotal++
	} else {
		trl.blocksTotal++
	}
	trl.muMetrics.Unlock()

	return allowed
}

// getLimiterWithConfig gets or creates a rate limiter with specific config
func (trl *TenantRateLimiter) getLimiterWithConfig(clientID, endpoint string, config RateLimitConfig) *rate.Limiter {
	key := clientLimiterKey{ClientID: clientID, Endpoint: endpoint}

	trl.mu.RLock()
	limiter, exists := trl.limiters[key]
	trl.mu.RUnlock()

	if exists {
		trl.cleanupMu.Lock()
		trl.lastUsed[key] = time.Now()
		trl.cleanupMu.Unlock()
		return limiter
	}

	// Create new limiter
	trl.mu.Lock()
	defer trl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = trl.limiters[key]; exists {
		return limiter
	}

	// Get rate limit for this endpoint from tenant config
	var limit RateLimit
	normalizedEndpoint := trl.normalizeEndpoint(endpoint)
	if l, ok := config.EndpointRates[normalizedEndpoint]; ok {
		limit = l
	} else {
		limit = RateLimit{
			Rate:  config.DefaultRate,
			Burst: config.DefaultBurst,
		}
	}

	limiter = rate.NewLimiter(rate.Limit(limit.Rate), limit.Burst)
	trl.limiters[key] = limiter

	trl.cleanupMu.Lock()
	trl.lastUsed[key] = time.Now()
	trl.cleanupMu.Unlock()

	return limiter
}

// MiddlewareWithTenant returns middleware that extracts tenant ID and applies tenant-specific rate limiting
func (trl *TenantRateLimiter) MiddlewareWithTenant(tenantIDFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := tenantIDFunc(r)
			if tenantID != "" {
				if !trl.AllowTenant(r, tenantID) {
					endpoint := r.URL.Path
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusTooManyRequests)
					fmt.Fprintf(w, `{"error": "rate limit exceeded for tenant %s on endpoint %s"}`, tenantID, endpoint)
					return
				}
			} else {
				// Fall back to default rate limiting if no tenant ID
				if !trl.Allow(r) {
					endpoint := r.URL.Path
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusTooManyRequests)
					fmt.Fprintf(w, `{"error": "rate limit exceeded for endpoint %s"}`, endpoint)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
