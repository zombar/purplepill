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
	t          *testing.T
	processes  map[string]*exec.Cmd
	ollamaUp   bool
	tempDBDir  string
}

// NewTestServices creates a new test services manager
func NewTestServices(t *testing.T) *TestServices {
	tempDir, err := os.MkdirTemp("", "purplepill-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	return &TestServices{
		t:         t,
		processes: make(map[string]*exec.Cmd),
		tempDBDir: tempDir,
	}
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
func (ts *TestServices) CheckOllamaAvailable() bool {
	if ts.ollamaUp {
		return true
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		ts.ollamaUp = true
		return true
	}
	if resp != nil {
		resp.Body.Close()
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
