package metrics

import (
	"database/sql"
	// "net/http"
	// "net/http/httptest"
	"strings"
	"testing"
	// "time"

	"github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	_ "modernc.org/sqlite" // Import SQLite driver for tests
)

// TODO: Fix these tests - they reference old interfaces that don't exist
/*
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
*/

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

/*
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
*/

// TestNewBusinessMetrics_Controller tests controller business metrics creation
func TestNewBusinessMetrics_Controller(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("controller")

	if metrics == nil {
		t.Fatal("NewBusinessMetrics returned nil")
	}

	if metrics.ScrapeRequestsTotal == nil {
		t.Error("ScrapeRequestsTotal counter is nil")
	}
	if metrics.ScrapeJobsTotal == nil {
		t.Error("ScrapeJobsTotal counter is nil")
	}
	if metrics.ScrapeJobsByStatus == nil {
		t.Error("ScrapeJobsByStatus gauge is nil")
	}
	if metrics.QueueLength == nil {
		t.Error("QueueLength gauge is nil")
	}
}

// TestNewBusinessMetrics_Scraper tests scraper business metrics creation
func TestNewBusinessMetrics_Scraper(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scraper")

	if metrics == nil {
		t.Fatal("NewBusinessMetrics returned nil")
	}

	if metrics.ScrapesCompletedTotal == nil {
		t.Error("ScrapesCompletedTotal counter is nil")
	}
	if metrics.LinksExtractedTotal == nil {
		t.Error("LinksExtractedTotal counter is nil")
	}
	if metrics.ImagesProcessedTotal == nil {
		t.Error("ImagesProcessedTotal counter is nil")
	}
	if metrics.OllamaRequestsTotal == nil {
		t.Error("OllamaRequestsTotal counter is nil")
	}
	if metrics.ScrapeDuration == nil {
		t.Error("ScrapeDuration histogram is nil")
	}
}

// TestNewBusinessMetrics_TextAnalyzer tests textanalyzer business metrics creation
func TestNewBusinessMetrics_TextAnalyzer(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("textanalyzer")

	if metrics == nil {
		t.Fatal("NewBusinessMetrics returned nil")
	}

	if metrics.AnalysesTotal == nil {
		t.Error("AnalysesTotal counter is nil")
	}
	if metrics.TagsGeneratedTotal == nil {
		t.Error("TagsGeneratedTotal counter is nil")
	}
	if metrics.SynopsisGeneratedTotal == nil {
		t.Error("SynopsisGeneratedTotal counter is nil")
	}
	if metrics.AnalyzerOllamaRequests == nil {
		t.Error("AnalyzerOllamaRequests counter is nil")
	}
	if metrics.AnalysisDuration == nil {
		t.Error("AnalysisDuration histogram is nil")
	}
}

// TestNewBusinessMetrics_Scheduler tests scheduler business metrics creation
func TestNewBusinessMetrics_Scheduler(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scheduler")

	if metrics == nil {
		t.Fatal("NewBusinessMetrics returned nil")
	}

	if metrics.TasksScheduledTotal == nil {
		t.Error("TasksScheduledTotal counter is nil")
	}
	if metrics.TasksExecutedTotal == nil {
		t.Error("TasksExecutedTotal counter is nil")
	}
	if metrics.TaskFailuresTotal == nil {
		t.Error("TaskFailuresTotal counter is nil")
	}
	if metrics.ActiveTasks == nil {
		t.Error("ActiveTasks gauge is nil")
	}
}

// TestControllerMetrics_ScrapeRequests tests controller scrape request metrics
func TestControllerMetrics_ScrapeRequests(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("controller")

	// Record accepted scrape requests
	metrics.ScrapeRequestsTotal.WithLabelValues("accepted").Inc()
	metrics.ScrapeRequestsTotal.WithLabelValues("accepted").Inc()
	metrics.ScrapeRequestsTotal.WithLabelValues("error").Inc()

	// Verify the counter values
	expectedMetric := `
		# HELP docutab_scrape_requests_total Total number of scrape requests received
		# TYPE docutab_scrape_requests_total counter
		docutab_scrape_requests_total{status="accepted"} 2
		docutab_scrape_requests_total{status="error"} 1
	`

	if err := testutil.CollectAndCompare(metrics.ScrapeRequestsTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

// TestControllerMetrics_ScrapeJobs tests controller scrape job metrics
func TestControllerMetrics_ScrapeJobs(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("controller")

	// Record scrape jobs created
	metrics.ScrapeJobsTotal.WithLabelValues("parent").Inc()
	metrics.ScrapeJobsTotal.WithLabelValues("child").Inc()
	metrics.ScrapeJobsTotal.WithLabelValues("child").Inc()
	metrics.ScrapeJobsTotal.WithLabelValues("child").Inc()

	expectedMetric := `
		# HELP docutab_scrape_jobs_total Total number of scrape jobs created
		# TYPE docutab_scrape_jobs_total counter
		docutab_scrape_jobs_total{type="parent"} 1
		docutab_scrape_jobs_total{type="child"} 3
	`

	if err := testutil.CollectAndCompare(metrics.ScrapeJobsTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

// TestControllerMetrics_JobsByStatus tests controller job status gauge
func TestControllerMetrics_JobsByStatus(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("controller")

	// Set job status counts
	metrics.ScrapeJobsByStatus.WithLabelValues("pending").Set(5)
	metrics.ScrapeJobsByStatus.WithLabelValues("processing").Set(2)
	metrics.ScrapeJobsByStatus.WithLabelValues("completed").Set(10)
	metrics.ScrapeJobsByStatus.WithLabelValues("failed").Set(1)

	expectedMetric := `
		# HELP docutab_scrape_jobs_by_status Number of scrape jobs by status
		# TYPE docutab_scrape_jobs_by_status gauge
		docutab_scrape_jobs_by_status{status="pending"} 5
		docutab_scrape_jobs_by_status{status="processing"} 2
		docutab_scrape_jobs_by_status{status="completed"} 10
		docutab_scrape_jobs_by_status{status="failed"} 1
	`

	if err := testutil.CollectAndCompare(metrics.ScrapeJobsByStatus, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

// TestScraperMetrics_ScrapesCompleted tests scraper completion metrics
func TestScraperMetrics_ScrapesCompleted(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scraper")

	// Record scrape completions
	metrics.ScrapesCompletedTotal.WithLabelValues("success").Inc()
	metrics.ScrapesCompletedTotal.WithLabelValues("success").Inc()
	metrics.ScrapesCompletedTotal.WithLabelValues("success").Inc()
	metrics.ScrapesCompletedTotal.WithLabelValues("error").Inc()

	expectedMetric := `
		# HELP docutab_scrapes_completed_total Total number of scrapes completed
		# TYPE docutab_scrapes_completed_total counter
		docutab_scrapes_completed_total{status="success"} 3
		docutab_scrapes_completed_total{status="error"} 1
	`

	if err := testutil.CollectAndCompare(metrics.ScrapesCompletedTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

// TestScraperMetrics_LinksAndImages tests link and image extraction metrics
func TestScraperMetrics_LinksAndImages(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scraper")

	// Record links and images extracted
	metrics.LinksExtractedTotal.Add(15)
	metrics.ImagesProcessedTotal.Add(7)

	expectedLinksMetric := `
		# HELP docutab_links_extracted_total Total number of links extracted from scraped pages
		# TYPE docutab_links_extracted_total counter
		docutab_links_extracted_total 15
	`

	if err := testutil.CollectAndCompare(metrics.LinksExtractedTotal, strings.NewReader(expectedLinksMetric)); err != nil {
		t.Errorf("unexpected links metric value: %v", err)
	}

	expectedImagesMetric := `
		# HELP docutab_images_processed_total Total number of images processed
		# TYPE docutab_images_processed_total counter
		docutab_images_processed_total 7
	`

	if err := testutil.CollectAndCompare(metrics.ImagesProcessedTotal, strings.NewReader(expectedImagesMetric)); err != nil {
		t.Errorf("unexpected images metric value: %v", err)
	}
}

// TestScraperMetrics_ScrapeDuration tests scrape duration histogram
func TestScraperMetrics_ScrapeDuration(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scraper")

	// Record scrape durations
	metrics.ScrapeDuration.WithLabelValues("success").Observe(1.5)
	metrics.ScrapeDuration.WithLabelValues("success").Observe(2.3)
	metrics.ScrapeDuration.WithLabelValues("error").Observe(0.5)

	// Verify histogram has observations
	reg := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, family := range families {
		if family.GetName() == "docutab_scrape_duration_seconds" {
			found = true
			// Verify we have metrics with the right labels
			if len(family.GetMetric()) == 0 {
				t.Error("Expected histogram to have metrics")
			}
		}
	}

	if !found {
		t.Error("docutab_scrape_duration_seconds metric not found")
	}
}

// TestTextAnalyzerMetrics_AnalysesTotal tests analysis completion metrics
func TestTextAnalyzerMetrics_AnalysesTotal(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("textanalyzer")

	// Record analysis completions
	metrics.AnalysesTotal.WithLabelValues("success").Inc()
	metrics.AnalysesTotal.WithLabelValues("success").Inc()
	metrics.AnalysesTotal.WithLabelValues("error").Inc()

	expectedMetric := `
		# HELP docutab_analyses_total Total number of text analyses performed
		# TYPE docutab_analyses_total counter
		docutab_analyses_total{status="success"} 2
		docutab_analyses_total{status="error"} 1
	`

	if err := testutil.CollectAndCompare(metrics.AnalysesTotal, strings.NewReader(expectedMetric)); err != nil {
		t.Errorf("unexpected metric value: %v", err)
	}
}

// TestTextAnalyzerMetrics_TagsAndSynopsis tests tag and synopsis generation metrics
func TestTextAnalyzerMetrics_TagsAndSynopsis(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("textanalyzer")

	// Record tags and synopses generated
	metrics.TagsGeneratedTotal.Add(25)
	metrics.SynopsisGeneratedTotal.Inc()
	metrics.SynopsisGeneratedTotal.Inc()

	expectedTagsMetric := `
		# HELP docutab_tags_generated_total Total number of tags generated
		# TYPE docutab_tags_generated_total counter
		docutab_tags_generated_total 25
	`

	if err := testutil.CollectAndCompare(metrics.TagsGeneratedTotal, strings.NewReader(expectedTagsMetric)); err != nil {
		t.Errorf("unexpected tags metric value: %v", err)
	}

	expectedSynopsisMetric := `
		# HELP docutab_synopsis_generated_total Total number of synopses generated
		# TYPE docutab_synopsis_generated_total counter
		docutab_synopsis_generated_total 2
	`

	if err := testutil.CollectAndCompare(metrics.SynopsisGeneratedTotal, strings.NewReader(expectedSynopsisMetric)); err != nil {
		t.Errorf("unexpected synopsis metric value: %v", err)
	}
}

// TestTextAnalyzerMetrics_AnalysisDuration tests analysis duration histogram
func TestTextAnalyzerMetrics_AnalysisDuration(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("textanalyzer")

	// Record analysis durations
	metrics.AnalysisDuration.WithLabelValues("success").Observe(2.5)
	metrics.AnalysisDuration.WithLabelValues("success").Observe(3.1)
	metrics.AnalysisDuration.WithLabelValues("error").Observe(0.2)

	// Verify histogram has observations
	reg := prometheus.DefaultRegisterer.(*prometheus.Registry)
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, family := range families {
		if family.GetName() == "docutab_analysis_duration_seconds" {
			found = true
			if len(family.GetMetric()) == 0 {
				t.Error("Expected histogram to have metrics")
			}
		}
	}

	if !found {
		t.Error("docutab_analysis_duration_seconds metric not found")
	}
}

// TestSchedulerMetrics tests scheduler business metrics
func TestSchedulerMetrics(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	metrics := NewBusinessMetrics("scheduler")

	// Record scheduled tasks
	metrics.TasksScheduledTotal.WithLabelValues("scrape").Inc()
	metrics.TasksScheduledTotal.WithLabelValues("analysis").Inc()
	metrics.TasksScheduledTotal.WithLabelValues("analysis").Inc()

	// Record executed tasks
	metrics.TasksExecutedTotal.WithLabelValues("scrape", "success").Inc()
	metrics.TasksExecutedTotal.WithLabelValues("analysis", "success").Inc()
	metrics.TasksExecutedTotal.WithLabelValues("analysis", "error").Inc()

	// Record task failures
	metrics.TaskFailuresTotal.WithLabelValues("analysis", "timeout").Inc()

	// Set active tasks
	metrics.ActiveTasks.Set(3)

	// Verify scheduled tasks
	expectedScheduled := `
		# HELP docutab_tasks_scheduled_total Total number of tasks scheduled
		# TYPE docutab_tasks_scheduled_total counter
		docutab_tasks_scheduled_total{type="scrape"} 1
		docutab_tasks_scheduled_total{type="analysis"} 2
	`

	if err := testutil.CollectAndCompare(metrics.TasksScheduledTotal, strings.NewReader(expectedScheduled)); err != nil {
		t.Errorf("unexpected scheduled tasks metric: %v", err)
	}

	// Verify active tasks
	expectedActive := `
		# HELP docutab_active_tasks Number of currently active tasks
		# TYPE docutab_active_tasks gauge
		docutab_active_tasks 3
	`

	if err := testutil.CollectAndCompare(metrics.ActiveTasks, strings.NewReader(expectedActive)); err != nil {
		t.Errorf("unexpected active tasks metric: %v", err)
	}
}
