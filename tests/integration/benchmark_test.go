package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	benchControllerURL   = "http://localhost:18080"
	benchScraperURL      = "http://localhost:18081"
	benchTextAnalyzerURL = "http://localhost:18082"
)

// BenchmarkResult holds benchmark statistics
type BenchmarkResult struct {
	TotalRequests     int64
	SuccessfulReqs    int64
	FailedReqs        int64
	TotalDuration     time.Duration
	AvgResponseTime   time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	RequestsPerSecond float64
}

// TestBenchmarkControllerLoad runs load tests on the controller
func TestBenchmarkControllerLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark tests in short mode")
	}

	// This test requires the BENCHMARK=true environment variable
	// to prevent accidental heavy load testing
	if !shouldRunBenchmark() {
		t.Skip("Skipping benchmark - set BENCHMARK=true to run")
	}

	services := NewTestServices(t)
	defer services.StopAll()

	// Build and start services
	setupBenchmarkServices(t, services)

	// Wait a bit for services to stabilize
	time.Sleep(2 * time.Second)

	t.Run("BenchmarkDirectAnalysis", func(t *testing.T) {
		benchmarkDirectAnalysis(t, 30, 2) // 30 requests, 2 concurrent workers
	})

	t.Run("BenchmarkMixedWorkload", func(t *testing.T) {
		benchmarkMixedWorkload(t, 25, 3) // 25 of each type, 3 concurrent workers
	})
}

// setupBenchmarkServices builds and starts all services for benchmarking
func setupBenchmarkServices(t *testing.T, services *TestServices) {
	t.Log("Setting up services for benchmarking...")

	scraperBin := BuildService(t, "apps/scraper", "scraper-api")
	analyzerBin := BuildService(t, "apps/textanalyzer", "textanalyzer")
	controllerBin := BuildService(t, "apps/controller", "controller")

	ollamaAvailable := services.CheckOllamaAvailable()
	if ollamaAvailable {
		t.Log("✓ Ollama is available - benchmarking with AI features")
	} else {
		t.Log("✗ Ollama is not available - benchmarking without AI features")
	}

	// Get PostgreSQL configuration for services
	pgHost, pgPort, pgUser, pgPass, _ := services.GetPostgresConfig()

	scraperConfig := ServiceConfig{
		Name:       "scraper",
		Port:       18081,
		BinaryPath: scraperBin,
		Args:       []string{"-port", "18081"},
		Env: []string{
			"OLLAMA_URL=" + services.GetOllamaURL(),
			"DB_HOST=" + pgHost,
			"DB_PORT=" + fmt.Sprintf("%d", pgPort),
			"DB_USER=" + pgUser,
			"DB_PASSWORD=" + pgPass,
			"DB_NAME=scraper_db",
		},
		HealthCheck: benchScraperURL + "/health",
	}

	analyzerConfig := ServiceConfig{
		Name:       "textanalyzer",
		Port:       18082,
		BinaryPath: analyzerBin,
		Args:       []string{"-port", "18082"},
		Env: []string{
			"OLLAMA_URL=" + services.GetOllamaURL(),
			"REDIS_ADDR=" + services.GetRedisAddr(),
			"DB_HOST=" + pgHost,
			"DB_PORT=" + fmt.Sprintf("%d", pgPort),
			"DB_USER=" + pgUser,
			"DB_PASSWORD=" + pgPass,
			"DB_NAME=textanalyzer_db",
		},
		HealthCheck: benchTextAnalyzerURL + "/health",
	}

	controllerConfig := ServiceConfig{
		Name:       "controller",
		Port:       18080,
		BinaryPath: controllerBin,
		Env: []string{
			"CONTROLLER_PORT=18080",
			"SCRAPER_BASE_URL=" + benchScraperURL,
			"TEXTANALYZER_BASE_URL=" + benchTextAnalyzerURL,
			"REDIS_ADDR=" + services.GetRedisAddr(),
			"DB_HOST=" + pgHost,
			"DB_PORT=" + fmt.Sprintf("%d", pgPort),
			"DB_USER=" + pgUser,
			"DB_PASSWORD=" + pgPass,
			"DB_NAME=controller_db",
		},
		HealthCheck: benchControllerURL + "/health",
	}

	if err := services.StartService(scraperConfig); err != nil {
		t.Fatalf("Failed to start scraper: %v", err)
	}

	if err := services.StartService(analyzerConfig); err != nil {
		t.Fatalf("Failed to start textanalyzer: %v", err)
	}

	if err := services.StartService(controllerConfig); err != nil {
		t.Fatalf("Failed to start controller: %v", err)
	}
}

// benchmarkDirectAnalysis benchmarks the /analyze endpoint
func benchmarkDirectAnalysis(t *testing.T, totalRequests, concurrency int) {
	t.Logf("Benchmarking direct text analysis: %d requests with %d concurrent workers", totalRequests, concurrency)

	testTexts := []string{
		"This is a short and positive message about technology.",
		"Climate change poses significant challenges. Scientists estimate 75% of species may be affected.",
		"The quick brown fox jumps over the lazy dog. This is a simple sentence for testing purposes.",
		"Artificial intelligence is transforming industries worldwide. Machine learning algorithms are becoming increasingly sophisticated.",
		"Economic indicators suggest moderate growth. Employment rates have increased by 3.5% this quarter.",
	}

	result := runLoadTest(t, totalRequests, concurrency, func(i int) (*http.Response, error) {
		// Rotate through test texts with unique request number integrated into the text
		// to ensure unique slugs even after text cleaning
		text := fmt.Sprintf("Request %d: %s", i, testTexts[i%len(testTexts)])

		reqBody := map[string]interface{}{
			"text": text,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		client := &http.Client{Timeout: 60 * time.Second}
		return client.Post(benchControllerURL+"/api/analyze", "application/json", bytes.NewReader(body))
	})

	printBenchmarkResults(t, "Direct Text Analysis", result)
}

// benchmarkMixedWorkload runs a mix of analyze and scrape requests
func benchmarkMixedWorkload(t *testing.T, requestsPerType, concurrency int) {
	t.Logf("Benchmarking mixed workload: %d requests of each type with %d concurrent workers", requestsPerType, concurrency)

	totalRequests := requestsPerType * 2 // analyze + scrape

	result := runLoadTest(t, totalRequests, concurrency, func(i int) (*http.Response, error) {
		client := &http.Client{Timeout: 120 * time.Second}

		// Alternate between analyze and scrape
		if i%2 == 0 {
			// Text analysis with unique identifier integrated into text to avoid slug conflicts
			reqBody := map[string]interface{}{
				"text": fmt.Sprintf("Request %d sample text for benchmarking purposes", i),
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				return nil, err
			}

			return client.Post(benchControllerURL+"/api/analyze", "application/json", bytes.NewReader(body))
		} else {
			// URL scraping
			reqBody := map[string]interface{}{
				"url": "https://example.com",
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				return nil, err
			}

			return client.Post(benchControllerURL+"/api/scrape", "application/json", bytes.NewReader(body))
		}
	})

	printBenchmarkResults(t, "Mixed Workload (50% analyze, 50% scrape)", result)
}

// runLoadTest executes a load test with the given parameters
func runLoadTest(t *testing.T, totalRequests, concurrency int, requestFunc func(int) (*http.Response, error)) BenchmarkResult {
	var (
		successCount    int64
		failCount       int64
		minResponseTime int64 = 999999999999 // Initialize to large value
		maxResponseTime int64
		totalTime       int64
	)

	startTime := time.Now()

	// Create work queue
	workQueue := make(chan int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		workQueue <- i
	}
	close(workQueue)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := range workQueue {
				reqStart := time.Now()
				resp, err := requestFunc(i)
				reqDuration := time.Since(reqStart).Nanoseconds()

				// Update stats
				atomic.AddInt64(&totalTime, reqDuration)

				// Update min/max response times
				for {
					oldMin := atomic.LoadInt64(&minResponseTime)
					if reqDuration >= oldMin || atomic.CompareAndSwapInt64(&minResponseTime, oldMin, reqDuration) {
						break
					}
				}

				for {
					oldMax := atomic.LoadInt64(&maxResponseTime)
					if reqDuration <= oldMax || atomic.CompareAndSwapInt64(&maxResponseTime, oldMax, reqDuration) {
						break
					}
				}

				if err != nil {
					atomic.AddInt64(&failCount, 1)
					t.Logf("Request %d failed: %v", i, err)
					continue
				}

				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
					atomic.AddInt64(&failCount, 1)
					// Read error response body
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Logf("Request %d returned status %d, body: %s", i, resp.StatusCode, string(bodyBytes))
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	avgResponseTime := time.Duration(0)
	if successCount > 0 {
		avgResponseTime = time.Duration(totalTime / successCount)
	}

	requestsPerSecond := float64(successCount) / totalDuration.Seconds()

	return BenchmarkResult{
		TotalRequests:     int64(totalRequests),
		SuccessfulReqs:    successCount,
		FailedReqs:        failCount,
		TotalDuration:     totalDuration,
		AvgResponseTime:   avgResponseTime,
		MinResponseTime:   time.Duration(minResponseTime),
		MaxResponseTime:   time.Duration(maxResponseTime),
		RequestsPerSecond: requestsPerSecond,
	}
}

// printBenchmarkResults outputs formatted benchmark results
func printBenchmarkResults(t *testing.T, name string, result BenchmarkResult) {
	t.Logf("\n"+
		"========================================\n"+
		"Benchmark Results: %s\n"+
		"========================================\n"+
		"Total Requests:        %d\n"+
		"Successful:            %d (%.1f%%)\n"+
		"Failed:                %d (%.1f%%)\n"+
		"Total Duration:        %v\n"+
		"Average Response Time: %v\n"+
		"Min Response Time:     %v\n"+
		"Max Response Time:     %v\n"+
		"Requests/Second:       %.2f\n"+
		"========================================\n",
		name,
		result.TotalRequests,
		result.SuccessfulReqs,
		float64(result.SuccessfulReqs)/float64(result.TotalRequests)*100,
		result.FailedReqs,
		float64(result.FailedReqs)/float64(result.TotalRequests)*100,
		result.TotalDuration,
		result.AvgResponseTime,
		result.MinResponseTime,
		result.MaxResponseTime,
		result.RequestsPerSecond,
	)

	// Set pass/fail criteria
	// For load tests with multiple services, 85% success rate is acceptable
	successRate := float64(result.SuccessfulReqs) / float64(result.TotalRequests)
	if successRate < 0.85 {
		t.Errorf("Success rate %.1f%% is below 85%% threshold", successRate*100)
	}
}

// shouldRunBenchmark checks if benchmarking should run
func shouldRunBenchmark() bool {
	// Check for BENCHMARK environment variable
	val := getEnvDefault("BENCHMARK", "false")
	return val == "true" || val == "1" || val == "yes"
}

// getEnvDefault returns an environment variable or default value
func getEnvDefault(key, defaultVal string) string {
	if val := getEnv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnv returns an environment variable value
func getEnv(key string) string {
	return os.Getenv(key)
}
