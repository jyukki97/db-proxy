package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	// Use a custom registry to avoid conflicts with default global registry
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	defer func() {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		prometheus.DefaultGatherer = prometheus.NewRegistry()
	}()

	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}

	// Verify counter increments work
	m.QueriesRouted.WithLabelValues("writer").Inc()
	m.QueriesRouted.WithLabelValues("reader").Inc()
	m.CacheHits.Inc()
	m.CacheMisses.Inc()
	m.CacheInvalidations.Inc()
	m.ReaderFallback.Inc()
	m.CacheEntries.Set(42)
	m.PoolOpenConns.WithLabelValues("reader", "localhost:5433").Set(5)
	m.PoolIdleConns.WithLabelValues("reader", "localhost:5433").Set(2)
	m.PoolAcquires.WithLabelValues("reader", "localhost:5433").Inc()
	m.PoolAcquireDur.WithLabelValues("reader", "localhost:5433").Observe(0.001)
	m.QueryDuration.WithLabelValues("writer").Observe(0.05)

	// Verify metrics are gathered
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	if len(families) == 0 {
		t.Fatal("no metric families gathered")
	}

	// Check expected metric names
	names := make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}

	expected := []string{
		"pgmux_queries_routed_total",
		"pgmux_query_duration_seconds",
		"pgmux_cache_hits_total",
		"pgmux_cache_misses_total",
		"pgmux_cache_entries",
		"pgmux_cache_invalidations_total",
		"pgmux_pool_connections_open",
		"pgmux_pool_connections_idle",
		"pgmux_pool_acquires_total",
		"pgmux_pool_acquire_duration_seconds",
		"pgmux_reader_fallback_total",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected metric %q not found", name)
		}
	}
}
