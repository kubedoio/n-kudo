package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckFunc is a function that performs a health check
// It returns an error if the check fails
// The context is passed for cancellation/timeout support
type CheckFunc func(ctx context.Context) error

// CheckResult contains the result of a single health check
type CheckResult struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Error     string        `json:"error,omitempty"`
	Latency   time.Duration `json:"latency_ms"`
	CheckedAt time.Time     `json:"checked_at"`
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    Status                 `json:"status"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Checker manages health checks
type Checker struct {
	checks  map[string]CheckFunc
	version string
	mu      sync.RWMutex
}

// NewChecker creates a new health checker with the given version
func NewChecker(version string) *Checker {
	if version == "" {
		version = "dev"
	}
	return &Checker{
		checks:  make(map[string]CheckFunc),
		version: version,
	}
}

// Register adds a health check with the given name
// If a check with the same name already exists, it will be overwritten
func (c *Checker) Register(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// Unregister removes a health check
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.checks, name)
}

// Check runs all registered health checks concurrently
// Returns a HealthStatus containing all check results
// The overall status is:
//   - "healthy" if all checks pass
//   - "unhealthy" if any critical check fails
func (c *Checker) Check(ctx context.Context) HealthStatus {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.checks))
	for name, fn := range c.checks {
		checks[name] = fn
	}
	c.mu.RUnlock()

	results := make(map[string]CheckResult, len(checks))
	var wg sync.WaitGroup
	var mu sync.Mutex

	overallStatus := StatusHealthy

	for name, fn := range checks {
		wg.Add(1)
		go func(name string, fn CheckFunc) {
			defer wg.Done()

			start := time.Now()
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			err := fn(checkCtx)
			latency := time.Since(start)

			result := CheckResult{
				Name:      name,
				Latency:   latency,
				CheckedAt: time.Now().UTC(),
			}

			if err != nil {
				result.Status = StatusUnhealthy
				result.Error = err.Error()
				mu.Lock()
				overallStatus = StatusUnhealthy
				mu.Unlock()
			} else {
				result.Status = StatusHealthy
			}

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, fn)
	}

	wg.Wait()

	return HealthStatus{
		Status:    overallStatus,
		Version:   c.version,
		Timestamp: time.Now().UTC(),
		Checks:    results,
	}
}

// IsHealthy returns true if the overall health status is healthy
func (c *Checker) IsHealthy(ctx context.Context) bool {
	status := c.Check(ctx)
	return status.Status == StatusHealthy
}

// LivenessCheck is a simple check that always returns healthy
// Used for Kubernetes liveness probes
func LivenessCheck(ctx context.Context) error {
	return nil
}
