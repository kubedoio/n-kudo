package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker("v1.0.0")
	if c == nil {
		t.Fatal("expected checker to be created")
	}
	if c.version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", c.version)
	}
	if len(c.checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(c.checks))
	}

	// Test default version
	c2 := NewChecker("")
	if c2.version != "dev" {
		t.Errorf("expected default version 'dev', got %s", c2.version)
	}
}

func TestRegisterAndUnregister(t *testing.T) {
	c := NewChecker("test")

	checkFn := func(ctx context.Context) error { return nil }
	c.Register("test-check", checkFn)

	if len(c.checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(c.checks))
	}

	c.Unregister("test-check")
	if len(c.checks) != 0 {
		t.Errorf("expected 0 checks after unregister, got %d", len(c.checks))
	}
}

func TestCheckHealthy(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("always-healthy", func(ctx context.Context) error {
		return nil
	})

	status := c.Check(context.Background())

	if status.Status != StatusHealthy {
		t.Errorf("expected status healthy, got %s", status.Status)
	}
	if status.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", status.Version)
	}
	if len(status.Checks) != 1 {
		t.Errorf("expected 1 check result, got %d", len(status.Checks))
	}

	check, ok := status.Checks["always-healthy"]
	if !ok {
		t.Fatal("expected always-healthy check in results")
	}
	if check.Status != StatusHealthy {
		t.Errorf("expected check status healthy, got %s", check.Status)
	}
	if check.Error != "" {
		t.Errorf("expected no error, got %s", check.Error)
	}
}

func TestCheckUnhealthy(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("always-fails", func(ctx context.Context) error {
		return errors.New("check failed")
	})

	status := c.Check(context.Background())

	if status.Status != StatusUnhealthy {
		t.Errorf("expected status unhealthy, got %s", status.Status)
	}

	check, ok := status.Checks["always-fails"]
	if !ok {
		t.Fatal("expected always-fails check in results")
	}
	if check.Status != StatusUnhealthy {
		t.Errorf("expected check status unhealthy, got %s", check.Status)
	}
	if check.Error != "check failed" {
		t.Errorf("expected error 'check failed', got %s", check.Error)
	}
}

func TestCheckMixed(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("healthy", func(ctx context.Context) error {
		return nil
	})
	c.Register("unhealthy", func(ctx context.Context) error {
		return errors.New("failed")
	})

	status := c.Check(context.Background())

	if status.Status != StatusUnhealthy {
		t.Errorf("expected status unhealthy when one check fails, got %s", status.Status)
	}
	if len(status.Checks) != 2 {
		t.Errorf("expected 2 check results, got %d", len(status.Checks))
	}
}

func TestCheckConcurrent(t *testing.T) {
	c := NewChecker("v1.0.0")

	var callCount int
	var mu sync.Mutex

	// Add multiple checks that track concurrent execution
	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		c.Register(name, func(ctx context.Context) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	start := time.Now()
	status := c.Check(context.Background())
	duration := time.Since(start)

	if status.Status != StatusHealthy {
		t.Errorf("expected status healthy, got %s", status.Status)
	}

	// If checks run sequentially, it would take at least 50ms
	// If they run concurrently, it should take around 10-20ms
	if duration > 40*time.Millisecond {
		t.Errorf("checks appear to run sequentially (took %v), expected concurrent execution", duration)
	}

	mu.Lock()
	if callCount != 5 {
		t.Errorf("expected 5 check calls, got %d", callCount)
	}
	mu.Unlock()
}

func TestCheckTimeout(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("slow-check", func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	status := c.Check(ctx)

	// The check should fail due to timeout
	if status.Status != StatusUnhealthy {
		t.Errorf("expected status unhealthy due to timeout, got %s", status.Status)
	}

	check, ok := status.Checks["slow-check"]
	if !ok {
		t.Fatal("expected slow-check in results")
	}
	if check.Status != StatusUnhealthy {
		t.Errorf("expected check status unhealthy, got %s", check.Status)
	}
}

func TestIsHealthy(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("healthy", func(ctx context.Context) error {
		return nil
	})

	if !c.IsHealthy(context.Background()) {
		t.Error("expected IsHealthy to return true")
	}

	c.Register("unhealthy", func(ctx context.Context) error {
		return errors.New("failed")
	})

	if c.IsHealthy(context.Background()) {
		t.Error("expected IsHealthy to return false after adding failing check")
	}
}

func TestCheckLatency(t *testing.T) {
	c := NewChecker("v1.0.0")

	c.Register("slow-check", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	status := c.Check(context.Background())

	check := status.Checks["slow-check"]
	if check.Latency < 50*time.Millisecond {
		t.Errorf("expected latency >= 50ms, got %v", check.Latency)
	}
}

func TestLivenessCheck(t *testing.T) {
	ctx := context.Background()
	if err := LivenessCheck(ctx); err != nil {
		t.Errorf("expected liveness check to always pass, got error: %v", err)
	}
}

func TestTimestamp(t *testing.T) {
	c := NewChecker("v1.0.0")
	c.Register("test", func(ctx context.Context) error { return nil })

	before := time.Now().UTC()
	status := c.Check(context.Background())
	after := time.Now().UTC()

	if status.Timestamp.Before(before) || status.Timestamp.After(after) {
		t.Errorf("timestamp %v not within expected range [%v, %v]", status.Timestamp, before, after)
	}

	for _, check := range status.Checks {
		if check.CheckedAt.Before(before) || check.CheckedAt.After(after) {
			t.Errorf("check timestamp %v not within expected range", check.CheckedAt)
		}
	}
}
