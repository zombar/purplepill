package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// MockOllamaServer provides a mock Ollama API server for testing
type MockOllamaServer struct {
	server *http.Server
	port   int
}

// NewMockOllamaServer creates a new mock Ollama server
func NewMockOllamaServer(port int) *MockOllamaServer {
	mux := http.NewServeMux()

	mock := &MockOllamaServer{
		port: port,
		server: &http.Server{
			Addr:    formatAddr(port),
			Handler: mux,
		},
	}

	// Register handlers
	mux.HandleFunc("/api/tags", mock.handleTags)
	mux.HandleFunc("/api/generate", mock.handleGenerate)

	return mock
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

	// Generate a simple mock response based on the prompt
	response := m.generateMockResponse(prompt)

	ollamaResponse := map[string]interface{}{
		"model":    request["model"],
		"response": response,
		"done":     true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ollamaResponse)
}

// generateMockResponse generates a mock response based on the prompt
func (m *MockOllamaServer) generateMockResponse(prompt string) string {
	promptLower := strings.ToLower(prompt)

	// Synopsis generation
	if strings.Contains(promptLower, "synopsis") {
		return "This is a mock synopsis. The text discusses various topics and presents information in a structured manner. This mock response helps test the integration without requiring a real AI model."
	}

	// Text cleaning
	if strings.Contains(promptLower, "clean") {
		return "This is cleaned text content without artifacts or formatting issues."
	}

	// Editorial analysis
	if strings.Contains(promptLower, "editorial") || strings.Contains(promptLower, "bias") {
		return "This text appears to be informational in nature. The writing maintains a neutral tone with balanced presentation. No significant editorial bias is detected in this mock analysis."
	}

	// Tag generation
	if strings.Contains(promptLower, "tags") && strings.Contains(promptLower, "json array") {
		// Generate tags based on content in the prompt
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
		return `[
			{
				"text": "Sample statistic or claim",
				"type": "statistic",
				"context": "Surrounding context for the claim",
				"confidence": "medium"
			}
		]`
	}

	// AI detection
	if strings.Contains(promptLower, "ai or a human") || strings.Contains(promptLower, "ai-generated") {
		return `{
			"likelihood": "unlikely",
			"confidence": "medium",
			"reasoning": "The text shows natural human writing patterns with varied sentence structure and authentic voice.",
			"indicators": ["natural flow", "varied vocabulary", "personal tone"],
			"human_score": 75
		}`
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
