package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockOllamaServer provides a mock Ollama API server for testing
type MockOllamaServer struct {
	server        *http.Server
	port          int
	responseCache map[string]string
	cacheMutex    sync.RWMutex
	requestCount  int64
}

// NewMockOllamaServer creates a new mock Ollama server
func NewMockOllamaServer(port int) *MockOllamaServer {
	mux := http.NewServeMux()

	mock := &MockOllamaServer{
		port:          port,
		responseCache: make(map[string]string),
		server: &http.Server{
			Addr:              formatAddr(port),
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       30 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1 MB
			ReadHeaderTimeout: 2 * time.Second,
		},
	}

	// Register handlers
	mux.HandleFunc("/api/tags", mock.handleTags)
	mux.HandleFunc("/api/generate", mock.handleGenerate)

	// Pre-cache common responses for faster benchmark performance
	mock.preCacheCommonResponses()

	return mock
}

// preCacheCommonResponses pre-generates and caches common responses
func (m *MockOllamaServer) preCacheCommonResponses() {
	commonPrompts := []string{
		"synopsis",
		"clean text",
		"editorial analysis",
		"extract references",
		"AI detection",
		"quality score",
		// Note: tags are NOT pre-cached as they need to be content-specific
	}

	for _, prompt := range commonPrompts {
		response := m.generateMockResponse(prompt)
		cacheKey := getCacheKey(prompt)
		m.responseCache[cacheKey] = response
	}
}

// Start starts the mock server
func (m *MockOllamaServer) Start() error {
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Mock Ollama server error: %v", err)
		}
	}()
	return nil
}

// Stop stops the mock server
func (m *MockOllamaServer) Stop() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

// handleTags handles the /api/tags endpoint (health check)
func (m *MockOllamaServer) handleTags(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"models": []map[string]interface{}{
			{
				"name":       "gpt-oss:20b",
				"modified_at": "2024-01-01T00:00:00Z",
				"size":       12345678,
			},
			{
				"name":       "llama3.2",
				"modified_at": "2024-01-01T00:00:00Z",
				"size":       12345678,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGenerate handles the /api/generate endpoint
func (m *MockOllamaServer) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	prompt, _ := request["prompt"].(string)

	// Check cache first
	cacheKey := getCacheKey(prompt)
	m.cacheMutex.RLock()
	cachedResponse, found := m.responseCache[cacheKey]
	m.cacheMutex.RUnlock()

	var response string
	if found {
		response = cachedResponse
	} else {
		// Generate and cache the response
		response = m.generateMockResponse(prompt)
		m.cacheMutex.Lock()
		m.responseCache[cacheKey] = response
		m.cacheMutex.Unlock()
	}

	ollamaResponse := map[string]interface{}{
		"model":    request["model"],
		"response": response,
		"done":     true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "keep-alive")
	json.NewEncoder(w).Encode(ollamaResponse)
}

// getCacheKey creates a cache key from a prompt
func getCacheKey(prompt string) string {
	// Use a simple hash-like key based on prompt length and first/last chars
	if len(prompt) == 0 {
		return "empty"
	}
	if len(prompt) < 50 {
		return prompt
	}
	// For long prompts, use a simple key based on length and content type
	key := fmt.Sprintf("%d_%c_%c", len(prompt), prompt[0], prompt[len(prompt)-1])
	if strings.Contains(prompt, "synopsis") {
		key += "_synopsis"
	} else if strings.Contains(prompt, "tags") {
		key += "_tags"
	} else if strings.Contains(prompt, "editorial") {
		key += "_editorial"
	} else if strings.Contains(prompt, "quality") {
		key += "_quality"
	}
	return key
}

// Pre-computed responses for performance
var (
	synopsisResponse = "This is a mock synopsis. The text discusses various topics and presents information in a structured manner. This mock response helps test the integration without requiring a real AI model."
	cleanResponse    = "This is cleaned text content without artifacts or formatting issues."
	editorialResponse = "This text appears to be informational in nature. The writing maintains a neutral tone with balanced presentation. No significant editorial bias is detected in this mock analysis."
	referencesResponse = `[{"text":"Sample statistic or claim","type":"statistic","context":"Surrounding context for the claim","confidence":"medium"}]`
	aiDetectionResponse = `{"likelihood":"unlikely","confidence":"medium","reasoning":"The text shows natural human writing patterns with varied sentence structure and authentic voice.","indicators":["natural flow","varied vocabulary","personal tone"],"human_score":75}`
	qualityScoreResponse = `{"score":0.30,"reason":"The text is well-written, informative, and provides valuable content.","categories":["informative","well_written"],"quality_indicators":["clear_structure","good_grammar","valuable_insights"],"problems_detected":[]}`
)

// generateMockResponse generates a mock response based on the prompt
func (m *MockOllamaServer) generateMockResponse(prompt string) string {
	promptLower := strings.ToLower(prompt)

	// Synopsis generation
	if strings.Contains(promptLower, "synopsis") {
		return synopsisResponse
	}

	// Text cleaning
	if strings.Contains(promptLower, "clean") {
		return cleanResponse
	}

	// Editorial analysis
	if strings.Contains(promptLower, "editorial") || strings.Contains(promptLower, "bias") {
		return editorialResponse
	}

	// Tag generation - analyze the prompt content to generate appropriate tags
	if strings.Contains(promptLower, "tags") && strings.Contains(promptLower, "json array") {
		tags := []string{}

		// Sentiment-based tags
		if strings.Contains(promptLower, "positive") || strings.Contains(promptLower, "happy") {
			tags = append(tags, "positive")
		}
		if strings.Contains(promptLower, "negative") || strings.Contains(promptLower, "sad") {
			tags = append(tags, "negative")
		}

		// Topic-based tags
		if strings.Contains(promptLower, "climate") || strings.Contains(promptLower, "environment") {
			tags = append(tags, "climate", "environment")
		}
		if strings.Contains(promptLower, "technology") || strings.Contains(promptLower, "programming") {
			tags = append(tags, "technology", "programming")
		}
		if strings.Contains(promptLower, "science") {
			tags = append(tags, "science")
		}

		// Default tags if none matched
		if len(tags) == 0 {
			tags = []string{"information", "analysis", "content"}
		}

		// Limit to 5 tags
		if len(tags) > 5 {
			tags = tags[:5]
		}

		// Marshal to JSON
		tagsJSON, _ := json.Marshal(tags)
		return string(tagsJSON)
	}

	// Reference extraction
	if strings.Contains(promptLower, "references") || strings.Contains(promptLower, "factual claims") {
		return referencesResponse
	}

	// AI detection
	if strings.Contains(promptLower, "ai or a human") || strings.Contains(promptLower, "ai-generated") {
		return aiDetectionResponse
	}

	// Link/Content quality scoring (from scraper)
	if (strings.Contains(promptLower, "content quality assessment") && strings.Contains(promptLower, "webpage")) ||
	   (strings.Contains(promptLower, "ingested into a knowledge database") || strings.Contains(promptLower, "knowledge database")) {
		// Determine score based on URL in the prompt
		score := 0.8
		reason := "The webpage contains informative content suitable for knowledge database ingestion."
		categories := []string{"informational", "reference"}

		if strings.Contains(promptLower, "social") || strings.Contains(promptLower, "facebook") ||
		   strings.Contains(promptLower, "twitter") || strings.Contains(promptLower, "instagram") {
			score = 0.2
			reason = "Social media platform detected - not suitable for knowledge database."
			categories = []string{"social_media", "low_quality"}
		}

		return fmt.Sprintf(`{
			"score": %f,
			"reason": "%s",
			"categories": %s,
			"malicious_indicators": []
		}`, score, reason, mustMarshalJSON(categories))
	}

	// Text quality scoring (from textanalyzer)
	if strings.Contains(promptLower, "text and determine its quality") ||
	   (strings.Contains(promptLower, "quality_indicators") && strings.Contains(promptLower, "problems_detected")) {
		return qualityScoreResponse
	}

	// Image analysis
	if strings.Contains(promptLower, "analyze this image") {
		return `{
			"summary": "This is a mock image analysis. The image appears to show various visual elements arranged in a composition. Colors and shapes are present in the frame. This is a test response for image analysis functionality.",
			"tags": ["image", "visual", "content", "test", "mock"]
		}`
	}

	// Content extraction
	if strings.Contains(promptLower, "content extraction") || strings.Contains(promptLower, "meaningful content") {
		return "This is the extracted main content from the page, with navigation and advertisements removed."
	}

	// Default response
	return "This is a mock response from the Ollama test server. The actual response would contain AI-generated content based on the prompt."
}

// formatAddr formats a port number into an address string
func formatAddr(port int) string {
	return fmt.Sprintf(":%d", port)
}

// mustMarshalJSON marshals a value to JSON, panicking on error (for test code only)
func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return string(b)
}
