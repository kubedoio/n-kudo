package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// VMsTotal tracks the total number of VMs by state (running, stopped)
	VMsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_vms_total",
		Help: "Total number of VMs by state",
	}, []string{"state"})

	// VMCPUCores tracks CPU cores per VM
	VMCPUCores = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_vm_cpu_cores",
		Help: "Number of CPU cores per VM",
	}, []string{"vm_id", "vm_name"})

	// VMMemoryBytes tracks memory allocation per VM
	VMMemoryBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_vm_memory_bytes",
		Help: "Memory allocated to each VM in bytes",
	}, []string{"vm_id", "vm_name"})

	// ActionsExecuted tracks total actions executed by type and status
	ActionsExecuted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nkudo_actions_executed_total",
		Help: "Total actions executed",
	}, []string{"action_type", "status"})

	// ActionDuration tracks action execution duration in seconds
	ActionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nkudo_actions_duration_seconds",
		Help:    "Action execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"action_type"})

	// HeartbeatsSent tracks total heartbeats sent
	HeartbeatsSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nkudo_heartbeats_sent_total",
		Help: "Total heartbeats sent",
	})

	// HeartbeatDuration tracks heartbeat duration in seconds
	HeartbeatDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "nkudo_heartbeat_duration_seconds",
		Help:    "Heartbeat execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// HeartbeatFailures tracks total heartbeat failures
	HeartbeatFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nkudo_heartbeat_failures_total",
		Help: "Total heartbeat failures",
	})

	// DiskUsageBytes tracks disk usage by path
	DiskUsageBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_disk_usage_bytes",
		Help: "Disk usage in bytes by path",
	}, []string{"path"})

	// DiskTotalBytes tracks total disk space by path
	DiskTotalBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_disk_total_bytes",
		Help: "Total disk space in bytes by path",
	}, []string{"path"})

	// HostCPUUsagePercent tracks host CPU usage percentage
	HostCPUUsagePercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nkudo_host_cpu_usage_percent",
		Help: "Host CPU usage percentage",
	})

	// HostMemoryUsageBytes tracks host memory usage in bytes
	HostMemoryUsageBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nkudo_host_memory_usage_bytes",
		Help: "Host memory usage in bytes",
	})
)

func init() {
	prometheus.MustRegister(
		VMsTotal,
		VMCPUCores,
		VMMemoryBytes,
		ActionsExecuted,
		ActionDuration,
		HeartbeatsSent,
		HeartbeatDuration,
		HeartbeatFailures,
		DiskUsageBytes,
		DiskTotalBytes,
		HostCPUUsagePercent,
		HostMemoryUsageBytes,
	)
}

// StartServer starts the metrics HTTP server on the given address
func StartServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
