package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/docutag/controller/internal/clients"
	"github.com/docutag/controller/internal/handlers"
	"github.com/docutag/controller/internal/queue"
	"github.com/docutag/controller/internal/storage"
	"github.com/docutag/controller/internal/urlcache"
	"github.com/docutag/platform/pkg/metrics"
)

/**
 * Integration Test Suite: Tombstone Workflow
 *
 * Tests end-to-end tombstone flows including:
 * 1. Low-score URL rejection → tombstone creation → metrics recording
 * 2. Tag-based tombstoning → metrics recording
 * 3. Manual tombstoning → metrics recording
 * 4. Async queue workflow → low-score tombstoning
 */

// setupTestDB is a stub function - TODO: implement PostgreSQL test database setup
func setupTestDB(t *testing.T, dbName string) (string, func()) {
	t.Helper()
	// TODO: Set up test PostgreSQL database
	// For now, skip tests that need this
	t.Skip("setupTestDB not implemented - requires PostgreSQL test infrastructure")
	return "", func() {}
}

// TestIntegration_LowScoreURL_Tombstoning tests complete flow for low-score URL
func TestIntegration_LowScoreURL_Tombstoning(t *testing.T) {
	t.Skip("TODO: Implement setupTestDB helper function")

	// Reset Prometheus registry for clean metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	// Setup test database
	connStr, dbCleanup := setupTestDB(t, "integration_low_score")
	defer dbCleanup()

	// Initialize storage with test config
	store, err := storage.New(connStr, []string{"low-quality", "sparse-content"}, 30, 90, 90)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Initialize business metrics
	businessMetrics := metrics.NewBusinessMetrics("controller")

	// Set up metrics adapter for storage
	metricsAdapter := storage.NewMetricsAdapter(businessMetrics)
	store.SetBusinessMetrics(metricsAdapter)

	// Mock scraper server that returns low score
	scraperMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/score") {
			// Return low score (0.3 < 0.5 threshold)
			response := map[string]interface{}{
				"score": map[string]interface{}{
					"score":           0.3,
					"reason":          "Low-quality content detected",
					"categories":      []string{"low-quality", "spam"},
					"is_recommended":  false,
					"malicious_indicators": []string{},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer scraperMock.Close()

	// Initialize clients and handler
	scraperClient := clients.NewScraperClient(scraperMock.URL)
	textAnalyzerClient := clients.NewTextAnalyzerClient("http://localhost:8082")
	schedulerClient := clients.NewSchedulerClient("http://localhost:8083")
	queueClient := queue.NewClient(queue.ClientConfig{RedisAddr: "localhost:6379"})
	defer queueClient.Close()
	urlCache := urlcache.New("localhost:6379")
	defer urlCache.Close()

	handler := handlers.NewWithMetrics(
		store,
		scraperClient,
		textAnalyzerClient,
		schedulerClient,
		queueClient,
		urlCache,
		0.5, // threshold
		"http://localhost:5173",
		scraperMock.URL,
		30,  // low-score period
		90,  // manual period
		businessMetrics,
	)

	// Make scrape request with low-quality URL
	reqBody := map[string]string{"url": "https://low-quality-site.com/spam"}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/scrape", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ScrapeURL(w, req)

	// Verify HTTP response
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify tombstone metadata
	metadata := response["metadata"].(map[string]interface{})
	if metadata["below_threshold"] != true {
		t.Error("Expected below_threshold to be true")
	}

	tombstoneDateTime := metadata["tombstone_datetime"]
	if tombstoneDateTime == nil {
		t.Fatal("Expected tombstone_datetime in metadata")
	}

	// Verify database record
	requestID := response["id"].(string)
	savedRequest, err := store.GetRequest(requestID)
	if err != nil {
		t.Fatalf("Failed to get saved request: %v", err)
	}

	if savedRequest.SEOEnabled {
		t.Error("Expected SEOEnabled to be false for tombstoned content")
	}

	// Verify metrics were recorded
	metricValue := getCounterValue(t, businessMetrics.TombstonesCreatedTotal, "low-score", "none")
	if metricValue < 1.0 {
		t.Errorf("Expected TombstonesCreatedTotal metric to be >= 1, got %.2f", metricValue)
	}

	histogramCount := getHistogramCount(t, businessMetrics.TombstoneDaysHistogram, "low-score")
	if histogramCount < 1 {
		t.Errorf("Expected TombstoneDaysHistogram count to be >= 1, got %d", histogramCount)
	}

	t.Logf("✅ Low-score URL tombstoning flow verified with metrics")
}

// TestIntegration_TagBased_Tombstoning tests tag-based tombstone creation
func TestIntegration_TagBased_Tombstoning(t *testing.T) {
	t.Skip("TODO: Implement setupTestDB helper function")

	// Reset Prometheus registry
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	// Setup test database
	connStr, dbCleanup := setupTestDB(t, "integration_tag_based")
	defer dbCleanup()

	// Initialize storage with custom tombstone tags
	store, err := storage.New(connStr, []string{"spam", "malicious"}, 30, 60, 90)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Initialize business metrics
	businessMetrics := metrics.NewBusinessMetrics("controller")
	metricsAdapter := storage.NewMetricsAdapter(businessMetrics)
	store.SetBusinessMetrics(metricsAdapter)

	// Create a test request
	sourceURL := "https://example.com/article"
	scraperUUID := "scraper-123"
	req := &storage.Request{
		ID:               "test-req-1",
		CreatedAt:        time.Now(),
		SourceType:       "url",
		SourceURL:        &sourceURL,
		ScraperUUID:      &scraperUUID,
		TextAnalyzerUUID: "analyzer-123",
		Tags:             []string{"technology"},
		Metadata:         map[string]interface{}{"initial": "data"},
		SEOEnabled:       true,
	}

	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	// Update tags to include tombstone trigger tag
	newTags := []string{"technology", "spam", "news"}
	if err := store.UpdateRequestTags(req.ID, newTags); err != nil {
		t.Fatalf("Failed to update tags: %v", err)
	}

	// Verify tombstone was added
	updated, err := store.GetRequest(req.ID)
	if err != nil {
		t.Fatalf("Failed to get updated request: %v", err)
	}

	if _, ok := updated.Metadata["tombstone_datetime"]; !ok {
		t.Fatal("Expected tombstone_datetime after tag update")
	}

	reason := updated.Metadata["tombstone_reason"].(string)
	if !strings.Contains(reason, "spam") {
		t.Errorf("Expected tombstone_reason to mention 'spam', got: %s", reason)
	}

	// Verify metrics
	metricValue := getCounterValue(t, businessMetrics.TombstonesCreatedTotal, "tag-based", "spam")
	if metricValue < 1.0 {
		t.Errorf("Expected tag-based tombstone metric >= 1, got %.2f", metricValue)
	}

	histogramCount := getHistogramCount(t, businessMetrics.TombstoneDaysHistogram, "tag-based")
	if histogramCount < 1 {
		t.Errorf("Expected histogram count >= 1, got %d", histogramCount)
	}

	t.Logf("✅ Tag-based tombstoning flow verified with metrics")
}

// TestIntegration_Manual_Tombstoning tests manual tombstone API endpoint
func TestIntegration_Manual_Tombstoning(t *testing.T) {
	t.Skip("TODO: Implement setupTestDB helper function")

	// Reset Prometheus registry
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	// Setup test database
	connStr, dbCleanup := setupTestDB(t, "integration_manual")
	defer dbCleanup()

	// Initialize storage
	store, err := storage.New(connStr, []string{"low-quality"}, 30, 90, 45)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Initialize business metrics
	businessMetrics := metrics.NewBusinessMetrics("controller")
	metricsAdapter := storage.NewMetricsAdapter(businessMetrics)
	store.SetBusinessMetrics(metricsAdapter)

	// Mock servers
	scraperMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer scraperMock.Close()

	scraperClient := clients.NewScraperClient(scraperMock.URL)
	textAnalyzerClient := clients.NewTextAnalyzerClient("http://localhost:8082")
	schedulerClient := clients.NewSchedulerClient("http://localhost:8083")
	queueClient := queue.NewClient(queue.ClientConfig{RedisAddr: "localhost:6379"})
	defer queueClient.Close()
	urlCache := urlcache.New("localhost:6379")
	defer urlCache.Close()

	handler := handlers.NewWithMetrics(
		store,
		scraperClient,
		textAnalyzerClient,
		schedulerClient,
		queueClient,
		urlCache,
		0.5,
		"http://localhost:5173",
		scraperMock.URL,
		30,
		45, // manual period: 45 days
		businessMetrics,
	)

	// Create a test request
	sourceURL := "https://example.com/article"
	scraperUUID := "scraper-123"
	req := &storage.Request{
		ID:               "test-req-manual",
		CreatedAt:        time.Now(),
		SourceType:       "url",
		SourceURL:        &sourceURL,
		ScraperUUID:      &scraperUUID,
		TextAnalyzerUUID: "analyzer-123",
		Tags:             []string{"technology"},
		Metadata:         map[string]interface{}{},
		SEOEnabled:       true,
	}

	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	// Make manual tombstone request
	tombstoneReq := httptest.NewRequest("PUT", "/api/requests/"+req.ID+"/tombstone", nil)
	w := httptest.NewRecorder()

	handler.TombstoneRequest(w, tombstoneReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify tombstone was added
	updated, err := store.GetRequest(req.ID)
	if err != nil {
		t.Fatalf("Failed to get updated request: %v", err)
	}

	if _, ok := updated.Metadata["tombstone_datetime"]; !ok {
		t.Fatal("Expected tombstone_datetime after manual tombstone")
	}

	// Verify metrics
	metricValue := getCounterValue(t, businessMetrics.TombstonesCreatedTotal, "manual", "none")
	if metricValue < 1.0 {
		t.Errorf("Expected manual tombstone metric >= 1, got %.2f", metricValue)
	}

	histogramCount := getHistogramCount(t, businessMetrics.TombstoneDaysHistogram, "manual")
	if histogramCount < 1 {
		t.Errorf("Expected histogram count >= 1, got %d", histogramCount)
	}

	t.Logf("✅ Manual tombstoning flow verified with metrics")
}

// Helper functions

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, reason, tag string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(reason, tag).Write(metric); err != nil {
		t.Logf("Warning: Could not read counter metric: %v", err)
		return 0
	}
	if metric.Counter == nil {
		return 0
	}
	return metric.Counter.GetValue()
}

func getHistogramCount(t *testing.T, histogram *prometheus.HistogramVec, reason string) uint64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := histogram.WithLabelValues(reason).(prometheus.Metric).Write(metric); err != nil {
		t.Logf("Warning: Could not read histogram metric: %v", err)
		return 0
	}
	if metric.Histogram == nil {
		return 0
	}
	return metric.Histogram.GetSampleCount()
}
