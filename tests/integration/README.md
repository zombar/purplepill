# Integration Tests

Comprehensive integration tests for the PurpleTab microservices platform.

## Overview

These integration tests verify the interactions between the Controller, Scraper, and TextAnalyzer services. They test the full request lifecycle, data flow between services, and system behavior under various conditions.

## Features

- **Full Stack Testing** - Tests complete workflows from Controller through to downstream services
- **Ollama Awareness** - Automatically detects Ollama availability and tests both enhanced and degraded modes
- **Metadata Validation** - Verifies response structure and field presence (not content quality)
- **Performance Benchmarking** - Optional load testing to measure system performance under stress
- **Isolated Environments** - Each test run uses temporary databases and test ports

## Test Coverage

### Functional Tests

1. **Direct Text Analysis** (`/analyze` endpoint)
   - Controller → TextAnalyzer flow
   - Metadata structure validation
   - Tag generation and storage
   - With and without Ollama

2. **URL Scraping & Analysis** (`/scrape` endpoint)
   - Controller → Scraper → TextAnalyzer flow
   - Dual UUID tracking (scraper + analyzer)
   - Combined metadata from both services
   - Content extraction validation

3. **Tag Search** (`/search/tags` endpoint)
   - Exact tag matching
   - Fuzzy search capabilities
   - Result accuracy

4. **Request Retrieval**
   - Individual request retrieval by ID
   - Paginated list queries
   - Data persistence validation

### Benchmark Tests

Optional performance tests that measure:
- Request throughput (requests/second)
- Response time statistics (avg, min, max)
- Success/failure rates
- System behavior under concurrent load
- Mixed workload handling

## Prerequisites

- Go 1.21 or higher
- GCC (for SQLite compilation)
- All three services built (`make build`)
- Ollama (optional, for enhanced testing)

## Running Tests

### Quick Start

```bash
# Run all integration tests (from project root)
make test-integration

# Run without benchmarks (faster)
make test-integration-short

# Run unit tests + integration tests
make test-all
```

### Direct Go Commands

```bash
# From the tests/integration directory
cd tests/integration

# Run all tests
go test -v -timeout 10m

# Run in short mode (skip benchmarks)
go test -v -short -timeout 5m

# Run specific test
go test -v -run TestControllerIntegration/DirectTextAnalysis
```

### Benchmark Tests

Benchmarks are disabled by default to prevent accidental load testing. To run:

```bash
# From project root
make test-benchmark

# Or directly with Go
cd tests/integration
BENCHMARK=true go test -v -timeout 15m -run TestBenchmark
```

## Test Ports

Integration tests use different ports to avoid conflicts with development servers:

| Service      | Test Port | Production Port |
|--------------|-----------|-----------------|
| Controller   | 18080     | 8080            |
| Scraper      | 18081     | 8081            |
| TextAnalyzer | 18082     | 8082            |

## Ollama Behavior

Tests automatically detect Ollama availability:

**When Ollama is available:**
- ✓ Tests verify AI-enhanced metadata fields
- ✓ Logs indicate AI features are being tested
- Synopsis, cleaned_text, ai_detection, etc. are checked

**When Ollama is unavailable:**
- ✓ Tests verify graceful degradation
- ✓ Core metadata fields still present
- ✓ No AI-specific fields expected
- ✓ System remains functional

## Test Structure

```
tests/integration/
├── README.md              # This file
├── go.mod                 # Go module definition
├── helpers.go             # Service lifecycle management utilities
├── controller_test.go     # Main integration tests
└── benchmark_test.go      # Performance/load tests
```

### Key Components

**helpers.go:**
- `TestServices` - Manages service lifecycle (start/stop)
- `ServiceConfig` - Service configuration
- `BuildService()` - Builds service binaries
- Health check utilities
- Ollama availability detection

**controller_test.go:**
- End-to-end workflow tests
- Metadata structure validation
- Request/response verification
- Tag search and retrieval tests

**benchmark_test.go:**
- Load testing framework
- Performance metrics collection
- Concurrent request handling
- Mixed workload simulation

## What Tests Verify

### ✓ Tests DO verify:
- Response structure and field presence
- Correct data flow between services
- UUID tracking and correlation
- Tag generation and searchability
- Metadata field types and structure
- Service orchestration
- Error handling and graceful degradation

### ✗ Tests DO NOT verify:
- Content quality (handled by unit tests)
- AI model accuracy (not deterministic)
- Specific sentiment scores
- Exact tag values
- Text cleaning quality

The focus is on **integration correctness**, not **content quality**.

## Benchmark Metrics

When running benchmarks, the following metrics are reported:

- **Total Requests** - Number of requests sent
- **Successful Requests** - Requests that returned 200 OK
- **Failed Requests** - Requests that failed or returned errors
- **Total Duration** - Time to complete all requests
- **Average Response Time** - Mean response time per request
- **Min/Max Response Time** - Response time bounds
- **Requests/Second** - Throughput measurement

### Success Criteria

Benchmarks fail if:
- Success rate < 95%
- Other custom thresholds (can be configured)

## Troubleshooting

### Tests fail to start services

**Problem:** Services don't start or fail health checks

**Solutions:**
- Ensure all services build successfully: `make build`
- Check ports 18080-18082 are available
- Verify SQLite/GCC is installed
- Check service build artifacts exist

### Tests timeout

**Problem:** Tests exceed timeout limits

**Solutions:**
- Increase timeout: `go test -timeout 15m`
- Check Ollama performance (AI analysis can be slow)
- Verify network connectivity for URL scraping
- Run in short mode: `go test -short`

### Ollama tests fail

**Problem:** Tests expect Ollama but it's not available

**Solutions:**
- Tests should auto-detect and adapt
- Verify Ollama is running: `curl http://localhost:11434/api/tags`
- Check test logs for "Ollama is available" message
- If intended, tests will run in degraded mode

### Database conflicts

**Problem:** Database locks or conflicts

**Solutions:**
- Each test run uses temporary databases
- Old temp databases should auto-clean
- Manually clean `/tmp/purpletab-integration-*` if needed

## Extending Tests

### Adding New Test Cases

1. Add test function to `controller_test.go`:
   ```go
   func testNewFeature(t *testing.T) {
       // Your test logic
   }
   ```

2. Register in main test:
   ```go
   t.Run("NewFeature", func(t *testing.T) {
       testNewFeature(t)
   })
   ```

### Adding Benchmarks

1. Add benchmark function to `benchmark_test.go`:
   ```go
   func benchmarkNewScenario(t *testing.T, requests, concurrency int) {
       // Your benchmark logic
   }
   ```

2. Call from `TestBenchmarkControllerLoad`:
   ```go
   t.Run("BenchmarkNewScenario", func(t *testing.T) {
       benchmarkNewScenario(t, 100, 10)
   })
   ```

## CI/CD Integration

Recommended CI pipeline:

```yaml
# Example GitHub Actions
- name: Run Unit Tests
  run: make test

- name: Run Integration Tests
  run: make test-integration-short

- name: Run Benchmarks (optional, on schedule)
  run: make test-benchmark
  if: github.event_name == 'schedule'
```

## Performance Baselines

Expected performance (with Ollama on consumer hardware):

| Operation        | Avg Time | Throughput   |
|------------------|----------|--------------|
| Direct Analysis  | 500ms    | ~20 req/s    |
| URL Scraping     | 3-10s    | ~5 req/s     |
| Tag Search       | <50ms    | ~200 req/s   |
| Request Retrieval| <20ms    | ~500 req/s   |

*Actual performance varies based on hardware, Ollama model, and network conditions.*

## Best Practices

1. **Run short mode during development** - Faster feedback
2. **Run full tests before commits** - Ensure nothing breaks
3. **Run benchmarks periodically** - Track performance trends
4. **Check Ollama availability** - Understand test environment
5. **Monitor test logs** - Rich diagnostic information
6. **Use temporary databases** - Avoid state pollution

## Contributing

When adding integration tests:

- Focus on integration points, not implementation details
- Verify structure and presence, not exact values
- Handle both Ollama and non-Ollama scenarios
- Add clear, descriptive test names
- Include logging for debugging
- Update this README if adding new features

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.
