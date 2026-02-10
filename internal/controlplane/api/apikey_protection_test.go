package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIKeyProtector_IsBlocked(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    1 * time.Minute,
		BlockDuration:     1 * time.Minute,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	ip := "192.168.1.1"

	// Initially not blocked
	if protector.IsBlocked(ip) {
		t.Errorf("expected IP %s to not be blocked initially", ip)
	}

	// Record failures up to limit
	for i := 0; i < 2; i++ {
		blocked := protector.RecordFailure(ip)
		if blocked {
			t.Errorf("expected IP %s to not be blocked after %d failures", ip, i+1)
		}
		if protector.IsBlocked(ip) {
			t.Errorf("expected IP %s to not be blocked after %d failures", ip, i+1)
		}
	}

	// One more failure should trigger block
	blocked := protector.RecordFailure(ip)
	if !blocked {
		t.Errorf("expected IP %s to be blocked after %d failures", ip, 3)
	}
	if !protector.IsBlocked(ip) {
		t.Errorf("expected IP %s to be blocked after %d failures", ip, 3)
	}
}

func TestAPIKeyProtector_RecordSuccessClearsFailures(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    1 * time.Minute,
		BlockDuration:     1 * time.Minute,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	ip := "192.168.1.2"

	// Record some failures
	protector.RecordFailure(ip)
	protector.RecordFailure(ip)

	// Record success should clear
	protector.RecordSuccess(ip)

	// Should not be blocked
	if protector.IsBlocked(ip) {
		t.Errorf("expected IP %s to not be blocked after success", ip)
	}

	// Need 3 more failures to block again
	for i := 0; i < 3; i++ {
		protector.RecordFailure(ip)
	}

	if !protector.IsBlocked(ip) {
		t.Errorf("expected IP %s to be blocked after new failures", ip)
	}
}

func TestAPIKeyProtector_WindowExpiry(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    100 * time.Millisecond,
		BlockDuration:     1 * time.Minute,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	ip := "192.168.1.3"

	// Record some failures
	protector.RecordFailure(ip)
	protector.RecordFailure(ip)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Record another failure - should reset window
	blocked := protector.RecordFailure(ip)
	if blocked {
		t.Errorf("expected window to have expired, not blocked")
	}

	// Check that count was reset to 1
	isBlocked, _, failedCount := protector.GetBlockInfo(ip)
	if isBlocked {
		t.Errorf("expected not to be blocked")
	}
	if failedCount != 1 {
		t.Errorf("expected failed count to be 1 after window reset, got %d", failedCount)
	}
}

func TestAPIKeyProtector_BlockExpiry(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    1 * time.Minute,
		BlockDuration:     100 * time.Millisecond,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	ip := "192.168.1.4"

	// Block the IP
	for i := 0; i < 3; i++ {
		protector.RecordFailure(ip)
	}

	if !protector.IsBlocked(ip) {
		t.Errorf("expected IP to be blocked")
	}

	// Wait for block to expire
	time.Sleep(150 * time.Millisecond)

	// Should no longer be blocked
	if protector.IsBlocked(ip) {
		t.Errorf("expected block to have expired")
	}
}

func TestAPIKeyProtector_DifferentIPs(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    1 * time.Minute,
		BlockDuration:     1 * time.Minute,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	ip1 := "192.168.1.5"
	ip2 := "192.168.1.6"

	// Block ip1
	for i := 0; i < 3; i++ {
		protector.RecordFailure(ip1)
	}

	// ip2 should not be affected
	if protector.IsBlocked(ip2) {
		t.Errorf("expected IP %s to not be blocked", ip2)
	}

	// ip2 should still be able to fail without being blocked
	for i := 0; i < 2; i++ {
		blocked := protector.RecordFailure(ip2)
		if blocked {
			t.Errorf("expected IP %s to not be blocked after %d failures", ip2, i+1)
		}
	}
}

func TestAPIKeyProtectionGetClientIP(t *testing.T) {
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
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3"},
			want:       "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Real-IP": "10.0.0.2"},
			want:       "10.0.0.2",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "10.0.0.2"},
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKeyProtector_Middleware(t *testing.T) {
	config := APIKeyProtectionConfig{
		MaxFailedAttempts: 3,
		WindowDuration:    1 * time.Minute,
		BlockDuration:     1 * time.Minute,
		CleanupInterval:   1 * time.Hour,
	}
	protector := NewAPIKeyProtector(config)
	defer protector.Stop()

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrapped := protector.Middleware()(handler)

	ip := "192.168.1.7"

	// Block the IP first
	for i := 0; i < 3; i++ {
		protector.RecordFailure(ip)
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":12345"
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	body := rr.Body.String()
	if body == "" {
		t.Errorf("expected error message in body")
	}
}
