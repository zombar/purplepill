package metrics

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	_ "modernc.org/sqlite" // Import SQLite driver for tests
)

func TestNewHTTPMetrics(t *testing.T) {
	// Reset the default registry to avoid conflicts
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	metrics := NewHTTPMetrics("test-service")

	if metrics == nil {
		t.Fatal("NewHTTPMetrics returned nil")
	}

	if metrics.RequestDuration == nil {
		t.Error("RequestDuration histogram is nil")
	}

	if metrics.RequestsTotal == nil {
		t.Error("RequestsTotal counter is nil")
	}

	if metrics.RequestsActive == nil {
		t.Error("RequestsActive gauge is nil")
	}
}

func TestNewDatabaseMetrics(t *testing.T) {
	// Reset the default registry
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	metrics := NewDatabaseMetrics("test-service")

	if metrics == nil {
		t.Fatal("NewDatabaseMetrics returned nil")
	}

	if metrics.QueryDuration == nil {
		t.Error("QueryDuration histogram is nil")
	}

	if metrics.QueriesTotal == nil {
		t.Error("QueriesTotal counter is nil")
	}

	if metrics.ConnectionsOpen == nil {
		t.Error("ConnectionsOpen gauge is nil")
	}

	if metrics.ConnectionsIdle == nil {
		t.Error("ConnectionsIdle gauge is nil")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	// Reset registry and create metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewHTTPMetrics("test-service")

	// Record a successful request
	metrics.RecordHTTPRequest("GET", "/api/test", 200, 100*time.Millisecond)

	// Verify the counter increased
	expectedMetric := `
		# HELP http_requests_total Total number of HTTP requests
		# TYPE http_requests_total counter
		http_requests_total{method="GET",path="/api/test",service="test-service",status="200"} 1
	`

	if err := testutil.CollectAndCompare(metrics.RequestsTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

func TestRecordDBQuery(t *testing.T) {
	// Reset registry and create metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewDatabaseMetrics("test-service")

	// Record a successful query
	metrics.RecordDBQuery("SELECT", "users", "success", 50*time.Millisecond)

	// Verify the counter increased
	expectedMetric := `
		# HELP db_queries_total Total number of database queries
		# TYPE db_queries_total counter
		db_queries_total{operation="SELECT",service="test-service",status="success",table="users"} 1
	`

	if err := testutil.CollectAndCompare(metrics.QueriesTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	// Reset registry and create metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewHTTPMetrics("test-service")

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrappedHandler := metrics.HTTPMiddleware(handler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the requests_total counter was incremented
	expectedMetric := `
		# HELP http_requests_total Total number of HTTP requests
		# TYPE http_requests_total counter
		http_requests_total{method="GET",path="/test",service="test-service",status="200"} 1
	`

	if err := testutil.CollectAndCompare(metrics.RequestsTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

func TestHTTPMiddlewareMultipleRequests(t *testing.T) {
	// Reset registry and create metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewHTTPMetrics("test-service")

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := metrics.HTTPMiddleware(handler)

	// Execute multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verify counter shows 5 requests
	expectedMetric := `
		# HELP http_requests_total Total number of HTTP requests
		# TYPE http_requests_total counter
		http_requests_total{method="POST",path="/api/test",service="test-service",status="200"} 5
	`

	if err := testutil.CollectAndCompare(metrics.RequestsTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

func TestHTTPMiddlewareDifferentStatusCodes(t *testing.T) {
	// Reset registry and create metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewHTTPMetrics("test-service")

	testCases := []struct {
		name       string
		statusCode int
	}{
		{"success", http.StatusOK},
		{"created", http.StatusCreated},
		{"bad_request", http.StatusBadRequest},
		{"not_found", http.StatusNotFound},
		{"server_error", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			wrappedHandler := metrics.HTTPMiddleware(handler)
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
			}
		})
	}
}

func TestUpdateDBStats(t *testing.T) {
	// This test requires a real database connection, so we'll use an in-memory SQLite
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Set connection pool limits
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Reset registry and create metrics
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	metrics := NewDatabaseMetrics("test-service")

	// Update stats
	metrics.UpdateDBStats(db)

	// Verify metrics were set by gathering from the same registry
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	foundOpen := false
	foundIdle := false

	for _, family := range families {
		if family.GetName() == "db_connections_open" {
			foundOpen = true
			// Verify the value is set correctly
			if len(family.GetMetric()) > 0 {
				value := family.GetMetric()[0].GetGauge().GetValue()
				if value < 0 {
					t.Errorf("Expected non-negative connections, got %f", value)
				}
			}
		}
		if family.GetName() == "db_connections_idle" {
			foundIdle = true
			if len(family.GetMetric()) > 0 {
				value := family.GetMetric()[0].GetGauge().GetValue()
				if value < 0 {
					t.Errorf("Expected non-negative idle connections, got %f", value)
				}
			}
		}
	}

	if !foundOpen {
		t.Error("db_connections_open metric not found")
	}
	if !foundIdle {
		t.Error("db_connections_idle metric not found")
	}
}

func TestHandler(t *testing.T) {
	// Reset registry and make sure DefaultGatherer uses it
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg

	// Create some metrics to ensure handler has something to expose
	metrics := NewHTTPMetrics("test-service")
	metrics.RecordHTTPRequest("GET", "/test", 200, 100*time.Millisecond)

	// Get the handler - note that promhttp.Handler() uses DefaultGatherer
	// We need to create a handler that uses our custom registry
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected content-type to contain text/plain, got %s", contentType)
	}

	// Verify response contains metrics
	body := w.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Error("Response does not contain http_requests_total metric")
	}
}
