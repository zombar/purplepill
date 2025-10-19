# PurpleTab

A microservices-based web content processing platform built in Go. The system scrapes web pages, extracts content using AI, and performs comprehensive text analysis.

## Architecture

PurpleTab consists of four services that work together:

- **Scraper** - Fetches web pages and extracts content, images, and metadata using Ollama AI models
- **TextAnalyzer** - Performs text analysis including sentiment analysis, readability scoring, named entity recognition, and AI-powered content detection
- **Controller** - Orchestrates the scraper and text analyzer services, providing a unified API, asynchronous scrape request tracking, and tag-based search
- **Web** - React-based web interface for content ingestion with real-time progress tracking, search, and viewing

```
┌──────────┐
│   Web    │
│ Port 3000│
└────┬─────┘
     │
     v
┌────────────┐       ┌──────────┐
│ Controller │──────>│ Scraper  │
│  Port 8080 │       │Port 8081 │
└────┬───────┘       └──────────┘
     │
     v              ┌──────────────┐
     └─────────────>│TextAnalyzer  │
                    │  Port 8082   │
                    └──────────────┘
```

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for containerized deployment)
- [Ollama](https://ollama.ai) (for AI-powered features)
- GCC (required for SQLite CGO compilation)

### Ollama Setup

Each service can use Ollama for AI-powered content processing:

```bash
# Install Ollama from https://ollama.ai

# Pull required models
ollama pull llama3.2           # For text generation
ollama pull llama3.2-vision    # For image analysis
ollama pull gpt-oss:20b        # Alternative model
```

## Quick Start

### Docker Deployment (Recommended)

```bash
# Build all services
make docker-build

# Start all services
make docker-up

# View logs
make docker-logs

# Stop all services
make docker-down
```

The services will be available at:
- Web Interface: http://localhost:3000
- Controller: http://localhost:8080
- Scraper: http://localhost:8081
- TextAnalyzer: http://localhost:8082

### Local Development

```bash
# Install dependencies for all services
make install

# Build all services
make build

# Run tests
make test

# Run each service in separate terminals:
make textanalyzer-run    # Terminal 1
make scraper-run-api     # Terminal 2
make controller-run      # Terminal 3
make web-dev             # Terminal 4 (optional - for UI)
```

## Configuration

### Docker Compose

Service configuration is managed through `docker-compose.yml`. Key environment variables:

**Controller:**
- `SCRAPER_BASE_URL` - Scraper service URL (default: http://scraper:8080)
- `TEXTANALYZER_BASE_URL` - TextAnalyzer service URL (default: http://textanalyzer:8080)
- `DATABASE_PATH` - SQLite database path

**Scraper:**
- `PORT` - HTTP server port
- `DB_PATH` - SQLite database path
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name

**TextAnalyzer:**
- `PORT` - HTTP server port
- `DB_PATH` - SQLite database path
- `USE_OLLAMA` - Enable/disable Ollama integration (true/false)
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name

**Web:**
- `CONTROLLER_API_URL` - Controller API URL (default: http://localhost:8080)

## Service Documentation

Detailed documentation for each service:

- [Controller](apps/controller/README.md) - Orchestration service with unified API
- [Scraper](apps/scraper/README.md) - Web scraping with AI content extraction
- [TextAnalyzer](apps/textanalyzer/README.md) - Comprehensive text analysis
- [Web](apps/web/README.md) - React-based web interface

API reference documentation:

- [Controller API](apps/controller/API.md)
- [Scraper API](apps/scraper/API.md)
- [TextAnalyzer API](apps/textanalyzer/API.md)

## Development

### Available Make Commands

```bash
# Build commands
make build              # Build all services
make controller-build   # Build controller only
make scraper-build      # Build scraper only
make textanalyzer-build # Build textanalyzer only
make web-build          # Build web interface

# Test commands
make test               # Run all tests
make test-coverage      # Generate coverage reports
make controller-test    # Test controller only
make scraper-test       # Test scraper only
make textanalyzer-test  # Test textanalyzer only
make web-test           # Test web interface
make web-test-coverage  # Test web with coverage
make web-lint           # Lint web interface

# Code quality
make fmt                # Format all code
make lint               # Lint all code
make check              # Run fmt, lint, and test

# Docker commands
make docker-build       # Build all Docker images
make docker-up          # Start all services
make docker-down        # Stop all services
make docker-logs        # View service logs
make docker-ps          # Show running containers
make docker-clean       # Remove all containers, volumes, images
make docker-restart     # Restart all services

# Utility commands
make clean              # Clean build artifacts
make help               # Show all available commands
```

### Project Structure

```
purplepill/
├── apps/
│   ├── controller/       # Orchestration service
│   │   ├── cmd/          # Application entry point
│   │   ├── internal/     # Internal packages
│   │   ├── README.md     # Service documentation
│   │   └── API.md        # API reference
│   ├── scraper/          # Web scraping service
│   │   ├── cmd/          # CLI and API entry points
│   │   ├── models/       # Data models
│   │   ├── ollama/       # Ollama client
│   │   ├── scraper/      # Core scraping logic
│   │   ├── db/           # Database layer
│   │   ├── api/          # API server
│   │   ├── README.md     # Service documentation
│   │   └── API.md        # API reference
│   ├── textanalyzer/     # Text analysis service
│   │   ├── cmd/          # Application entry point
│   │   ├── internal/     # Internal packages
│   │   ├── README.md     # Service documentation
│   │   └── API.md        # API reference
│   └── web/              # Web interface
│       ├── src/          # React source code
│       ├── public/       # Static assets
│       └── README.md     # Service documentation
├── docker-compose.yml    # Docker orchestration
├── Makefile              # Build automation
└── README.md             # This file
```

### Testing

Each service includes comprehensive test coverage:

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run service-specific tests
go test ./apps/controller/...
go test ./apps/scraper/...
go test ./apps/textanalyzer/...

# Run integration tests
make test-integration

# Run integration tests (skip benchmarks)
make test-integration-short

# Run performance benchmarks
make test-benchmark

# Run all tests (unit + integration)
make test-all
```

#### Integration Tests

PurplePill includes comprehensive integration tests that verify the interactions between all services:

- Tests full Controller → Scraper → TextAnalyzer workflows
- Validates metadata structure and service orchestration
- Automatically adapts to Ollama availability
- Optional performance benchmarking

See [tests/integration/README.md](tests/integration/README.md) for detailed documentation.

### Database

All services use SQLite by default with migration systems for schema management. The codebase is designed for easy PostgreSQL migration for production deployments.

To migrate to PostgreSQL:
1. Update database connection strings in service configurations
2. Modify SQL migration syntax (AUTOINCREMENT → SERIAL, DATETIME → TIMESTAMP)
3. Add PostgreSQL driver dependencies

## Production Considerations

- **Database**: Migrate to PostgreSQL for multi-instance deployments
- **Service Discovery**: Use environment variables or service mesh
- **Monitoring**: Add structured logging and metrics collection
- **Security**: Implement authentication, rate limiting, and HTTPS
- **Scaling**: Deploy services independently based on load

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
