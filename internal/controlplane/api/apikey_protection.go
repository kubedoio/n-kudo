package controlplane

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// APIKeyBlockedAttemptsTotal tracks the number of blocked API key authentication attempts
	APIKeyBlockedAttemptsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "nkudo",
		Name:      "api_key_blocked_attempts_total",
		Help:      "Total number of blocked API key authentication attempts due to failed attempt limiting",
	}, []string{"ip_address"})
)

func init() {
	prometheus.MustRegister(APIKeyBlockedAttemptsTotal)
}

// APIKeyProtectionConfig holds configuration for API key failed attempt protection
type APIKeyProtectionConfig struct {
	// MaxFailedAttempts is the number of failed attempts before blocking (default: 5)
	MaxFailedAttempts int
	// WindowDuration is the time window for counting failed attempts (default: 15 minutes)
	WindowDuration time.Duration
	// BlockDuration is how long to block the IP after exceeding max attempts (default: 30 minutes)
	BlockDuration time.Duration
	// CleanupInterval is how often to run cleanup of old entries (default: 5 minutes)
	CleanupInterval time.Duration
}

// DefaultAPIKeyProtectionConfig returns the default configuration
func DefaultAPIKeyProtectionConfig() APIKeyProtectionConfig {
	return APIKeyProtectionConfig{
		MaxFailedAttempts: 5,
		WindowDuration:    15 * time.Minute,
		BlockDuration:     30 * time.Minute,
		CleanupInterval:   5 * time.Minute,
	}
}

// apiKeyAttempt tracks failed authentication attempts for a client IP
type apiKeyAttempt struct {
	FailedCount    int
	FirstAttemptAt time.Time
	BlockedUntil   *time.Time
}

// APIKeyProtector provides failed attempt limiting for API key authentication
type APIKeyProtector struct {
	config   APIKeyProtectionConfig
	attempts map[string]*apiKeyAttempt // key: IP address
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewAPIKeyProtector creates a new API key protector with the given configuration
func NewAPIKeyProtector(config APIKeyProtectionConfig) *APIKeyProtector {
	// Set defaults for any zero values
	if config.MaxFailedAttempts <= 0 {
		config.MaxFailedAttempts = 5
	}
	if config.WindowDuration <= 0 {
		config.WindowDuration = 15 * time.Minute
	}
	if config.BlockDuration <= 0 {
		config.BlockDuration = 30 * time.Minute
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	ap := &APIKeyProtector{
		config:   config,
		attempts: make(map[string]*apiKeyAttempt),
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	go ap.cleanupLoop()

	return ap
}

// Stop stops the background cleanup goroutine
func (ap *APIKeyProtector) Stop() {
	close(ap.stopCh)
}

// IsBlocked checks if the client IP is currently blocked
func (ap *APIKeyProtector) IsBlocked(ip string) bool {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	attempt, exists := ap.attempts[ip]
	if !exists {
		return false
	}

	// Check if block has expired
	if attempt.BlockedUntil != nil {
		if time.Now().After(*attempt.BlockedUntil) {
			return false
		}
		return true
	}

	return false
}

// RecordFailure records a failed API key attempt for the client IP
// Returns true if the IP is now blocked
func (ap *APIKeyProtector) RecordFailure(ip string) bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	now := time.Now()
	attempt, exists := ap.attempts[ip]

	if !exists {
		// First failed attempt for this IP
		ap.attempts[ip] = &apiKeyAttempt{
			FailedCount:    1,
			FirstAttemptAt: now,
		}
		return false
	}

	// Check if the window has expired - reset if so
	if now.After(attempt.FirstAttemptAt.Add(ap.config.WindowDuration)) {
		attempt.FailedCount = 1
		attempt.FirstAttemptAt = now
		attempt.BlockedUntil = nil
		return false
	}

	// Increment failed count
	attempt.FailedCount++

	// Check if we should block
	if attempt.FailedCount >= ap.config.MaxFailedAttempts {
		blockedUntil := now.Add(ap.config.BlockDuration)
		attempt.BlockedUntil = &blockedUntil

		// Log security event
		log.Printf("[SECURITY] IP %s blocked for %v after %d failed API key attempts",
			ip, ap.config.BlockDuration, attempt.FailedCount)

		// Increment metrics counter
		APIKeyBlockedAttemptsTotal.WithLabelValues(ip).Inc()

		return true
	}

	return false
}

// RecordSuccess clears failed attempts for the client IP on successful authentication
func (ap *APIKeyProtector) RecordSuccess(ip string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	delete(ap.attempts, ip)
}

// GetBlockInfo returns blocking information for an IP (for debugging/metrics)
// Returns (isBlocked, blockedUntil, failedCount)
func (ap *APIKeyProtector) GetBlockInfo(ip string) (bool, *time.Time, int) {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	attempt, exists := ap.attempts[ip]
	if !exists {
		return false, nil, 0
	}

	if attempt.BlockedUntil != nil && time.Now().Before(*attempt.BlockedUntil) {
		return true, attempt.BlockedUntil, attempt.FailedCount
	}

	return false, nil, attempt.FailedCount
}

// cleanupLoop runs periodically to remove stale entries
func (ap *APIKeyProtector) cleanupLoop() {
	ticker := time.NewTicker(ap.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ap.cleanupStaleEntries()
		case <-ap.stopCh:
			return
		}
	}
}

// cleanupStaleEntries removes entries that are no longer relevant
func (ap *APIKeyProtector) cleanupStaleEntries() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	now := time.Now()
	var toDelete []string

	for ip, attempt := range ap.attempts {
		// Remove if block has expired and window has passed
		if attempt.BlockedUntil != nil {
			if now.After(*attempt.BlockedUntil) {
				// Block expired, also check if window has passed
				if now.After(attempt.FirstAttemptAt.Add(ap.config.WindowDuration)) {
					toDelete = append(toDelete, ip)
				}
			}
		} else {
			// No block, check if window has passed
			if now.After(attempt.FirstAttemptAt.Add(ap.config.WindowDuration)) {
				toDelete = append(toDelete, ip)
			}
		}
	}

	for _, ip := range toDelete {
		delete(ap.attempts, ip)
	}

	if len(toDelete) > 0 {
		log.Printf("[SECURITY] Cleaned up %d stale API key protection entries", len(toDelete))
	}
}

// getClientIP extracts the real client IP from the request
// Checks X-Forwarded-For, X-Real-IP headers, then RemoteAddr
func getClientIP(r *http.Request) string {
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

// Middleware returns an HTTP middleware that checks if the client is blocked
// before allowing the request to proceed. This should be used BEFORE the
// apiKeyAuth middleware.
func (ap *APIKeyProtector) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if ap.IsBlocked(ip) {
				_, blockedUntil, _ := ap.GetBlockInfo(ip)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				msg := "API key authentication blocked due to too many failed attempts"
				if blockedUntil != nil {
					msg = "API key authentication blocked due to too many failed attempts. Try again after " + blockedUntil.Format(time.RFC3339)
				}

				writeJSON(w, http.StatusForbidden, map[string]any{
					"error": msg,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
