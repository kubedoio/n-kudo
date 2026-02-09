# Phase 5 Task 5: Monitoring & Alerting

## Task Description
Integrate Alertmanager, add SLA metrics, and implement error budget tracking.

## Requirements

### 1. Prometheus Alerting Rules

**File:** `deployments/monitoring/alerts.yml`

```yaml
groups:
  - name: nkudo-alerts
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(nkudo_http_requests_total{status=~"5.."}[5m])) 
            / 
            sum(rate(nkudo_http_requests_total[5m]))
          ) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} over the last 5 minutes"
      
      # Agent offline
      - alert: AgentsOffline
        expr: nkudo_agents_offline_total > 5
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Multiple agents offline"
          description: "{{ $value }} agents have been offline for more than 10 minutes"
      
      # Certificate expiry
      - alert: CertificateExpiringSoon
        expr: |
          nkudo_agent_certificate_expiry_timestamp - time() < 86400 * 7
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Agent certificate expiring soon"
          description: "Certificate expires in less than 7 days"
      
      # Heartbeat failures
      - alert: HeartbeatFailures
        expr: rate(nkudo_heartbeat_failures_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High heartbeat failure rate"
          description: "Heartbeat failure rate is {{ $value }} per second"
      
      # Database connection issues
      - alert: DatabaseConnectionErrors
        expr: nkudo_db_connection_errors_total > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database connection errors"
          description: "Unable to connect to database"
```

### 2. SLA Metrics

**File:** `internal/controlplane/metrics/sla.go`

```go
package metrics

import (
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
)

var (
    // Availability tracking
    SLAUptime = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_sla_uptime_seconds_total",
        Help: "Total seconds of uptime per service",
    }, []string{"service"})
    
    SLADowntime = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_sla_downtime_seconds_total",
        Help: "Total seconds of downtime per service",
    }, []string{"service"})
    
    // Error budget
    ErrorBudgetTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_error_budget_total",
        Help: "Total error budget (in seconds) for the period",
    }, []string{"service", "slo"})
    
    ErrorBudgetUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "nkudo_error_budget_used_seconds",
        Help: "Error budget consumed (in seconds)",
    }, []string{"service", "slo"})
    
    // SLO tracking
    SLOStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "nkudo_slo_status",
        Help: "Current SLO status (1 = healthy, 0 = at risk, -1 = breached)",
    }, []string{"service", "slo"})
)

type SLAMonitor struct {
    service   string
    sloTarget float64 // e.g., 0.999 for 99.9%
}

func (s *SLAMonitor) RecordAvailability(success bool, duration time.Duration) {
    if success {
        SLAUptime.WithLabelValues(s.service).Add(duration.Seconds())
    } else {
        SLADowntime.WithLabelValues(s.service).Add(duration.Seconds())
    }
}

func (s *SLAMonitor) CheckErrorBudget() float64 {
    // Calculate remaining error budget
    total := 30 * 24 * time.Hour // 30 days
    allowedErrors := time.Duration((1 - s.sloTarget) * float64(total))
    
    // Get used budget from metrics
    // ... implementation
    
    return allowedErrors.Seconds()
}
```

### 3. Alertmanager Integration

**File:** `deployments/monitoring/alertmanager.yml`

```yaml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@nkudo.io'

route:
  receiver: 'default'
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty-critical'
      continue: true
    
    - match:
        severity: warning
      receiver: 'slack-warnings'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@nkudo.io'
  
  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: '<pagerduty-key>'
        severity: critical
  
  - name: 'slack-warnings'
    slack_configs:
      - api_url: '<slack-webhook>'
        channel: '#alerts'
        title: 'Nkudo Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

### 4. Custom Metrics Endpoint

**File:** `internal/controlplane/api/metrics.go`

```go
package api

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
    // Add custom runtime metrics
    updateRuntimeMetrics(a)
    
    // Serve Prometheus metrics
    promhttp.Handler().ServeHTTP(w, r)
}

func updateRuntimeMetrics(a *App) {
    // Update agent counts
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if counts, err := a.repo.GetAgentCounts(ctx); err == nil {
        AgentsTotal.Set(float64(counts.Total))
        AgentsOnline.Set(float64(counts.Online))
        AgentsOffline.Set(float64(counts.Offline))
    }
}

var (
    AgentsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "nkudo_agents_total",
        Help: "Total number of agents",
    })
    
    AgentsOnline = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "nkudo_agents_online",
        Help: "Number of online agents",
    })
    
    AgentsOffline = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "nkudo_agents_offline",
        Help: "Number of offline agents",
    })
)
```

### 5. Health Dashboard JSON

**File:** `deployments/monitoring/grafana-dashboard.json`

Complete Grafana dashboard with panels for:
- Agent status overview
- API request rate/latency/errors
- Database performance
- Certificate expiry timeline
- SLO compliance
- Error budget burn rate

### 6. Alert Testing Script

**File:** `scripts/test-alerts.sh`

```bash
#!/bin/bash
# Test alerting pipeline

echo "Testing alert firing..."
curl -X POST http://localhost:9093/-/reload

# Simulate high error rate
echo "Simulating errors..."
for i in {1..100}; do
    curl -s http://localhost:8080/invalid > /dev/null
done

echo "Check Alertmanager at http://localhost:9093"
```

## Deliverables
1. `deployments/monitoring/alerts.yml` - Prometheus alerting rules
2. `deployments/monitoring/alertmanager.yml` - Alertmanager config
3. `deployments/monitoring/grafana-dashboard.json` - Grafana dashboard
4. `internal/controlplane/metrics/sla.go` - SLA tracking
5. `internal/controlplane/api/metrics.go` - Custom metrics
6. `scripts/test-alerts.sh` - Alert testing

## Dependencies
Add to docker-compose:
```yaml
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./deployments/monitoring/alerts.yml:/etc/prometheus/alerts.yml
      - prometheus-data:/prometheus
    
  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./deployments/monitoring/alertmanager.yml:/etc/alertmanager/config.yml
      
  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./deployments/monitoring/grafana-dashboard.json:/var/lib/grafana/dashboards/nkudo.json
```

## Estimated Effort
6-8 hours
