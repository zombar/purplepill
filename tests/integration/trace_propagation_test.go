package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestHTTPTracePropagation tests that trace IDs are properly propagated
// across HTTP requests and logged in structured logs
func TestHTTPTracePropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// This test verifies the fix for the middleware order issue where
	// trace_id was empty in logs because logging middleware executed
	// before tracing middleware created the span

	// Wait for services to be ready
	waitForService(t, "http://localhost:9080/health", "controller", 30*time.Second)
	waitForService(t, "http://localhost:9081/health", "scraper", 30*time.Second)
	waitForService(t, "http://localhost:9082/health", "textanalyzer", 30*time.Second)

	t.Run("ControllerHTTPRequestHasTraceID", func(t *testing.T) {
		// Make a scrape request to controller
		requestBody := map[string]string{
			"url": "https://example.com",
		}
		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		resp, err := http.Post("http://localhost:9080/api/scrape",
			"application/json",
			bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to make scrape request: %v", err)
		}
		defer resp.Body.Close()

		// Request should succeed (or fail gracefully)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusInternalServerError {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}

		// Give logs time to flush
		time.Sleep(500 * time.Millisecond)

		// Check controller logs for trace_id
		logs := getContainerLogs(t, "docutab-controller", 50)

		// Verify at least one HTTP request log has a non-empty trace_id
		foundTraceID := false
		for _, line := range logs {
			if !strings.Contains(line, "http_request") {
				continue
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				continue // Skip non-JSON lines
			}

			// Check if this is an HTTP request log with trace_id
			if logEntry["msg"] == "http_request" {
				traceID, ok := logEntry["trace_id"].(string)
				if ok && traceID != "" && len(traceID) > 10 {
					t.Logf("✓ Found trace_id in logs: %s", traceID)
					foundTraceID = true

					// Also verify span_id is present
					spanID, ok := logEntry["span_id"].(string)
					if !ok || spanID == "" {
						t.Errorf("Found trace_id but span_id is empty or missing")
					} else {
						t.Logf("✓ Found span_id in logs: %s", spanID)
					}
					break
				}
			}
		}

		if !foundTraceID {
			t.Error("No HTTP request logs with valid trace_id found in controller logs")
			t.Log("Sample logs:")
			for i, line := range logs {
				if i >= 5 {
					break
				}
				t.Logf("  %s", line)
			}
		}
	})

	t.Run("ScraperHTTPRequestHasTraceID", func(t *testing.T) {
		// Make a direct request to scraper
		requestBody := map[string]string{
			"url": "https://example.com",
		}
		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		resp, err := http.Post("http://localhost:9081/api/score",
			"application/json",
			bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to make score request: %v", err)
		}
		defer resp.Body.Close()

		// Give logs time to flush
		time.Sleep(500 * time.Millisecond)

		// Check scraper logs for trace_id
		logs := getContainerLogs(t, "docutab-scraper", 50)

		foundTraceID := false
		for _, line := range logs {
			if !strings.Contains(line, "http_request") {
				continue
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				continue
			}

			if logEntry["msg"] == "http_request" {
				traceID, ok := logEntry["trace_id"].(string)
				if ok && traceID != "" && len(traceID) > 10 {
					t.Logf("✓ Found trace_id in scraper logs: %s", traceID)
					foundTraceID = true

					spanID, ok := logEntry["span_id"].(string)
					if !ok || spanID == "" {
						t.Errorf("Found trace_id but span_id is empty or missing")
					} else {
						t.Logf("✓ Found span_id in scraper logs: %s", spanID)
					}
					break
				}
			}
		}

		if !foundTraceID {
			t.Error("No HTTP request logs with valid trace_id found in scraper logs")
		}
	})

	t.Run("TextAnalyzerHTTPRequestHasTraceID", func(t *testing.T) {
		// Make a direct request to textanalyzer
		requestBody := map[string]interface{}{
			"text": "This is a test document for trace propagation testing.",
		}
		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		resp, err := http.Post("http://localhost:9082/api/analyze",
			"application/json",
			bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to make analyze request: %v", err)
		}
		defer resp.Body.Close()

		// Give logs time to flush
		time.Sleep(500 * time.Millisecond)

		// Check textanalyzer logs for trace_id
		logs := getContainerLogs(t, "docutab-textanalyzer", 50)

		foundTraceID := false
		for _, line := range logs {
			if !strings.Contains(line, "http_request") {
				continue
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				continue
			}

			if logEntry["msg"] == "http_request" {
				traceID, ok := logEntry["trace_id"].(string)
				if ok && traceID != "" && len(traceID) > 10 {
					t.Logf("✓ Found trace_id in textanalyzer logs: %s", traceID)
					foundTraceID = true

					spanID, ok := logEntry["span_id"].(string)
					if !ok || spanID == "" {
						t.Errorf("Found trace_id but span_id is empty or missing")
					} else {
						t.Logf("✓ Found span_id in textanalyzer logs: %s", spanID)
					}
					break
				}
			}
		}

		if !foundTraceID {
			t.Error("No HTTP request logs with valid trace_id found in textanalyzer logs")
		}
	})
}

// waitForService waits for a service health endpoint to respond
func waitForService(t *testing.T, url, name string, timeout time.Duration) {
	t.Helper()
	t.Logf("Waiting for %s to be ready at %s...", name, url)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Logf("✓ %s is ready", name)
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Service %s did not become ready within %v", name, timeout)
}

// getContainerLogs retrieves the last N lines of logs from a Docker container
func getContainerLogs(t *testing.T, containerName string, lines int) []string {
	t.Helper()

	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", lines), containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: Failed to get container logs: %v", err)
		return []string{}
	}

	// Split output into lines and filter out empty lines
	logLines := strings.Split(string(output), "\n")
	var result []string
	for _, line := range logLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
