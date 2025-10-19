package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	controllerURL    = "http://localhost:18080"
	scraperURL       = "http://localhost:18081"
	textAnalyzerURL  = "http://localhost:18082"
)

// TestControllerIntegration tests the full integration between controller and services
func TestControllerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	services := NewTestServices(t)
	defer services.StopAll()

	// Build all services
	t.Log("Building services...")
	scraperBin := BuildService(t, "apps/scraper", "scraper-api")
	analyzerBin := BuildService(t, "apps/textanalyzer", "textanalyzer")
	controllerBin := BuildService(t, "apps/controller", "controller")

	// Check if Ollama is available
	ollamaAvailable := services.CheckOllamaAvailable()
	if ollamaAvailable {
		t.Log("✓ Ollama is available - will test AI-enhanced features")
	} else {
		t.Log("✗ Ollama is not available - will test graceful degradation")
	}

	// Start services in order
	scraperConfig := ServiceConfig{
		Name:        "scraper",
		Port:        18081,
		BinaryPath:  scraperBin,
		Args:        []string{"-port", "18081", "-db", services.GetDBPath("scraper")},
		Env:         []string{"OLLAMA_URL=" + services.GetOllamaURL()},
		HealthCheck: scraperURL + "/health",
	}

	analyzerConfig := ServiceConfig{
		Name:        "textanalyzer",
		Port:        18082,
		BinaryPath:  analyzerBin,
		Args:        []string{"-port", "18082", "-db", services.GetDBPath("textanalyzer")},
		Env:         []string{"OLLAMA_URL=" + services.GetOllamaURL()},
		HealthCheck: textAnalyzerURL + "/health",
	}

	controllerConfig := ServiceConfig{
		Name:        "controller",
		Port:        18080,
		BinaryPath:  controllerBin,
		Env:         []string{
			"CONTROLLER_PORT=18080",
			"SCRAPER_BASE_URL=" + scraperURL,
			"TEXTANALYZER_BASE_URL=" + textAnalyzerURL,
			"DATABASE_PATH=" + services.GetDBPath("controller"),
		},
		HealthCheck: controllerURL + "/health",
	}

	// Start all services
	if err := services.StartService(scraperConfig); err != nil {
		t.Fatalf("Failed to start scraper: %v", err)
	}

	if err := services.StartService(analyzerConfig); err != nil {
		t.Fatalf("Failed to start textanalyzer: %v", err)
	}

	if err := services.StartService(controllerConfig); err != nil {
		t.Fatalf("Failed to start controller: %v", err)
	}

	// Run test suite
	t.Run("DirectTextAnalysis", func(t *testing.T) {
		testDirectTextAnalysis(t, ollamaAvailable)
	})

	t.Run("URLScrapeAndAnalysis", func(t *testing.T) {
		testURLScrapeAndAnalysis(t, ollamaAvailable)
	})

	t.Run("TagSearch", func(t *testing.T) {
		testTagSearch(t)
	})

	t.Run("RequestRetrieval", func(t *testing.T) {
		testRequestRetrieval(t)
	})

	t.Run("FuzzyImageTagSearch", func(t *testing.T) {
		testFuzzyImageTagSearch(t, ollamaAvailable)
	})

	t.Run("DocumentImages", func(t *testing.T) {
		testDocumentImages(t, ollamaAvailable)
	})

	t.Run("LinkScoring", func(t *testing.T) {
		testLinkScoring(t, ollamaAvailable)
	})

	t.Run("AutomaticScoringOnScrape", func(t *testing.T) {
		testAutomaticScoringOnScrape(t, ollamaAvailable)
	})

	t.Run("AsyncTextAnalysis", func(t *testing.T) {
		testAsyncTextAnalysis(t, ollamaAvailable)
	})
}

// testDirectTextAnalysis tests POST /analyze endpoint
func testDirectTextAnalysis(t *testing.T, ollamaAvailable bool) {
	testText := `Climate change is a pressing global issue that affects millions of people worldwide.
Scientists estimate that 75% of species may face extinction if temperatures rise by 2 degrees Celsius.
This is a critical challenge that requires immediate action from governments and citizens alike.`

	reqBody := map[string]interface{}{
		"text": testText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(controllerURL+"/api/analyze", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify required fields are present
	assertFieldExists(t, result, "id")
	assertFieldExists(t, result, "created_at")
	assertFieldExists(t, result, "source_type")
	assertFieldExists(t, result, "textanalyzer_uuid")
	assertFieldExists(t, result, "tags")
	assertFieldExists(t, result, "metadata")

	// Verify source type
	if result["source_type"] != "text" {
		t.Errorf("Expected source_type 'text', got '%v'", result["source_type"])
	}

	// Verify metadata structure
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	assertFieldExists(t, metadata, "analyzer_metadata")

	// Verify analyzer_metadata has expected fields
	analyzerMeta, ok := metadata["analyzer_metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("analyzer_metadata is not a map")
	}

	// Core metadata fields that should always be present
	assertFieldExists(t, analyzerMeta, "word_count")
	assertFieldExists(t, analyzerMeta, "sentiment")
	assertFieldExists(t, analyzerMeta, "readability_score")
	assertFieldExists(t, analyzerMeta, "tags")

	// Check tags array is populated
	tags, ok := result["tags"].([]interface{})
	if !ok {
		t.Fatal("tags is not an array")
	}

	if len(tags) == 0 {
		t.Error("Expected tags array to have at least one element")
	}

	// If Ollama is available, check for AI-enhanced fields
	if ollamaAvailable {
		t.Log("Checking for AI-enhanced metadata fields...")
		// Note: These may not always be present depending on Ollama availability
		// We just log if they're present, don't fail if missing
		if _, exists := analyzerMeta["synopsis"]; exists {
			t.Log("✓ Found AI-generated synopsis")
		}
		if _, exists := analyzerMeta["ai_detection"]; exists {
			t.Log("✓ Found AI detection metadata")
		}
	}

	t.Logf("✓ Direct text analysis completed successfully (request ID: %v)", result["id"])
}

// testURLScrapeAndAnalysis tests POST /scrape endpoint
func testURLScrapeAndAnalysis(t *testing.T, ollamaAvailable bool) {
	// Use example.com as a reliable test URL
	testURL := "https://example.com"

	reqBody := map[string]interface{}{
		"url": testURL,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Note: This might take longer due to scraping + AI processing
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(controllerURL+"/api/scrape", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify required fields
	assertFieldExists(t, result, "id")
	assertFieldExists(t, result, "created_at")
	assertFieldExists(t, result, "source_type")
	assertFieldExists(t, result, "source_url")
	assertFieldExists(t, result, "tags")
	assertFieldExists(t, result, "metadata")

	// Verify source type and URL
	if result["source_type"] != "url" {
		t.Errorf("Expected source_type 'url', got '%v'", result["source_type"])
	}

	if result["source_url"] != testURL {
		t.Errorf("Expected source_url '%s', got '%v'", testURL, result["source_url"])
	}

	// Verify metadata structure
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	// Verify link_score is present in metadata
	assertFieldExists(t, metadata, "link_score")
	linkScore, ok := metadata["link_score"].(map[string]interface{})
	if !ok {
		t.Fatal("link_score is not a map")
	}

	// Verify link_score has expected fields
	assertFieldExists(t, linkScore, "score")
	assertFieldExists(t, linkScore, "reason")
	assertFieldExists(t, linkScore, "categories")
	assertFieldExists(t, linkScore, "is_recommended")

	// Verify score is a valid number between 0.0 and 1.0
	scoreValue, ok := linkScore["score"].(float64)
	if !ok {
		t.Error("link_score.score is not a float64")
	} else if scoreValue < 0.0 || scoreValue > 1.0 {
		t.Errorf("link_score.score should be between 0.0 and 1.0, got %f", scoreValue)
	}

	// Check if the URL was fully processed based on score
	if scoreValue >= 0.5 {
		t.Logf("✓ High-quality URL (score: %.2f) - checking for full processing", scoreValue)

		// Should have scraper and analyzer UUIDs
		assertFieldExists(t, result, "scraper_uuid")
		assertFieldExists(t, result, "textanalyzer_uuid")

		// Verify both UUIDs are present and different
		scraperUUID, ok1 := result["scraper_uuid"].(string)
		analyzerUUID, ok2 := result["textanalyzer_uuid"].(string)

		if !ok1 || !ok2 {
			t.Fatal("UUIDs are not strings")
		}

		if scraperUUID == "" || analyzerUUID == "" {
			t.Error("UUIDs should not be empty")
		}

		// Should have both scraper and analyzer metadata
		assertFieldExists(t, metadata, "scraper_metadata")
		assertFieldExists(t, metadata, "analyzer_metadata")

		// Verify scraper_metadata
		scraperMeta, ok := metadata["scraper_metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("scraper_metadata is not a map")
		}

		// Note: scraper_metadata may be empty if the scraper doesn't return title/content
		if len(scraperMeta) > 0 {
			t.Logf("scraper_metadata has %d fields", len(scraperMeta))
		} else {
			t.Logf("scraper_metadata is empty (content was passed to analyzer)")
		}

		// Verify analyzer_metadata
		analyzerMeta, ok := metadata["analyzer_metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("analyzer_metadata is not a map")
		}

		assertFieldExists(t, analyzerMeta, "word_count")
		assertFieldExists(t, analyzerMeta, "sentiment")

		// If Ollama is available, verify quality_score in analyzer_metadata
		if ollamaAvailable {
			if qualityScore, exists := analyzerMeta["quality_score"]; exists {
				qualityScoreMap, ok := qualityScore.(map[string]interface{})
				if !ok {
					t.Log("⚠ quality_score exists but is not a map")
				} else {
					assertFieldExists(t, qualityScoreMap, "score")
					assertFieldExists(t, qualityScoreMap, "reason")
					assertFieldExists(t, qualityScoreMap, "is_recommended")

					// Verify quality score is within valid range
					qScore, ok := qualityScoreMap["score"].(float64)
					if !ok {
						t.Error("quality_score.score is not a float64")
					} else if qScore < 0.0 || qScore > 1.0 {
						t.Errorf("quality_score.score should be between 0.0 and 1.0, got %f", qScore)
					} else {
						t.Logf("✓ Text quality score: %.2f (recommended: %v)", qScore, qualityScoreMap["is_recommended"])
					}
				}
			} else {
				t.Log("⚠ quality_score not present in analyzer_metadata (Ollama may have failed)")
			}
		}

		// Should NOT have below_threshold flag
		if belowThreshold, exists := metadata["below_threshold"]; exists && belowThreshold == true {
			t.Error("Expected below_threshold to be false or absent for high-quality URL")
		}
	} else {
		t.Logf("✓ Low-quality URL (score: %.2f) - checking for metadata-only response", scoreValue)

		// Should NOT have scraper or analyzer UUIDs
		if scraperUUID, exists := result["scraper_uuid"]; exists && scraperUUID != nil && scraperUUID != "" {
			t.Error("Expected scraper_uuid to be nil or empty for low-quality URL")
		}
		if analyzerUUID, exists := result["textanalyzer_uuid"]; exists && analyzerUUID != nil && analyzerUUID != "" {
			t.Error("Expected textanalyzer_uuid to be nil or empty for low-quality URL")
		}

		// Should have below_threshold flag
		assertFieldExists(t, metadata, "below_threshold")
		if belowThreshold, ok := metadata["below_threshold"].(bool); !ok || !belowThreshold {
			t.Error("Expected below_threshold to be true for low-quality URL")
		}

		// Should NOT have scraper or analyzer metadata
		if _, exists := metadata["scraper_metadata"]; exists {
			t.Error("Expected scraper_metadata to be absent for low-quality URL")
		}
		if _, exists := metadata["analyzer_metadata"]; exists {
			t.Error("Expected analyzer_metadata to be absent for low-quality URL")
		}
	}

	t.Logf("✓ URL scrape and analysis completed successfully (request ID: %v, link score: %.2f)", result["id"], scoreValue)
}

// testTagSearch tests POST /search/tags endpoint
func testTagSearch(t *testing.T) {
	// First, create some content with known tags
	testText := "This is a very positive and happy message about technology and programming!"

	reqBody := map[string]interface{}{
		"text": testText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(controllerURL+"/api/analyze", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	// Small delay to ensure data is committed
	time.Sleep(500 * time.Millisecond)

	// Now search for tags
	searchReq := map[string]interface{}{
		"tags":  []string{"positive"},
		"fuzzy": false,
	}

	searchBody, err := json.Marshal(searchReq)
	if err != nil {
		t.Fatalf("Failed to marshal search request: %v", err)
	}

	searchResp, err := http.Post(controllerURL+"/api/search", "application/json", bytes.NewReader(searchBody))
	if err != nil {
		t.Fatalf("Search request failed: %v", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(searchResp.Body)
		t.Fatalf("Expected status 200, got %d: %s", searchResp.StatusCode, string(bodyBytes))
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		t.Fatalf("Failed to decode search response: %v", err)
	}

	assertFieldExists(t, searchResult, "request_ids")
	assertFieldExists(t, searchResult, "count")

	count, ok := searchResult["count"].(float64)
	if !ok {
		t.Fatal("count is not a number")
	}

	if count < 1 {
		t.Error("Expected at least one search result")
	}

	t.Logf("✓ Tag search completed successfully (found %v results)", count)
}

// testRequestRetrieval tests GET /requests/{id} and GET /requests endpoints
func testRequestRetrieval(t *testing.T) {
	// First, create a request
	testText := "Sample text for retrieval testing."

	reqBody := map[string]interface{}{
		"text": testText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(controllerURL+"/api/analyze", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var createResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	requestID, ok := createResult["id"].(string)
	if !ok {
		t.Fatal("id is not a string")
	}

	// Test GET /requests/{id}
	getResp, err := http.Get(fmt.Sprintf("%s/api/requests/%s", controllerURL, requestID))
	if err != nil {
		t.Fatalf("Get request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", getResp.StatusCode)
	}

	var getResult map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
		t.Fatalf("Failed to decode get response: %v", err)
	}

	if getResult["id"] != requestID {
		t.Errorf("Expected id '%s', got '%v'", requestID, getResult["id"])
	}

	// Test GET /requests (list)
	listResp, err := http.Get(controllerURL + "/api/requests?limit=10&offset=0")
	if err != nil {
		t.Fatalf("List request failed: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", listResp.StatusCode)
	}

	var listResult map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	assertFieldExists(t, listResult, "requests")
	assertFieldExists(t, listResult, "count")

	t.Log("✓ Request retrieval completed successfully")
}

// testFuzzyImageTagSearch tests POST /api/images/search endpoint
func testFuzzyImageTagSearch(t *testing.T, ollamaAvailable bool) {
	// First, scrape a URL that contains images
	// Use example.com as our test URL
	testURL := "https://example.com"

	reqBody := map[string]interface{}{
		"url": testURL,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Scrape the URL (this will also process images if Ollama is available)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(controllerURL+"/api/scrape", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Small delay to ensure images are committed to database
	time.Sleep(500 * time.Millisecond)

	// Now test the fuzzy image search endpoint
	// We'll search for common tags that might be present
	searchReq := map[string]interface{}{
		"tags": []string{"example", "domain", "illustration"},
	}

	searchBody, err := json.Marshal(searchReq)
	if err != nil {
		t.Fatalf("Failed to marshal search request: %v", err)
	}

	searchResp, err := client.Post(controllerURL+"/api/images/search", "application/json", bytes.NewReader(searchBody))
	if err != nil {
		t.Fatalf("Image search request failed: %v", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(searchResp.Body)
		t.Fatalf("Expected status 200, got %d: %s", searchResp.StatusCode, string(bodyBytes))
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		t.Fatalf("Failed to decode search response: %v", err)
	}

	// Verify response structure
	assertFieldExists(t, searchResult, "images")
	assertFieldExists(t, searchResult, "count")

	count, ok := searchResult["count"].(float64)
	if !ok {
		t.Fatal("count is not a number")
	}

	images, ok := searchResult["images"].([]interface{})
	if !ok {
		t.Fatal("images is not an array")
	}

	if ollamaAvailable {
		// If Ollama is available, we should get images with tags
		if count > 0 {
			t.Logf("✓ Fuzzy image tag search found %v images", count)

			// Verify first image has expected fields
			if len(images) > 0 {
				firstImage, ok := images[0].(map[string]interface{})
				if !ok {
					t.Fatal("First image is not a map")
				}

				assertFieldExists(t, firstImage, "id")
				assertFieldExists(t, firstImage, "url")
				assertFieldExists(t, firstImage, "tags")

				tags, ok := firstImage["tags"].([]interface{})
				if ok && len(tags) > 0 {
					t.Logf("✓ Image has tags: %v", tags)
				}
			}
		} else {
			t.Log("⚠ No images found (this may be expected if example.com has no images or image analysis failed)")
		}
	} else {
		// If Ollama is not available, images won't have AI-generated tags
		t.Log("✗ Ollama not available - images won't have AI-generated tags")
		t.Logf("Search returned %v images (may be 0 without tags)", count)
	}

	// Test empty tags validation
	emptySearchReq := map[string]interface{}{
		"tags": []string{},
	}
	emptyBody, _ := json.Marshal(emptySearchReq)
	emptyResp, err := client.Post(controllerURL+"/api/images/search", "application/json", bytes.NewReader(emptyBody))
	if err != nil {
		t.Fatalf("Empty search request failed: %v", err)
	}
	defer emptyResp.Body.Close()

	if emptyResp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty tags, got %d", emptyResp.StatusCode)
	}

	t.Log("✓ Fuzzy image tag search completed successfully")
}

// testDocumentImages tests retrieving images associated with a document
func testDocumentImages(t *testing.T, ollamaAvailable bool) {
	// First, scrape a URL to create a document with images
	testURL := "https://example.com"

	reqBody := map[string]interface{}{
		"url": testURL,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Scrape the URL
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(controllerURL+"/api/scrape", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var scrapeResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&scrapeResult); err != nil {
		t.Fatalf("Failed to decode scrape response: %v", err)
	}

	// Check if the URL was scored and if it met the quality threshold
	metadata, ok := scrapeResult["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	// Get link score if present
	var scoreValue float64
	if linkScore, exists := metadata["link_score"].(map[string]interface{}); exists {
		if score, ok := linkScore["score"].(float64); ok {
			scoreValue = score
		}
	}

	// Get the scraper UUID from the response - it may not exist if URL scored low
	scraperUUID, ok := scrapeResult["scraper_uuid"].(string)
	if !ok || scraperUUID == "" {
		// Check if this was a low-quality URL that wasn't fully processed
		if scoreValue < 0.5 {
			t.Logf("✓ URL scored %.2f (below threshold) - skipping document images test", scoreValue)
			t.Skip("URL scored below threshold, no document to retrieve images from")
		}
		t.Fatal("scraper_uuid not found in scrape response for high-quality URL")
	}

	t.Logf("Scraped document with scraper UUID: %s", scraperUUID)

	// Small delay to ensure images are committed to database
	time.Sleep(500 * time.Millisecond)

	// Now test the document images endpoint
	imagesURL := fmt.Sprintf("%s/api/documents/%s/images", controllerURL, scraperUUID)
	imagesResp, err := client.Get(imagesURL)
	if err != nil {
		t.Fatalf("Document images request failed: %v", err)
	}
	defer imagesResp.Body.Close()

	if imagesResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(imagesResp.Body)
		t.Fatalf("Expected status 200, got %d: %s", imagesResp.StatusCode, string(bodyBytes))
	}

	var imagesResult map[string]interface{}
	if err := json.NewDecoder(imagesResp.Body).Decode(&imagesResult); err != nil {
		t.Fatalf("Failed to decode images response: %v", err)
	}

	// Verify response structure
	assertFieldExists(t, imagesResult, "images")
	assertFieldExists(t, imagesResult, "count")

	images, ok := imagesResult["images"].([]interface{})
	if !ok {
		t.Fatal("images is not an array")
	}

	count, ok := imagesResult["count"].(float64)
	if !ok {
		t.Fatal("count is not a number")
	}

	if int(count) != len(images) {
		t.Errorf("Count %v doesn't match images array length %d", count, len(images))
	}

	t.Logf("Document has %v associated images", count)

	if ollamaAvailable && len(images) > 0 {
		// Verify image structure
		firstImage, ok := images[0].(map[string]interface{})
		if !ok {
			t.Fatal("First image is not a map")
		}

		assertFieldExists(t, firstImage, "id")
		assertFieldExists(t, firstImage, "url")

		// These fields may exist if image analysis was successful
		if _, exists := firstImage["tags"]; exists {
			t.Log("✓ Image has AI-generated tags")
		}
		if _, exists := firstImage["summary"]; exists {
			t.Log("✓ Image has AI-generated summary")
		}
	}

	// Test with invalid UUID
	invalidURL := fmt.Sprintf("%s/api/documents/invalid-uuid/images", controllerURL)
	invalidResp, err := client.Get(invalidURL)
	if err != nil {
		t.Fatalf("Invalid UUID request failed: %v", err)
	}
	defer invalidResp.Body.Close()

	// Should return 200 with empty images array, not an error
	if invalidResp.StatusCode != http.StatusOK {
		t.Logf("Note: Invalid UUID returned status %d (may vary by implementation)", invalidResp.StatusCode)
	}

	t.Log("✓ Document images retrieval completed successfully")
}

// testLinkScoring tests the /api/score endpoint
func testLinkScoring(t *testing.T, ollamaAvailable bool) {
	if !ollamaAvailable {
		t.Skip("Skipping link scoring test - Ollama not available")
	}

	client := &http.Client{Timeout: 60 * time.Second}

	// Test scoring a good URL (example.com should score reasonably well)
	scoreReq := map[string]interface{}{
		"url": "https://example.com",
	}
	body, err := json.Marshal(scoreReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(controllerURL+"/api/score", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Score request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	assertFieldExists(t, result, "url")
	assertFieldExists(t, result, "score")
	assertFieldExists(t, result, "meets_threshold")
	assertFieldExists(t, result, "threshold")

	// Verify score object
	score, ok := result["score"].(map[string]interface{})
	if !ok {
		t.Fatal("score is not a map")
	}

	assertFieldExists(t, score, "score")
	assertFieldExists(t, score, "reason")
	assertFieldExists(t, score, "categories")
	assertFieldExists(t, score, "is_recommended")
	assertFieldExists(t, score, "malicious_indicators")

	// Verify score is a valid number between 0 and 1
	scoreValue, ok := score["score"].(float64)
	if !ok {
		t.Fatal("score value is not a number")
	}
	if scoreValue < 0.0 || scoreValue > 1.0 {
		t.Errorf("Score should be between 0.0 and 1.0, got %f", scoreValue)
	}

	// Verify threshold
	threshold, ok := result["threshold"].(float64)
	if !ok {
		t.Fatal("threshold is not a number")
	}
	if threshold != 0.5 {
		t.Logf("Note: Threshold is %f (expected default 0.5)", threshold)
	}

	t.Logf("✓ Link scoring completed successfully (score: %.2f, meets_threshold: %v)",
		scoreValue, result["meets_threshold"])
}

// testAutomaticScoringOnScrape tests that scrape requests automatically score URLs
func testAutomaticScoringOnScrape(t *testing.T, ollamaAvailable bool) {
	if !ollamaAvailable {
		t.Skip("Skipping automatic scoring test - Ollama not available")
	}

	client := &http.Client{Timeout: 120 * time.Second}

	// Test scraping example.com - should have high score and be fully processed
	scrapeReq := map[string]interface{}{
		"url": "https://example.com",
	}
	body, err := json.Marshal(scrapeReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(controllerURL+"/api/scrape", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Scrape request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify basic response structure
	assertFieldExists(t, result, "id")
	assertFieldExists(t, result, "metadata")

	// Verify metadata contains link_score
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	assertFieldExists(t, metadata, "link_score")
	linkScore, ok := metadata["link_score"].(map[string]interface{})
	if !ok {
		t.Fatal("link_score is not a map")
	}

	// Verify link score structure
	assertFieldExists(t, linkScore, "score")
	assertFieldExists(t, linkScore, "reason")
	assertFieldExists(t, linkScore, "categories")

	scoreValue, ok := linkScore["score"].(float64)
	if !ok {
		t.Fatal("score value is not a number")
	}

	// Check if this was a high-quality URL that was fully processed
	if scoreValue >= 0.5 {
		t.Logf("✓ High-quality URL (score: %.2f) - checking for full processing", scoreValue)

		// Should have scraper and analyzer data
		if _, exists := result["scraper_uuid"]; !exists {
			t.Error("Expected scraper_uuid for high-quality URL")
		}
		if _, exists := result["textanalyzer_uuid"]; !exists {
			t.Error("Expected textanalyzer_uuid for high-quality URL")
		}

		// Should have both scraper and analyzer metadata
		assertFieldExists(t, metadata, "scraper_metadata")
		assertFieldExists(t, metadata, "analyzer_metadata")

		// Should NOT have below_threshold flag
		if belowThreshold, exists := metadata["below_threshold"]; exists && belowThreshold == true {
			t.Error("Expected below_threshold to be false or absent for high-quality URL")
		}
	} else {
		t.Logf("✓ Low-quality URL (score: %.2f) - checking for metadata-only response", scoreValue)

		// Should NOT have scraper or analyzer UUIDs
		if scraperUUID, exists := result["scraper_uuid"]; exists && scraperUUID != nil {
			t.Error("Expected scraper_uuid to be nil for low-quality URL")
		}
		if analyzerUUID, exists := result["textanalyzer_uuid"]; exists && analyzerUUID != "" {
			t.Error("Expected textanalyzer_uuid to be empty for low-quality URL")
		}

		// Should have below_threshold flag
		assertFieldExists(t, metadata, "below_threshold")
		if belowThreshold, ok := metadata["below_threshold"].(bool); !ok || !belowThreshold {
			t.Error("Expected below_threshold to be true for low-quality URL")
		}

		// Should NOT have scraper or analyzer metadata
		if _, exists := metadata["scraper_metadata"]; exists {
			t.Error("Expected scraper_metadata to be absent for low-quality URL")
		}
		if _, exists := metadata["analyzer_metadata"]; exists {
			t.Error("Expected analyzer_metadata to be absent for low-quality URL")
		}
	}

	t.Logf("✓ Automatic scoring on scrape completed successfully (score: %.2f)", scoreValue)
}

// testAsyncTextAnalysis tests POST /api/analyze-requests endpoint (async text analysis)
func testAsyncTextAnalysis(t *testing.T, ollamaAvailable bool) {
	testText := `Async text analysis test: This text will be processed asynchronously.
The system should create a request, process it in the background, and allow polling for status.`

	// Step 1: Create async text analysis request
	reqBody := map[string]interface{}{
		"text": testText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(controllerURL+"/api/analyze-requests", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var createResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Verify create response
	assertFieldExists(t, createResult, "id")
	assertFieldExists(t, createResult, "source_type")
	assertFieldExists(t, createResult, "status")
	assertFieldExists(t, createResult, "progress")
	assertFieldExists(t, createResult, "text")

	if createResult["source_type"] != "text" {
		t.Errorf("Expected source_type 'text', got '%v'", createResult["source_type"])
	}

	if createResult["text"] != testText {
		t.Errorf("Expected text to match input")
	}

	requestID, ok := createResult["id"].(string)
	if !ok || requestID == "" {
		t.Fatal("Expected non-empty request ID")
	}

	t.Logf("Created async text analysis request: %s", requestID)

	// Step 2: Poll for completion
	maxAttempts := 30
	pollInterval := 200 * time.Millisecond
	var finalStatus map[string]interface{}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(pollInterval)

		getResp, err := http.Get(controllerURL + "/api/scrape-requests/" + requestID)
		if err != nil {
			t.Fatalf("Failed to get request status: %v", err)
		}

		if getResp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(getResp.Body)
			getResp.Body.Close()
			t.Fatalf("Expected status 200, got %d: %s", getResp.StatusCode, string(bodyBytes))
		}

		if err := json.NewDecoder(getResp.Body).Decode(&finalStatus); err != nil {
			getResp.Body.Close()
			t.Fatalf("Failed to decode status response: %v", err)
		}
		getResp.Body.Close()

		status, ok := finalStatus["status"].(string)
		if !ok {
			t.Fatal("Status field missing or not a string")
		}

		progress, _ := finalStatus["progress"].(float64)
		t.Logf("Attempt %d: status=%s, progress=%.0f%%", attempt+1, status, progress)

		if status == "completed" {
			t.Log("✓ Request completed successfully")
			break
		} else if status == "failed" {
			errorMsg := finalStatus["error_message"]
			t.Fatalf("Request failed: %v", errorMsg)
		}

		if attempt == maxAttempts-1 {
			t.Fatal("Request did not complete within timeout")
		}
	}

	// Step 3: Verify completion
	assertFieldExists(t, finalStatus, "result_request_id")

	resultRequestID, ok := finalStatus["result_request_id"].(string)
	if !ok || resultRequestID == "" {
		t.Fatal("Expected non-empty result_request_id")
	}

	t.Logf("Result request ID: %s", resultRequestID)

	// Step 4: Retrieve the actual result
	resultResp, err := http.Get(controllerURL + "/api/requests/" + resultRequestID)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}
	defer resultResp.Body.Close()

	if resultResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resultResp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resultResp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resultResp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode result: %v", err)
	}

	// Verify result structure
	assertFieldExists(t, result, "id")
	assertFieldExists(t, result, "source_type")
	assertFieldExists(t, result, "textanalyzer_uuid")
	assertFieldExists(t, result, "tags")
	assertFieldExists(t, result, "metadata")

	if result["source_type"] != "text" {
		t.Errorf("Expected source_type 'text', got '%v'", result["source_type"])
	}

	// Verify metadata
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	assertFieldExists(t, metadata, "analyzer_metadata")

	// Step 5: Test listing requests (should include our request)
	listResp, err := http.Get(controllerURL + "/api/scrape-requests")
	if err != nil {
		t.Fatalf("Failed to list requests: %v", err)
	}
	defer listResp.Body.Close()

	var listResult map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	requests, ok := listResult["requests"].([]interface{})
	if !ok {
		t.Fatal("requests is not an array")
	}

	// Find our request in the list
	found := false
	for _, req := range requests {
		reqMap, ok := req.(map[string]interface{})
		if !ok {
			continue
		}
		if reqMap["id"] == requestID {
			found = true
			if reqMap["source_type"] != "text" {
				t.Error("Expected source_type 'text' in list")
			}
			break
		}
	}

	if found {
		t.Log("✓ Found text analysis request in list")
	} else {
		t.Log("Request not found in list (may have been auto-deleted)")
	}

	// Step 6: Test delete
	deleteReq, err := http.NewRequest(http.MethodDelete, controllerURL+"/api/scrape-requests/"+requestID, nil)
	if err != nil {
		t.Fatalf("Failed to create delete request: %v", err)
	}

	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete request: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(deleteResp.Body)
		t.Fatalf("Expected status 200, got %d: %s", deleteResp.StatusCode, string(bodyBytes))
	}

	t.Log("✓ Successfully deleted async text analysis request")

	t.Logf("✓ Async text analysis workflow completed successfully")
}

// Helper function to assert a field exists in a map
func assertFieldExists(t *testing.T, m map[string]interface{}, field string) {
	t.Helper()
	if _, exists := m[field]; !exists {
		t.Errorf("Expected field '%s' to exist in response", field)
	}
}
