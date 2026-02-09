package sla

import (
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DBQueryDuration is a histogram of DB query durations labeled by query/table name.
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nkudo_db_query_duration_seconds",
			Help:    "Histogram of DB query durations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query", "table"},
	)

	// DBConnectionsActive is a gauge of active DB connections.
	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nkudo_db_connections_active",
			Help: "Number of active DB connections",
		},
	)

	// DBConnectionsIdle is a gauge of idle DB connections.
	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nkudo_db_connections_idle",
			Help: "Number of idle DB connections",
		},
	)

	// DBConnectionsMax is a gauge of maximum allowed DB connections.
	DBConnectionsMax = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nkudo_db_connections_max",
			Help: "Maximum number of DB connections allowed",
		},
	)

	// DBWaitCount is a counter of total connection waits.
	DBWaitCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nkudo_db_wait_count_total",
			Help: "Total number of connection waits",
		},
	)

	// CacheHitRate is a gauge of cache hit rate (0-1).
	CacheHitRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nkudo_cache_hit_rate",
			Help: "Cache hit rate (0-1)",
		},
	)

	// CacheHits is a counter of cache hits.
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nkudo_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	// CacheMisses is a counter of cache misses.
	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nkudo_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)
)

// RecordCacheHit records a cache hit and updates the hit rate.
func RecordCacheHit() {
	CacheHits.Inc()
	updateCacheHitRate()
}

// RecordCacheMiss records a cache miss and updates the hit rate.
func RecordCacheMiss() {
	CacheMisses.Inc()
	updateCacheHitRate()
}

// updateCacheHitRate recalculates and sets the cache hit rate gauge.
func updateCacheHitRate() {
	hits := getCounterValueInternal(CacheHits)
	misses := getCounterValueInternal(CacheMisses)
	total := hits + misses
	if total > 0 {
		CacheHitRate.Set(hits / total)
	}
}

// getCounterValueInternal extracts the current value from a counter.
// This is a helper for testing/metrics calculation purposes.
func getCounterValueInternal(c prometheus.Counter) float64 {
	metric := &dto.Metric{}
	c.Write(metric)
	if metric.Counter != nil && metric.Counter.Value != nil {
		return *metric.Counter.Value
	}
	return 0
}
