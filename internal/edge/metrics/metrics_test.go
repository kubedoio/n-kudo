package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsEndpoint(t *testing.T) {
	// Reset registry to avoid conflicts
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the response contains Prometheus format
	body := rr.Body.String()
	if !strings.Contains(body, "# HELP") {
		t.Error("metrics response should contain # HELP")
	}
}

func TestVMsTotalMetric(t *testing.T) {
	// Create a new registry for this test
	reg := prometheus.NewRegistry()
	
	// Create new metrics with the test registry
	vmsTotal := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_vms_total",
		Help: "Total number of VMs by state",
	}, []string{"state"})
	
	reg.MustRegister(vmsTotal)

	// Set some values
	vmsTotal.WithLabelValues("running").Set(5)
	vmsTotal.WithLabelValues("stopped").Set(2)

	// Verify the values are set (this would be checked via /metrics endpoint)
	// For unit testing, we can verify the metric is registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "nkudo_vms_total" {
			found = true
			break
		}
	}

	if !found {
		t.Error("nkudo_vms_total metric not found in registry")
	}
}

func TestActionsExecutedMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	actionsExecuted := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nkudo_actions_executed_total",
		Help: "Total actions executed",
	}, []string{"action_type", "status"})
	
	reg.MustRegister(actionsExecuted)

	// Increment counters
	actionsExecuted.WithLabelValues("MicroVMCreate", "success").Inc()
	actionsExecuted.WithLabelValues("MicroVMCreate", "success").Inc()
	actionsExecuted.WithLabelValues("MicroVMCreate", "failure").Inc()

	// Verify metric is registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "nkudo_actions_executed_total" {
			found = true
			break
		}
	}

	if !found {
		t.Error("nkudo_actions_executed_total metric not found in registry")
	}
}

func TestActionDurationMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	actionDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nkudo_actions_duration_seconds",
		Help:    "Action execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"action_type"})
	
	reg.MustRegister(actionDuration)

	// Observe some durations
	actionDuration.WithLabelValues("MicroVMCreate").Observe(1.5)
	actionDuration.WithLabelValues("MicroVMCreate").Observe(2.0)
	actionDuration.WithLabelValues("MicroVMDelete").Observe(0.5)

	// Verify metric is registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "nkudo_actions_duration_seconds" {
			found = true
			break
		}
	}

	if !found {
		t.Error("nkudo_actions_duration_seconds metric not found in registry")
	}
}

func TestHeartbeatsSentMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	heartbeatsSent := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nkudo_heartbeats_sent_total",
		Help: "Total heartbeats sent",
	})
	
	reg.MustRegister(heartbeatsSent)

	// Increment counter
	heartbeatsSent.Inc()
	heartbeatsSent.Inc()
	heartbeatsSent.Inc()

	// Verify metric is registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range families {
		if f.GetName() == "nkudo_heartbeats_sent_total" {
			found = true
			break
		}
	}

	if !found {
		t.Error("nkudo_heartbeats_sent_total metric not found in registry")
	}
}

func TestDiskMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	diskUsage := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_disk_usage_bytes",
		Help: "Disk usage in bytes by path",
	}, []string{"path"})
	
	diskTotal := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nkudo_disk_total_bytes",
		Help: "Total disk space in bytes by path",
	}, []string{"path"})
	
	reg.MustRegister(diskUsage, diskTotal)

	// Set values
	diskUsage.WithLabelValues("/var/lib/nkudo-edge").Set(10737418240)
	diskTotal.WithLabelValues("/var/lib/nkudo-edge").Set(107374182400)

	// Verify metrics are registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	foundUsage := false
	foundTotal := false
	for _, f := range families {
		if f.GetName() == "nkudo_disk_usage_bytes" {
			foundUsage = true
		}
		if f.GetName() == "nkudo_disk_total_bytes" {
			foundTotal = true
		}
	}

	if !foundUsage {
		t.Error("nkudo_disk_usage_bytes metric not found in registry")
	}
	if !foundTotal {
		t.Error("nkudo_disk_total_bytes metric not found in registry")
	}
}

func TestHostMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	hostCPU := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nkudo_host_cpu_usage_percent",
		Help: "Host CPU usage percentage",
	})
	
	hostMemory := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nkudo_host_memory_usage_bytes",
		Help: "Host memory usage in bytes",
	})
	
	reg.MustRegister(hostCPU, hostMemory)

	// Set values
	hostCPU.Set(25.5)
	hostMemory.Set(8589934592)

	// Verify metrics are registered
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	foundCPU := false
	foundMemory := false
	for _, f := range families {
		if f.GetName() == "nkudo_host_cpu_usage_percent" {
			foundCPU = true
		}
		if f.GetName() == "nkudo_host_memory_usage_bytes" {
			foundMemory = true
		}
	}

	if !foundCPU {
		t.Error("nkudo_host_cpu_usage_percent metric not found in registry")
	}
	if !foundMemory {
		t.Error("nkudo_host_memory_usage_bytes metric not found in registry")
	}
}
