package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ServiceConfig holds configuration for a test service
type ServiceConfig struct {
	Name        string
	Port        int
	BinaryPath  string
	Args        []string
	Env         []string
	HealthCheck string
}

// TestServices manages the lifecycle of test services
type TestServices struct {
	t                *testing.T
	processes        map[string]*exec.Cmd
	ollamaUp         bool
	tempDBDir        string
	mockOllama       *MockOllamaServer
	mockOllamaPort   int
	redisContainer   string
	redisPort        int
	postgresContainer string
	postgresPort      int
}

// NewTestServices creates a new test services manager
func NewTestServices(t *testing.T) *TestServices {
	tempDir, err := os.MkdirTemp("", "purpletab-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	ts := &TestServices{
		t:              t,
		processes:      make(map[string]*exec.Cmd),
		tempDBDir:      tempDir,
		mockOllamaPort: 11435, // Use port 11435 to avoid conflict with real Ollama
		redisPort:      16379, // Use port 16379 for test Redis
		postgresPort:   15432, // Use port 15432 for test PostgreSQL
	}

	// Start PostgreSQL container for database tests
	ts.startPostgres()

	// Start Redis container for async queue support
	ts.startRedis()

	// Start mock Ollama server
	ts.startMockOllama()

	return ts
}

// StartService starts a service and waits for it to be healthy
func (ts *TestServices) StartService(config ServiceConfig) error {
	ts.t.Logf("Starting %s on port %d...", config.Name, config.Port)

	cmd := exec.Command(config.BinaryPath, config.Args...)
	cmd.Env = append(os.Environ(), config.Env...)

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", config.Name, err)
	}

	ts.processes[config.Name] = cmd

	// Wait for service to be healthy
	if err := ts.waitForHealth(config.HealthCheck, 30*time.Second); err != nil {
		ts.StopService(config.Name)
		return fmt.Errorf("%s failed health check: %w", config.Name, err)
	}

	ts.t.Logf("%s started successfully", config.Name)
	return nil
}

// StopService stops a running service
func (ts *TestServices) StopService(name string) {
	cmd, exists := ts.processes[name]
	if !exists {
		return
	}

	ts.t.Logf("Stopping %s...", name)

	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	delete(ts.processes, name)
}

// StopAll stops all running services and cleans up
func (ts *TestServices) StopAll() {
	for name := range ts.processes {
		ts.StopService(name)
	}

	// Stop mock Ollama server
	ts.stopMockOllama()

	// Stop Redis container
	ts.stopRedis()

	// Stop PostgreSQL container
	ts.stopPostgres()

	// Clean up temp directory
	if ts.tempDBDir != "" {
		_ = os.RemoveAll(ts.tempDBDir)
	}
}

// waitForHealth polls the health endpoint until it responds or times out
func (ts *TestServices) waitForHealth(healthURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 1 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timeout for %s", healthURL)
		case <-ticker.C:
			resp, err := client.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// CheckOllamaAvailable checks if Ollama is running
// In test mode, this always returns true since we use the mock server
func (ts *TestServices) CheckOllamaAvailable() bool {
	if ts.ollamaUp {
		return true
	}

	// Check if mock Ollama server is running
	if ts.mockOllama != nil {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/tags", ts.mockOllamaPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			ts.ollamaUp = true
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return false
}

// GetDBPath returns a path for a temporary test database
func (ts *TestServices) GetDBPath(serviceName string) string {
	return filepath.Join(ts.tempDBDir, fmt.Sprintf("%s.db", serviceName))
}

// BuildService builds a service binary for testing
func BuildService(t *testing.T, serviceDir, binaryName string) string {
	t.Helper()

	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	servicePath := filepath.Join(projectRoot, serviceDir)
	binaryPath := filepath.Join(servicePath, binaryName)

	t.Logf("Building %s...", binaryName)

	cmd := exec.Command("make", "build")
	cmd.Dir = servicePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build %s: %v", binaryName, err)
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not found after build: %s", binaryPath)
	}

	return binaryPath
}

// GetProjectRoot returns the absolute path to the project root
func GetProjectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	return root
}

// startMockOllama starts the mock Ollama server
func (ts *TestServices) startMockOllama() {
	ts.t.Logf("Starting mock Ollama server on port %d...", ts.mockOllamaPort)

	ts.mockOllama = NewMockOllamaServer(ts.mockOllamaPort)
	if err := ts.mockOllama.Start(); err != nil {
		ts.t.Fatalf("Failed to start mock Ollama server: %v", err)
	}

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Verify it's running
	client := &http.Client{Timeout: 2 * time.Second}
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/tags", ts.mockOllamaPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			ts.t.Log("Mock Ollama server started successfully")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	ts.t.Fatal("Mock Ollama server failed to become ready")
}

// stopMockOllama stops the mock Ollama server
func (ts *TestServices) stopMockOllama() {
	if ts.mockOllama != nil {
		ts.t.Log("Stopping mock Ollama server...")
		if err := ts.mockOllama.Stop(); err != nil {
			ts.t.Logf("Error stopping mock Ollama server: %v", err)
		}
		ts.mockOllama = nil
	}
}

// GetOllamaURL returns the URL for the Ollama server (mock in tests)
func (ts *TestServices) GetOllamaURL() string {
	return fmt.Sprintf("http://localhost:%d", ts.mockOllamaPort)
}

// startRedis starts a Redis container for tests
func (ts *TestServices) startRedis() {
	ts.t.Logf("Starting Redis container on port %d...", ts.redisPort)

	// Check if Redis container already exists and remove it
	_ = exec.Command("docker", "rm", "-f", "docutab-test-redis").Run()

	// Start Redis container
	cmd := exec.Command("docker", "run", "-d",
		"--name", "docutab-test-redis",
		"-p", fmt.Sprintf("%d:6379", ts.redisPort),
		"redis:7-alpine")

	output, err := cmd.CombinedOutput()
	if err != nil {
		ts.t.Fatalf("Failed to start Redis container: %v\nOutput: %s", err, string(output))
	}

	ts.redisContainer = "docutab-test-redis"

	// Wait for Redis to be ready
	time.Sleep(2 * time.Second)

	// Verify Redis is responsive
	client := &http.Client{Timeout: 1 * time.Second}
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("docker", "exec", ts.redisContainer, "redis-cli", "ping")
		if output, err := cmd.CombinedOutput(); err == nil && string(output) == "PONG\n" {
			ts.t.Log("Redis container started successfully")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	ts.t.Log("Warning: Redis may not be fully ready, but continuing anyway...")
	_ = client
}

// stopRedis stops and removes the Redis container
func (ts *TestServices) stopRedis() {
	if ts.redisContainer != "" {
		ts.t.Log("Stopping Redis container...")
		cmd := exec.Command("docker", "rm", "-f", ts.redisContainer)
		if output, err := cmd.CombinedOutput(); err != nil {
			ts.t.Logf("Error stopping Redis container: %v\nOutput: %s", err, string(output))
		}
		ts.redisContainer = ""
	}
}

// GetRedisAddr returns the Redis address for services
func (ts *TestServices) GetRedisAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", ts.redisPort)
}

// startPostgres starts a PostgreSQL container for tests
func (ts *TestServices) startPostgres() {
	ts.t.Logf("Starting PostgreSQL container on port %d...", ts.postgresPort)

	// Check if PostgreSQL container already exists and remove it
	_ = exec.Command("docker", "rm", "-f", "docutab-test-postgres").Run()

	// Start PostgreSQL container
	cmd := exec.Command("docker", "run", "-d",
		"--name", "docutab-test-postgres",
		"-p", fmt.Sprintf("%d:5432", ts.postgresPort),
		"-e", "POSTGRES_USER=docutab_test",
		"-e", "POSTGRES_PASSWORD=test_pass",
		"-e", "POSTGRES_DB=docutab_test",
		"postgres:16-alpine")

	output, err := cmd.CombinedOutput()
	if err != nil {
		ts.t.Fatalf("Failed to start PostgreSQL container: %v\nOutput: %s", err, string(output))
	}

	ts.postgresContainer = "docutab-test-postgres"

	// Wait for PostgreSQL to be ready
	ts.t.Log("Waiting for PostgreSQL to be ready...")
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("docker", "exec", ts.postgresContainer, "pg_isready", "-U", "docutab_test")
		if _, err := cmd.CombinedOutput(); err == nil {
			ts.t.Log("PostgreSQL container started successfully")
			// Give it one more second to be fully ready
			time.Sleep(1 * time.Second)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	ts.t.Fatal("PostgreSQL container failed to become ready")
}

// stopPostgres stops and removes the PostgreSQL container
func (ts *TestServices) stopPostgres() {
	if ts.postgresContainer != "" {
		ts.t.Log("Stopping PostgreSQL container...")
		cmd := exec.Command("docker", "rm", "-f", ts.postgresContainer)
		if output, err := cmd.CombinedOutput(); err != nil {
			ts.t.Logf("Error stopping PostgreSQL container: %v\nOutput: %s", err, string(output))
		}
		ts.postgresContainer = ""
	}
}

// GetPostgresConfig returns PostgreSQL connection details for services
func (ts *TestServices) GetPostgresConfig() (host string, port int, user, password, database string) {
	return "127.0.0.1", ts.postgresPort, "docutab_test", "test_pass", "docutab_test"
}
