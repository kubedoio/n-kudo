package controlplane

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)

	if rl == nil {
		t.Fatal("expected rate limiter to be created")
	}

	if rl.config.DefaultRate != 100.0/60.0 {
		t.Errorf("expected default rate 100/min, got %f", rl.config.DefaultRate*60)
	}

	if rl.config.DefaultBurst != 200 {
		t.Errorf("expected default burst 200, got %d", rl.config.DefaultBurst)
	}
}

func TestRateLimiter_WithDefaults(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{})

	if rl.config.DefaultRate != 100.0/60.0 {
		t.Errorf("expected default rate to be set, got %f", rl.config.DefaultRate)
	}

	if rl.config.DefaultBurst != 200 {
		t.Errorf("expected default burst to be set, got %d", rl.config.DefaultBurst)
	}
}

func TestGetClientIP(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{},
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			want:       "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 70.41.3.18, 150.172.238.178"},
			want:       "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "198.51.100.1"},
			want:       "198.51.100.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1", "X-Real-IP": "198.51.100.1"},
			want:       "203.0.113.1",
		},
		{
			name:       "IPv6 RemoteAddr",
			remoteAddr: "[::1]:12345",
			headers:    map[string]string{},
			want:       "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := rl.getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetClientID(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())

	tests := []struct {
		name       string
		apiKey     string
		adminKey   string
		remoteAddr string
		wantPrefix string
	}{
		{
			name:       "API key present",
			apiKey:     "test-api-key-123",
			remoteAddr: "192.168.1.1:12345",
			wantPrefix: "key:",
		},
		{
			name:       "Admin key present",
			adminKey:   "admin-key-456",
			remoteAddr: "192.168.1.1:12345",
			wantPrefix: "admin:",
		},
		{
			name:       "API key takes precedence over admin key",
			apiKey:     "test-api-key-123",
			adminKey:   "admin-key-456",
			remoteAddr: "192.168.1.1:12345",
			wantPrefix: "key:",
		},
		{
			name:       "IP fallback",
			remoteAddr: "192.168.1.1:12345",
			wantPrefix: "ip:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			if tt.adminKey != "" {
				req.Header.Set("X-Admin-Key", tt.adminKey)
			}

			got := rl.getClientID(req)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("getClientID() = %v, want prefix %v", got, tt.wantPrefix)
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())

	tests := []struct {
		input    string
		expected string
	}{
		{"/enroll", "/enroll"},
		{"/v1/enroll", "/v1/enroll"},
		{"/agents/heartbeat", "/agents/heartbeat"},
		{"/v1/heartbeat", "/v1/heartbeat"},
		{"/tenants", "/tenants"},
		{"/tenants/abc-123/api-keys", "/tenants/{id}/api-keys"},
		{"/tenants/xyz-789/api-keys?test=1", "/tenants/{id}/api-keys"},
		{"/unknown/path", "/unknown/path"},
		{"/healthz", "/healthz"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := rl.normalizeEndpoint(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeEndpoint(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAllow(t *testing.T) {
	// Create config with high rate for testing
	config := RateLimitConfig{
		DefaultRate:  1000, // Very high limit
		DefaultBurst: 2000,
		EndpointRates: map[string]RateLimit{
			"/test": {Rate: 10, Burst: 20},
		},
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request should be allowed
	if !rl.Allow(req) {
		t.Error("first request should be allowed")
	}

	// Many more requests should be allowed due to burst
	for i := 0; i < 15; i++ {
		if !rl.Allow(req) {
			t.Errorf("request %d should be allowed within burst", i+2)
		}
	}
}

func TestMiddleware(t *testing.T) {
	// Create config with very low rate to ensure blocking
	config := RateLimitConfig{
		DefaultRate:  0.001, // Very low rate
		DefaultBurst: 1,     // Only 1 request allowed
	}
	rl := NewRateLimiter(config)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, rr1.Code)
	}

	// Wait a tiny bit and make many requests - some should be blocked
	time.Sleep(10 * time.Millisecond)

	blocked := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			blocked++
			// Check for Retry-After header
			if rr.Header().Get("Retry-After") == "" {
				t.Error("expected Retry-After header on 429 response")
			}
		}
	}

	if blocked == 0 {
		t.Error("expected some requests to be blocked")
	}
}

func TestMiddleware_DifferentClients(t *testing.T) {
	// Create config with low burst
	config := RateLimitConfig{
		DefaultRate:  0.001,
		DefaultBurst: 2,
	}
	rl := NewRateLimiter(config)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Different IPs should have separate limits
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, ip := range ips {
		// Each IP should be able to make burst requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = ip + ":12345"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("IP %s request %d: expected status %d, got %d", ip, i+1, http.StatusOK, rr.Code)
			}
		}
	}
}

func TestMiddleware_DifferentEndpoints(t *testing.T) {
	config := RateLimitConfig{
		DefaultRate:  0.001,
		DefaultBurst: 1,
		EndpointRates: map[string]RateLimit{
			"/limited": {Rate: 0.001, Burst: 1},
		},
	}
	rl := NewRateLimiter(config)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Block one endpoint
	req1 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, rr1.Code)
	}

	// Try different endpoint - should be allowed (separate limiter)
	req2 := httptest.NewRequest(http.MethodGet, "/other", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("different endpoint request: expected status %d, got %d", http.StatusOK, rr2.Code)
	}
}

func TestGetMetrics(t *testing.T) {
	config := RateLimitConfig{
		DefaultRate:  1000,
		DefaultBurst: 2000,
	}
	rl := NewRateLimiter(config)

	// Make some requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rl.Allow(req)
	}

	hits, blocks := rl.GetMetrics()

	if hits != 5 {
		t.Errorf("expected 5 hits, got %d", hits)
	}

	if blocks != 0 {
		t.Errorf("expected 0 blocks, got %d", blocks)
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	// Check default
	if config.DefaultRate != 100.0/60.0 {
		t.Errorf("expected default rate 100/min, got %f/min", config.DefaultRate*60)
	}
	if config.DefaultBurst != 200 {
		t.Errorf("expected default burst 200, got %d", config.DefaultBurst)
	}

	// Check specific endpoints
	testCases := []struct {
		endpoint    string
		wantRate    float64
		wantBurst   int
	}{
		{"/enroll", 10.0 / 60.0, 20},
		{"/v1/enroll", 10.0 / 60.0, 20},
		{"/agents/heartbeat", 60.0 / 60.0, 120},
		{"/v1/heartbeat", 60.0 / 60.0, 120},
		{"/tenants", 5.0 / 60.0, 10},
		{"/tenants/{id}/api-keys", 10.0 / 60.0, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.endpoint, func(t *testing.T) {
			limit, ok := config.EndpointRates[tc.endpoint]
			if !ok {
				t.Errorf("endpoint %s not found in EndpointRates", tc.endpoint)
				return
			}
			if limit.Rate != tc.wantRate {
				t.Errorf("endpoint %s: expected rate %f, got %f", tc.endpoint, tc.wantRate, limit.Rate)
			}
			if limit.Burst != tc.wantBurst {
				t.Errorf("endpoint %s: expected burst %d, got %d", tc.endpoint, tc.wantBurst, limit.Burst)
			}
		})
	}
}

func TestRateLimitConfigFromEnv(t *testing.T) {
	config := RateLimitConfigFromEnv()

	// Should return default config
	if config.DefaultRate == 0 {
		t.Error("expected non-zero default rate")
	}
	if len(config.EndpointRates) == 0 {
		t.Error("expected endpoint rates to be populated")
	}
}

func TestGetLimiter_CreatesSeparateLimiters(t *testing.T) {
	config := RateLimitConfig{
		DefaultRate:  10,
		DefaultBurst: 20,
	}
	rl := NewRateLimiter(config)

	// Get limiters for different clients
	limiter1 := rl.getLimiter("client1", "/test")
	limiter2 := rl.getLimiter("client2", "/test")
	limiter3 := rl.getLimiter("client1", "/other")

	// Same client+endpoint should return same limiter
	limiter1Again := rl.getLimiter("client1", "/test")

	if limiter1 == limiter2 {
		t.Error("different clients should have different limiters")
	}

	if limiter1 == limiter3 {
		t.Error("different endpoints should have different limiters")
	}

	if limiter1 != limiter1Again {
		t.Error("same client+endpoint should return same limiter")
	}
}

func TestCleanupStaleLimiters(t *testing.T) {
	config := RateLimitConfig{
		DefaultRate:  10,
		DefaultBurst: 20,
	}
	rl := NewRateLimiter(config)

	// Create a limiter
	key := clientLimiterKey{ClientID: "test-client", Endpoint: "/test"}
	rl.getLimiter("test-client", "/test")

	// Verify it exists
	rl.mu.RLock()
	_, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		t.Fatal("limiter should exist after creation")
	}

	// Manually set last used time to be old
	rl.cleanupMu.Lock()
	rl.lastUsed[key] = time.Now().Add(-15 * time.Minute)
	rl.cleanupMu.Unlock()

	// Run cleanup
	rl.cleanupStaleLimiters()

	// Verify it was removed
	rl.mu.RLock()
	_, exists = rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		t.Error("stale limiter should have been cleaned up")
	}
}

func BenchmarkAllow(b *testing.B) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)

	req := httptest.NewRequest(http.MethodGet, "/enroll", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow(req)
	}
}

func BenchmarkMiddleware(b *testing.B) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/enroll", nil)
		req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i%100)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// Test that the limiter actually limits based on rate.Limiter behavior
func TestRateLimitEnforcement(t *testing.T) {
	// Create a limiter with rate of 10/sec and burst of 5
	limiter := rate.NewLimiter(rate.Limit(10), 5)

	// First 5 should succeed immediately (burst)
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}

	// 6th should fail (no tokens left)
	if limiter.Allow() {
		t.Error("6th request should be blocked")
	}
}

func TestMiddleware_429ResponseBody(t *testing.T) {
	config := RateLimitConfig{
		DefaultRate:  0.001,
		DefaultBurst: 1,
	}
	rl := NewRateLimiter(config)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request succeeds
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Second request should be blocked with 429
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr2.Code)
	}

	// Check response body contains error message
	body := rr2.Body.String()
	if !strings.Contains(body, "rate limit exceeded") {
		t.Errorf("expected error message in body, got: %s", body)
	}

	if !strings.Contains(body, "/test") {
		t.Errorf("expected endpoint path in error message, got: %s", body)
	}
}
