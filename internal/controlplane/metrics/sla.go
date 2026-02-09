// Package sla provides SLA (Service Level Agreement) and SLO (Service Level Objective)
// monitoring metrics for the n-kudo control plane.
package sla

import (
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
)

// SLOStatus represents the health status of an SLO.
type SLOStatus int

const (
	// SLOStatusBreached indicates the error budget has been exhausted.
	SLOStatusBreached SLOStatus = -1
	// SLOStatusAtRisk indicates the error budget is at risk of being exhausted.
	SLOStatusAtRisk SLOStatus = 0
	// SLOStatusHealthy indicates the SLO is healthy.
	SLOStatusHealthy SLOStatus = 1
)

var (
	// SLAUptime tracks total uptime in seconds by service.
	SLAUptime = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "nkudo",
		Name:      "sla_uptime_seconds_total",
		Help:      "Total uptime in seconds by service",
	}, []string{"service"})

	// SLADowntime tracks total downtime in seconds by service.
	SLADowntime = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "nkudo",
		Name:      "sla_downtime_seconds_total",
		Help:      "Total downtime in seconds by service",
	}, []string{"service"})

	// ErrorBudgetTotal tracks the total error budget available for an SLO.
	ErrorBudgetTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nkudo",
		Name:      "error_budget_total",
		Help:      "Total error budget available for an SLO (in errors or seconds)",
	}, []string{"service", "slo"})

	// ErrorBudgetUsed tracks the amount of error budget used for an SLO.
	ErrorBudgetUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "nkudo",
		Name:      "error_budget_used",
		Help:      "Amount of error budget used for an SLO (in errors or seconds)",
	}, []string{"service", "slo"})

	// SLOStatusGauge tracks the current status of an SLO (1=healthy, 0=at risk, -1=breached).
	SLOStatusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nkudo",
		Name:      "slo_status",
		Help:      "Current SLO status: 1=healthy, 0=at risk, -1=breached",
	}, []string{"service", "slo", "slo_name"})

	// SLOCompliancePercentage tracks the current compliance percentage for an SLO.
	SLOCompliancePercentage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nkudo",
		Name:      "slo_compliance_percentage",
		Help:      "Current SLO compliance percentage",
	}, []string{"service", "slo"})

	// AvailabilityPercentage tracks the availability percentage by service.
	AvailabilityPercentage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nkudo",
		Name:      "availability_percentage",
		Help:      "Current availability percentage by service",
	}, []string{"service"})
)

func init() {
	prometheus.MustRegister(
		SLAUptime,
		SLADowntime,
		ErrorBudgetTotal,
		ErrorBudgetUsed,
		SLOStatusGauge,
		SLOCompliancePercentage,
		AvailabilityPercentage,
	)
}

// SLOConfig defines the configuration for an SLO.
type SLOConfig struct {
	Name           string
	Target         float64 // Target percentage (e.g., 99.9)
	Window         time.Duration
	ErrorBudgetPct float64 // Error budget as percentage (e.g., 0.1 for 0.1%)
}

// SLAMonitor tracks SLA metrics for a service.
type SLAMonitor struct {
	service     string
	slos        map[string]*SLOConfig
	mu          sync.RWMutex
	startTime   time.Time
	totalErrors int64
	totalReqs   int64
}

// NewSLAMonitor creates a new SLA monitor for a service.
func NewSLAMonitor(service string) *SLAMonitor {
	m := &SLAMonitor{
		service:   service,
		slos:      make(map[string]*SLOConfig),
		startTime: time.Now(),
	}
	return m
}

// RegisterSLO registers an SLO for this service.
func (m *SLAMonitor) RegisterSLO(config *SLOConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.slos[config.Name] = config

	// Initialize the error budget total based on window
	// For availability SLO: error budget = (100 - target)% of window
	budgetSeconds := config.Window.Seconds() * (100.0 - config.Target) / 100.0
	ErrorBudgetTotal.WithLabelValues(m.service, config.Name).Set(budgetSeconds)

	// Initialize status as healthy
	SLOStatusGauge.WithLabelValues(m.service, config.Name, config.Name).Set(float64(SLOStatusHealthy))
}

// RecordAvailability records an availability measurement.
// isSuccess indicates whether the request/operation succeeded.
func (m *SLAMonitor) RecordAvailability(isSuccess bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalReqs++
	if !isSuccess {
		m.totalErrors++
	}

	// Update availability percentage
	if m.totalReqs > 0 {
		avail := 100.0 * float64(m.totalReqs-m.totalErrors) / float64(m.totalReqs)
		AvailabilityPercentage.WithLabelValues(m.service).Set(avail)
	}
}

// RecordDowntime records a period of downtime.
func (m *SLAMonitor) RecordDowntime(duration time.Duration) {
	SLADowntime.WithLabelValues(m.service).Add(duration.Seconds())

	// Update error budgets for all SLOs
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name := range m.slos {
		ErrorBudgetUsed.WithLabelValues(m.service, name).Add(duration.Seconds())
		m.updateSLOStatus(name)
	}
}

// RecordUptime records a period of uptime.
func (m *SLAMonitor) RecordUptime(duration time.Duration) {
	SLAUptime.WithLabelValues(m.service).Add(duration.Seconds())
}

// RecordError increments the error budget used for the specified SLO.
func (m *SLAMonitor) RecordError(sloName string) {
	m.mu.RLock()
	slo, exists := m.slos[sloName]
	m.mu.RUnlock()

	if !exists {
		return
	}

	ErrorBudgetUsed.WithLabelValues(m.service, sloName).Inc()
	m.updateSLOStatus(sloName)

	// Update compliance percentage
	used := getCounterValueVec(ErrorBudgetUsed, m.service, sloName)
	total := getGaugeValue(ErrorBudgetTotal.WithLabelValues(m.service, sloName))
	if total > 0 {
		compliance := 100.0 * (1.0 - used/total)
		if compliance < 0 {
			compliance = 0
		}
		SLOCompliancePercentage.WithLabelValues(m.service, sloName).Set(compliance)
	}

	// Record for overall availability
	m.RecordAvailability(false)

	// Recalculate availability percentage based on SLO target
	if m.totalReqs > 0 {
		avail := 100.0 * float64(m.totalReqs-m.totalErrors) / float64(m.totalReqs)
		SLOCompliancePercentage.WithLabelValues(m.service, sloName).Set(
			min(avail/slo.Target*100.0, 100.0),
		)
	}
}

// updateSLOStatus updates the SLO status based on error budget usage.
func (m *SLAMonitor) updateSLOStatus(sloName string) {
	used := getCounterValueVec(ErrorBudgetUsed, m.service, sloName)
	total := getGaugeValue(ErrorBudgetTotal.WithLabelValues(m.service, sloName))

	if total <= 0 {
		return
	}

	burnRatio := used / total
	var status SLOStatus

	switch {
	case burnRatio >= 1.0:
		status = SLOStatusBreached
	case burnRatio >= 0.8:
		status = SLOStatusAtRisk
	default:
		status = SLOStatusHealthy
	}

	SLOStatusGauge.WithLabelValues(m.service, sloName, sloName).Set(float64(status))
}

// GetSLOStatus returns the current status of an SLO.
func (m *SLAMonitor) GetSLOStatus(sloName string) SLOStatus {
	value := getGaugeValue(SLOStatusGauge.WithLabelValues(m.service, sloName, sloName))
	return SLOStatus(value)
}

// GetErrorBudgetRemaining returns the remaining error budget for an SLO.
func (m *SLAMonitor) GetErrorBudgetRemaining(sloName string) float64 {
	total := getGaugeValue(ErrorBudgetTotal.WithLabelValues(m.service, sloName))
	used := getCounterValueVec(ErrorBudgetUsed, m.service, sloName)
	return total - used
}

// GetBurnRate returns the current error budget burn rate.
func (m *SLAMonitor) GetBurnRate(sloName string) float64 {
	total := getGaugeValue(ErrorBudgetTotal.WithLabelValues(m.service, sloName))
	if total <= 0 {
		return 0
	}
	used := getCounterValueVec(ErrorBudgetUsed, m.service, sloName)
	return used / total
}

// Reset resets all metrics for this monitor.
func (m *SLAMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalErrors = 0
	m.totalReqs = 0
	m.startTime = time.Now()

	// Reset counters
	SLAUptime.DeleteLabelValues(m.service)
	SLADowntime.DeleteLabelValues(m.service)
	AvailabilityPercentage.DeleteLabelValues(m.service)

	for name := range m.slos {
		ErrorBudgetUsed.DeleteLabelValues(m.service, name)
		SLOStatusGauge.DeleteLabelValues(m.service, name, name)
		SLOCompliancePercentage.DeleteLabelValues(m.service, name)
	}
}

// getCounterValueVec extracts the current value from a counter vector.
func getCounterValueVec(cv *prometheus.CounterVec, labels ...string) float64 {
	// Create a temporary counter to get value
	// Note: This is a workaround as prometheus.Counter doesn't expose value directly
	counter, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	return getCounterValue(counter)
}

// getCounterValue extracts the current value from a counter.
func getCounterValue(c prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := c.Write(metric); err != nil {
		return 0
	}
	if metric.Counter != nil && metric.Counter.Value != nil {
		return *metric.Counter.Value
	}
	return 0
}

// getGaugeValue extracts the current value from a gauge.
func getGaugeValue(g prometheus.Gauge) float64 {
	metric := &dto.Metric{}
	if err := g.Write(metric); err != nil {
		return 0
	}
	if metric.Gauge != nil && metric.Gauge.Value != nil {
		return *metric.Gauge.Value
	}
	return 0
}

// Global monitors map
type monitorRegistry struct {
	monitors map[string]*SLAMonitor
	mu       sync.RWMutex
}

var registry = &monitorRegistry{
	monitors: make(map[string]*SLAMonitor),
}

// GetOrCreateMonitor gets an existing monitor or creates a new one.
func GetOrCreateMonitor(service string) *SLAMonitor {
	registry.mu.RLock()
	m, exists := registry.monitors[service]
	registry.mu.RUnlock()

	if exists {
		return m
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Double-check after acquiring write lock
	if m, exists := registry.monitors[service]; exists {
		return m
	}

	m = NewSLAMonitor(service)
	registry.monitors[service] = m
	return m
}

// GetMonitor retrieves an existing monitor.
func GetMonitor(service string) (*SLAMonitor, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	m, exists := registry.monitors[service]
	return m, exists
}
