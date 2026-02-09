# Phase 3 Task 2: Edge Agent Observability

## Task Description
Add observability features to the edge agent: Prometheus metrics, structured logging, and execution tracking.

## Features

### 1. Prometheus Metrics Endpoint

**Endpoint:** `http://localhost:9090/metrics`

**Metrics to expose:**

```
# VM metrics
nkudo_vms_total{state="running"} 5
nkudo_vms_total{state="stopped"} 2
nkudo_vm_cpu_cores{vm_id="xxx",vm_name="web-1"} 2
nkudo_vm_memory_bytes{vm_id="xxx",vm_name="web-1"} 536870912

# Action execution metrics
nkudo_actions_executed_total{action_type="MicroVMCreate",status="success"} 10
nkudo_actions_executed_total{action_type="MicroVMCreate",status="failure"} 1
nkudo_actions_duration_seconds{action_type="MicroVMCreate"} 15.5

# Heartbeat metrics
nkudo_heartbeats_sent_total 100
nkudo_heartbeat_duration_seconds 0.15
nkudo_heartbeat_failures_total 2

# System metrics
nkudo_disk_usage_bytes{path="/var/lib/nkudo-edge"} 10737418240
nkudo_disk_total_bytes{path="/var/lib/nkudo-edge"} 107374182400
nkudo_host_cpu_usage_percent 25.5
nkudo_host_memory_usage_bytes 8589934592
```

**Implementation:**

Create `internal/edge/metrics/metrics.go`:
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    VMsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "nkudo_vms_total",
        Help: "Total number of VMs by state",
    }, []string{"state"})
    
    ActionsExecuted = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_actions_executed_total",
        Help: "Total actions executed",
    }, []string{"action_type", "status"})
    
    ActionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "nkudo_actions_duration_seconds",
        Help: "Action execution duration",
    }, []string{"action_type"})
)

func init() {
    prometheus.MustRegister(VMsTotal, ActionsExecuted, ActionDuration)
}

func StartServer(addr string) error {
    http.Handle("/metrics", promhttp.Handler())
    return http.ListenAndServe(addr, nil)
}
```

### 2. Structured JSON Logging

**Format:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "executor",
  "message": "VM created successfully",
  "fields": {
    "vm_id": "550e8400-e29b-41d4-a716-446655440000",
    "vm_name": "web-server-01",
    "duration_ms": 15000
  }
}
```

**Configuration:**
- `--log-format=json|text` (default: text)
- `--log-level=debug|info|warn|error` (default: info)

**Implementation:**

Create `internal/edge/logger/logger.go`:
```go
package logger

import (
    "github.com/sirupsen/logrus"
)

var log *logrus.Logger

func Init(format, level string) {
    log = logrus.New()
    
    if format == "json" {
        log.SetFormatter(&logrus.JSONFormatter{})
    } else {
        log.SetFormatter(&logrus.TextFormatter{
            FullTimestamp: true,
        })
    }
    
    lvl, _ := logrus.ParseLevel(level)
    log.SetLevel(lvl)
}

func WithFields(fields map[string]interface{}) *logrus.Entry {
    return log.WithFields(fields)
}

func Info(msg string) { log.Info(msg) }
func Error(msg string) { log.Error(msg) }
func Debug(msg string) { log.Debug(msg) }
// ... etc
```

### 3. Execution Tracking

**Track per-execution metrics:**
- Execution duration
- Resource usage (CPU, memory)
- Success/failure rates
- Retry counts

**Storage:**
Write to `edge-state.json`:
```json
{
  "executions": [
    {
      "action_id": "action-1",
      "action_type": "MicroVMCreate",
      "started_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:30:15Z",
      "duration_ms": 15000,
      "status": "success",
      "error": null,
      "resource_usage": {
        "cpu_percent": 25.5,
        "memory_bytes": 1073741824
      }
    }
  ]
}
```

**Implementation:**

Modify `internal/edge/executor/executor.go`:
```go
func (e *Executor) Execute(ctx context.Context, action Action) Result {
    start := time.Now()
    
    // Record start
    e.recordExecutionStart(action)
    
    // Execute
    result := e.executeInternal(ctx, action)
    
    // Record completion
    duration := time.Since(start)
    e.recordExecutionComplete(action, result, duration)
    
    // Update metrics
    metrics.ActionDuration.WithLabelValues(action.Type).Observe(duration.Seconds())
    status := "success"
    if result.Error != "" {
        status = "failure"
    }
    metrics.ActionsExecuted.WithLabelValues(action.Type, status).Inc()
    
    return result
}
```

## Files to Modify

### New Files
- `internal/edge/metrics/metrics.go` - Prometheus metrics
- `internal/edge/logger/logger.go` - Structured logging
- `internal/edge/executor/tracker.go` - Execution tracking

### Modified Files
- `cmd/edge/main.go` - Add log flags, start metrics server
- `internal/edge/executor/executor.go` - Integrate metrics/logging
- `internal/edge/state/store.go` - Add execution history storage

## Dependencies

Add to `go.mod`:
```
github.com/prometheus/client_golang v1.18.0
github.com/sirupsen/logrus v1.9.3
```

## Testing

Add tests:
- `internal/edge/metrics/metrics_test.go` - Verify metrics exported correctly
- `internal/edge/logger/logger_test.go` - Verify log format
- `internal/edge/executor/tracker_test.go` - Verify execution tracking

## Definition of Done
- [ ] Prometheus endpoint serves metrics
- [ ] JSON logging format works
- [ ] Log level filtering works
- [ ] Execution duration tracked
- [ ] All tests pass

## Estimated Effort
6-8 hours
