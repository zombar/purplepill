package metrics

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics tracks HTTP request metrics
type HTTPMetrics struct {
	RequestDuration *prometheus.HistogramVec
	RequestsTotal   *prometheus.CounterVec
	RequestsActive  prometheus.Gauge
}

// DatabaseMetrics tracks database operation metrics
type DatabaseMetrics struct {
	QueryDuration   *prometheus.HistogramVec
	QueriesTotal    *prometheus.CounterVec
	ConnectionsOpen prometheus.Gauge
	ConnectionsIdle prometheus.Gauge
}

// NewHTTPMetrics creates HTTP metrics for a service
func NewHTTPMetrics(serviceName string) *HTTPMetrics {
	return &HTTPMetrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latencies in seconds",
				Buckets: prometheus.DefBuckets,
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "path", "status"},
		),
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "path", "status"},
		),
		RequestsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_active",
				Help: "Number of active HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
	}
}

// NewDatabaseMetrics creates database metrics for a service
func NewDatabaseMetrics(serviceName string) *DatabaseMetrics {
	return &DatabaseMetrics{
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query latencies in seconds",
				Buckets: prometheus.DefBuckets,
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"operation", "table"},
		),
		QueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"operation", "table", "status"},
		),
		ConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_open",
				Help: "Number of open database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
		ConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *HTTPMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	m.RequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	m.RequestsTotal.WithLabelValues(method, path, status).Inc()
}

// RecordDBQuery records database query metrics
func (m *DatabaseMetrics) RecordDBQuery(operation, table, status string, duration time.Duration) {
	m.QueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	m.QueriesTotal.WithLabelValues(operation, table, status).Inc()
}

// UpdateDBStats updates database connection pool stats
func (m *DatabaseMetrics) UpdateDBStats(db *sql.DB) {
	stats := db.Stats()
	m.ConnectionsOpen.Set(float64(stats.OpenConnections))
	m.ConnectionsIdle.Set(float64(stats.Idle))
}

// HTTPMiddleware is a middleware that records HTTP metrics
func (m *HTTPMetrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		m.RequestsActive.Inc()
		defer m.RequestsActive.Dec()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		m.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
