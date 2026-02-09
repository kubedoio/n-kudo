# Phase 5 Task 3: Operational Features

## Task Description
Add health checks, graceful shutdowns, and configuration hot-reload.

## Requirements

### 1. Enhanced Health Checks

**File:** `internal/controlplane/health/health.go`

```go
package health

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

type Checker struct {
    checks map[string]CheckFunc
    mu     sync.RWMutex
}

type CheckFunc func(ctx context.Context) error

type HealthStatus struct {
    Status    string            `json:"status"`
    Version   string            `json:"version"`
    Timestamp time.Time         `json:"timestamp"`
    Checks    map[string]Check  `json:"checks"`
}

type Check struct {
    Status    string        `json:"status"`
    Latency   time.Duration `json:"latency_ms"`
    Error     string        `json:"error,omitempty"`
}

func NewChecker(version string) *Checker {
    return &Checker{
        checks: make(map[string]CheckFunc),
        version: version,
    }
}

func (c *Checker) Register(name string, fn CheckFunc) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.checks[name] = fn
}

func (c *Checker) Check(ctx context.Context) HealthStatus {
    c.mu.RLock()
    checks := make(map[string]CheckFunc, len(c.checks))
    for k, v := range c.checks {
        checks[k] = v
    }
    c.mu.RUnlock()
    
    status := HealthStatus{
        Status:    "healthy",
        Version:   c.version,
        Timestamp: time.Now().UTC(),
        Checks:    make(map[string]Check),
    }
    
    for name, fn := range checks {
        start := time.Now()
        err := fn(ctx)
        latency := time.Since(start)
        
        check := Check{
            Status:  "pass",
            Latency: latency,
        }
        
        if err != nil {
            check.Status = "fail"
            check.Error = err.Error()
            status.Status = "degraded"
        }
        
        status.Checks[name] = check
    }
    
    return status
}
```

### 2. Health Check Endpoints

**File:** `internal/controlplane/api/server.go`

```go
func (a *App) registerRoutes() {
    // Basic health (liveness)
    a.mux.HandleFunc("GET /healthz", a.handleHealthz)
    
    // Detailed health (readiness)
    a.mux.HandleFunc("GET /readyz", a.handleReadyz)
}

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]string{
        "status": "ok",
    })
}

func (a *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
    status := a.healthChecker.Check(r.Context())
    
    code := http.StatusOK
    if status.Status == "degraded" {
        code = http.StatusServiceUnavailable
    }
    
    writeJSON(w, code, status)
}
```

### 3. Graceful Shutdown

**File:** `cmd/control-plane/main.go`

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    
    server := &http.Server{
        Addr:    cfg.BindAddr,
        Handler: app.Handler(),
    }
    
    // Start server in goroutine
    go func() {
        log.Printf("Server starting on %s", cfg.BindAddr)
        if err := server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    // Wait for shutdown signal
    <-ctx.Done()
    log.Println("Shutdown signal received, initiating graceful shutdown...")
    
    // Graceful shutdown with timeout
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("Graceful shutdown failed: %v", err)
        server.Close()
    }
    
    // Close database connections
    if err := repo.Close(); err != nil {
        log.Printf("Error closing database: %v", err)
    }
    
    log.Println("Shutdown complete")
}
```

### 4. Edge Agent Graceful Shutdown

**File:** `cmd/edge/main.go` in `runService`

```go
func runService(ctx context.Context, args []string) error {
    // ... setup code ...
    
    // Create cancelable context for graceful shutdown
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    // Handle shutdown signals
    go func() {
        <-ctx.Done()
        log.Println("Agent shutting down...")
        
        // Stop VMs gracefully
        vms, _ := st.ListMicroVMs()
        for _, vm := range vms {
            if vm.Status == "running" {
                log.Printf("Stopping VM %s...", vm.ID)
                // Send graceful stop
            }
        }
        
        // Send final heartbeat
        cp.Heartbeat(context.Background(), enroll.HeartbeatRequest{
            AgentID: id.AgentID,
            // ... mark as shutting down
        })
    }()
    
    // Main loop
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-time.After(*interval):
            if err := loop(); err != nil {
                log.Printf("Loop error: %v", err)
            }
        }
    }
}
```

### 5. Configuration Hot-Reload

**File:** `internal/controlplane/config/watcher.go`

```go
package config

import (
    "context"
    "log"
    "os"
    "sync"
    "time"
)

type Watcher struct {
    path      string
    onChange  func()
    lastMod   time.Time
    mu        sync.RWMutex
    stopCh    chan struct{}
}

func NewWatcher(path string, onChange func()) *Watcher {
    return &Watcher{
        path:     path,
        onChange: onChange,
        stopCh:   make(chan struct{}),
    }
}

func (w *Watcher) Start(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-w.stopCh:
            return
        case <-ticker.C:
            w.check()
        }
    }
}

func (w *Watcher) check() {
    info, err := os.Stat(w.path)
    if err != nil {
        return
    }
    
    w.mu.RLock()
    lastMod := w.lastMod
    w.mu.RUnlock()
    
    if info.ModTime().After(lastMod) {
        w.mu.Lock()
        w.lastMod = info.ModTime()
        w.mu.Unlock()
        
        log.Printf("Config file changed, reloading...")
        w.onChange()
    }
}

func (w *Watcher) Stop() {
    close(w.stopCh)
}
```

### 6. Readiness Gates

Add readiness checks for:
- Database connection
- CA certificate loaded
- CRL manager initialized

## Deliverables
1. `internal/controlplane/health/health.go` - Health check framework
2. `internal/controlplane/config/watcher.go` - Config hot-reload
3. Updated `cmd/control-plane/main.go` - Graceful shutdown
4. Updated `cmd/edge/main.go` - Agent graceful shutdown
5. Updated `internal/controlplane/api/server.go` - Health endpoints
6. Health check documentation

## Estimated Effort
6-8 hours
