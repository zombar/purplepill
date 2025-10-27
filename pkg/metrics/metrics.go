package metrics

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
)

// Handler returns an HTTP handler for Prometheus metrics endpoint
func Handler() http.Handler {
	return promhttp.Handler()
}

// RegisterDefaultMetrics registers default Go runtime metrics
func RegisterDefaultMetrics() {
	// Default registry already includes Go runtime metrics
	// This function is provided for consistency and future customization
}

// BusinessMetrics contains business-specific metrics for DocuTab services
type BusinessMetrics struct {
	// Controller metrics
	ScrapeRequestsTotal *prometheus.CounterVec
	ScrapeJobsTotal     *prometheus.CounterVec
	ScrapeJobsByStatus  *prometheus.GaugeVec
	QueueLength         prometheus.Gauge

	// Scraper metrics
	ScrapesCompletedTotal *prometheus.CounterVec
	LinksExtractedTotal   prometheus.Counter
	ImagesProcessedTotal  prometheus.Counter
	OllamaRequestsTotal   *prometheus.CounterVec
	ScrapeDuration        *prometheus.HistogramVec

	// TextAnalyzer metrics
	AnalysesTotal         *prometheus.CounterVec
	TagsGeneratedTotal    prometheus.Counter
	SynopsisGeneratedTotal prometheus.Counter
	AnalyzerOllamaRequests *prometheus.CounterVec
	AnalysisDuration      *prometheus.HistogramVec

	// Scheduler metrics
	TasksScheduledTotal *prometheus.CounterVec
	TasksExecutedTotal  *prometheus.CounterVec
	TaskFailuresTotal   *prometheus.CounterVec
	ActiveTasks         prometheus.Gauge
}

// NewBusinessMetrics creates and registers business metrics for a specific service
func NewBusinessMetrics(serviceName string) *BusinessMetrics {
	m := &BusinessMetrics{}

	switch serviceName {
	case "controller":
		m.ScrapeRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_scrape_requests_total",
				Help: "Total number of scrape requests received",
			},
			[]string{"status"},
		)
		m.ScrapeJobsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_scrape_jobs_total",
				Help: "Total number of scrape jobs created",
			},
			[]string{"type"},
		)
		m.ScrapeJobsByStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docutab_scrape_jobs_by_status",
				Help: "Number of scrape jobs by status",
			},
			[]string{"status"},
		)
		m.QueueLength = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "docutab_queue_length",
				Help: "Current number of jobs in the queue",
			},
		)
		prometheus.MustRegister(m.ScrapeRequestsTotal)
		prometheus.MustRegister(m.ScrapeJobsTotal)
		prometheus.MustRegister(m.ScrapeJobsByStatus)
		prometheus.MustRegister(m.QueueLength)

	case "scraper":
		m.ScrapesCompletedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_scrapes_completed_total",
				Help: "Total number of scrapes completed",
			},
			[]string{"status"},
		)
		m.LinksExtractedTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "docutab_links_extracted_total",
				Help: "Total number of links extracted from scraped pages",
			},
		)
		m.ImagesProcessedTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "docutab_images_processed_total",
				Help: "Total number of images processed",
			},
		)
		m.OllamaRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_ollama_requests_total",
				Help: "Total number of Ollama API requests",
			},
			[]string{"type", "status"},
		)
		m.ScrapeDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "docutab_scrape_duration_seconds",
				Help:    "Duration of scrape operations in seconds",
				Buckets: []float64{0.5, 1, 2.5, 5, 10, 30, 60, 120},
			},
			[]string{"status"},
		)
		prometheus.MustRegister(m.ScrapesCompletedTotal)
		prometheus.MustRegister(m.LinksExtractedTotal)
		prometheus.MustRegister(m.ImagesProcessedTotal)
		prometheus.MustRegister(m.OllamaRequestsTotal)
		prometheus.MustRegister(m.ScrapeDuration)

	case "textanalyzer":
		m.AnalysesTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_analyses_total",
				Help: "Total number of text analyses performed",
			},
			[]string{"status"},
		)
		m.TagsGeneratedTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "docutab_tags_generated_total",
				Help: "Total number of tags generated",
			},
		)
		m.SynopsisGeneratedTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "docutab_synopsis_generated_total",
				Help: "Total number of synopses generated",
			},
		)
		m.AnalyzerOllamaRequests = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_analyzer_ollama_requests_total",
				Help: "Total number of Ollama requests from text analyzer",
			},
			[]string{"status"},
		)
		m.AnalysisDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "docutab_analysis_duration_seconds",
				Help:    "Duration of text analysis operations in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"status"},
		)
		prometheus.MustRegister(m.AnalysesTotal)
		prometheus.MustRegister(m.TagsGeneratedTotal)
		prometheus.MustRegister(m.SynopsisGeneratedTotal)
		prometheus.MustRegister(m.AnalyzerOllamaRequests)
		prometheus.MustRegister(m.AnalysisDuration)

	case "scheduler":
		m.TasksScheduledTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_tasks_scheduled_total",
				Help: "Total number of tasks scheduled",
			},
			[]string{"type"},
		)
		m.TasksExecutedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_tasks_executed_total",
				Help: "Total number of tasks executed",
			},
			[]string{"type", "status"},
		)
		m.TaskFailuresTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "docutab_task_failures_total",
				Help: "Total number of task failures",
			},
			[]string{"type", "reason"},
		)
		m.ActiveTasks = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "docutab_active_tasks",
				Help: "Number of currently active tasks",
			},
		)
		prometheus.MustRegister(m.TasksScheduledTotal)
		prometheus.MustRegister(m.TasksExecutedTotal)
		prometheus.MustRegister(m.TaskFailuresTotal)
		prometheus.MustRegister(m.ActiveTasks)
	}

	return m
}

// ObserveDurationWithExemplar records a duration observation with an exemplar linking to the current trace.
// This enables jumping from Grafana metrics to Tempo traces for correlated analysis.
// If no trace context is available, it falls back to a regular observation.
func (m *BusinessMetrics) ObserveDurationWithExemplar(ctx context.Context, histogram *prometheus.HistogramVec, duration float64, labels ...string) {
	if histogram == nil {
		return
	}

	// Extract trace context from the request
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		// No trace context, record without exemplar
		histogram.WithLabelValues(labels...).Observe(duration)
		return
	}

	// Get trace ID for the exemplar
	traceID := span.SpanContext().TraceID().String()

	// Create exemplar with trace ID
	exemplar := prometheus.Labels{
		"trace_id": traceID,
	}

	// Record observation with exemplar
	observer := histogram.WithLabelValues(labels...)
	if exemplarObserver, ok := observer.(prometheus.ExemplarObserver); ok {
		exemplarObserver.ObserveWithExemplar(duration, exemplar)
	} else {
		// Fallback if ExemplarObserver interface not available
		observer.Observe(duration)
	}
}

// DatabaseMetrics contains database-related metrics
type DatabaseMetrics struct {
	ConnectionsOpen    prometheus.Gauge
	ConnectionsIdle    prometheus.Gauge
	ConnectionsInUse   prometheus.Gauge
	WaitCount          prometheus.Counter
	WaitDuration       prometheus.Counter
	QueryDuration      *prometheus.HistogramVec
}

// NewDatabaseMetrics creates and registers database metrics for a specific service
func NewDatabaseMetrics(serviceName string) *DatabaseMetrics {
	m := &DatabaseMetrics{
		ConnectionsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_open",
				Help: "Number of open database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		),
		ConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		),
		ConnectionsInUse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_in_use",
				Help: "Number of database connections currently in use",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		),
		WaitCount: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_connections_wait_count_total",
				Help: "Total number of times a connection was waited for",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		),
		WaitDuration: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_connections_wait_duration_seconds_total",
				Help: "Total time waited for database connections in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		),
		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
			[]string{"operation"},
		),
	}

	prometheus.MustRegister(m.ConnectionsOpen)
	prometheus.MustRegister(m.ConnectionsIdle)
	prometheus.MustRegister(m.ConnectionsInUse)
	prometheus.MustRegister(m.WaitCount)
	prometheus.MustRegister(m.WaitDuration)
	prometheus.MustRegister(m.QueryDuration)

	return m
}

// UpdateDBStats updates database connection pool metrics from sql.DBStats
func (m *DatabaseMetrics) UpdateDBStats(db *sql.DB) {
	stats := db.Stats()
	m.ConnectionsOpen.Set(float64(stats.OpenConnections))
	m.ConnectionsIdle.Set(float64(stats.Idle))
	m.ConnectionsInUse.Set(float64(stats.InUse))
	m.WaitCount.Add(float64(stats.WaitCount))
	m.WaitDuration.Add(stats.WaitDuration.Seconds())
}

var (
	httpMetricsOnce          sync.Once
	httpRequestsTotal        *prometheus.CounterVec
	httpRequestDuration      *prometheus.HistogramVec
	httpRequestSize          *prometheus.HistogramVec
	httpResponseSize         *prometheus.HistogramVec
	httpRequestsActiveByService = make(map[string]prometheus.Gauge)
	httpRequestsActiveMutex     sync.Mutex
)

// HTTPMiddleware wraps an HTTP handler with Prometheus metrics
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	// Register shared metrics only once
	httpMetricsOnce.Do(func() {
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"service", "method", "path", "status"},
		)

		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method", "path", "status"},
		)

		httpRequestSize = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"service", "method", "path"},
		)

		httpResponseSize = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"service", "method", "path"},
		)

		prometheus.MustRegister(httpRequestsTotal)
		prometheus.MustRegister(httpRequestDuration)
		prometheus.MustRegister(httpRequestSize)
		prometheus.MustRegister(httpResponseSize)
	})

	// Create per-service active requests gauge
	httpRequestsActiveMutex.Lock()
	httpRequestsActive, exists := httpRequestsActiveByService[serviceName]
	if !exists {
		httpRequestsActive = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_active",
				Help: "Number of HTTP requests currently being served",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
					"app":     "docutab",
				},
			},
		)
		prometheus.MustRegister(httpRequestsActive)
		httpRequestsActiveByService[serviceName] = httpRequestsActive
	}
	httpRequestsActiveMutex.Unlock()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment active requests
			httpRequestsActive.Inc()
			defer httpRequestsActive.Dec()

			// Record request size
			if r.ContentLength > 0 {
				httpRequestSize.WithLabelValues(serviceName, r.Method, r.URL.Path).Observe(float64(r.ContentLength))
			}

			// Create response writer wrapper to capture status code and size
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				size:           0,
			}

			// Start timer
			start := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				status := strconv.Itoa(wrapped.statusCode)
				httpRequestDuration.WithLabelValues(serviceName, r.Method, r.URL.Path, status).Observe(v)
			}))
			defer start.ObserveDuration()

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Record metrics
			status := strconv.Itoa(wrapped.statusCode)
			httpRequestsTotal.WithLabelValues(serviceName, r.Method, r.URL.Path, status).Inc()
			httpResponseSize.WithLabelValues(serviceName, r.Method, r.URL.Path).Observe(float64(wrapped.size))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}
