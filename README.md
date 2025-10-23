# PurpleTab

[![CI](https://github.com/zombar/purpletab/actions/workflows/ci.yml/badge.svg)](https://github.com/zombar/purpletab/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/zombar/purpletab)](https://github.com/zombar/purpletab/issues)
[![GitHub stars](https://img.shields.io/github/stars/zombar/purpletab?style=social)](https://github.com/zombar/purpletab/stargazers)

A microservices-based web content processing platform built in Go. The system scrapes web pages, extracts content using AI, and performs comprehensive text analysis.

## Architecture

PurpleTab consists of five services that work together:

- **Scraper** - Fetches web pages and extracts content, images, and metadata using Ollama AI models. Stores files in filesystem with SEO-friendly slugs
- **TextAnalyzer** - Performs text analysis including sentiment analysis, readability scoring, named entity recognition, and AI-powered content detection
- **Controller** - Orchestrates the scraper and text analyzer services, providing a unified API, asynchronous scrape request tracking, tag-based search, and SEO-optimized content serving
- **Scheduler** - Manages scheduled tasks for automated scraping and database maintenance using cron expressions
- **Web** - React-based web interface for content ingestion with real-time progress tracking, search, and viewing

### Two-Audience Architecture

PurpleTab serves **two distinct audiences** with different interfaces:

**1. Internal Users (Administrators)**
- Use the React Web App (Port 3000)
- Ingest content, manage scrapes, search and filter
- Client-side SPA - no SEO needed
- Private admin interface

**2. Public Users & Search Engines**
- Access Controller's SEO endpoints (Port 9080)
- Server-rendered HTML at `/content/{slug}`
- XML sitemaps at `/sitemap.xml` and `/images-sitemap.xml`
- Discoverable by Google and other search engines
- Public knowledge base

```
Internal Users:
┌──────────────┐
│   Web App    │ (React SPA - Admin Interface)
│  Port 3000   │
└──────┬───────┘
       │ API calls
       v
┌──────────────┐       ┌──────────┐
│  Controller  │──────>│ Scraper  │
│  Port 9080   │       │Port 9081 │
└──────┬───────┘       └──────────┘
       │
       v              ┌──────────────┐
       └─────────────>│TextAnalyzer  │
                      │  Port 9082   │
                      └──────────────┘

Public Users & Search Engines:
┌──────────────┐
│Search Engines│ (Google, Bing, etc.)
└──────┬───────┘
       │ GET /content/{slug}
       │ GET /sitemap.xml
       v
┌──────────────┐
│  Controller  │ (SEO-optimized HTML)
│  Port 9080   │
└──────────────┘
```

**Why This Architecture?**
- Clean separation of concerns (API vs. content serving)
- Fast SEO pages with direct database access
- Simple deployment (no SSR complexity)
- Follows microservices principles
- Both audiences get optimized experiences

## Features

### Core Capabilities
- **AI-Powered Scraping**: Extracts and cleans web content using Ollama AI models
- **Comprehensive Text Analysis**: Sentiment analysis, readability scoring, named entity recognition
- **Smart Link Scoring**: AI-based quality assessment for scraped URLs
- **Tag-Based Search**: Fuzzy search across extracted content and metadata
- **Asynchronous Processing**: Background scraping with real-time progress tracking

### SEO and Public Content
- **SEO-Friendly URLs**: Automatic slug generation from content titles for clean, readable URLs
- **Structured Data**: JSON-LD schema.org markup for rich search engine results
- **Meta Tags**: Complete Open Graph and Twitter Card metadata for social sharing
- **XML Sitemaps**: Automatically generated sitemaps for search engine indexing
- **Robots.txt**: Configurable crawler directives
- **Filesystem Storage**: Efficient file-based storage for images and content with organized directory structure

The Controller service provides public-facing endpoints for serving SEO-optimized HTML pages of scraped content, making the platform suitable for building searchable knowledge bases and content repositories.

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
- Controller: http://localhost:9080
- Scraper: http://localhost:9081
- TextAnalyzer: http://localhost:9082
- Scheduler: http://localhost:9083

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

**Web UI with hot reload:**

```bash
# Run backend services in Docker
docker-compose up -d controller scraper textanalyzer

# Run web UI locally with hot reload
cd apps/web
npm install
npm run dev
# Access at http://localhost:5173
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
- `STORAGE_BASE_PATH` - Base path for filesystem storage (default: ./storage)
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name

**TextAnalyzer:**
- `PORT` - HTTP server port
- `DB_PATH` - SQLite database path
- `USE_OLLAMA` - Enable/disable Ollama integration (true/false)
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name

**Web:**
- `CONTROLLER_API_URL` - Controller API URL (default: http://localhost:9080)

## Service Documentation

Detailed documentation for each service:

- [Controller](apps/controller/README.md) - Orchestration service with unified API
- [Scraper](apps/scraper/README.md) - Web scraping with AI content extraction
- [TextAnalyzer](apps/textanalyzer/README.md) - Comprehensive text analysis
- [Scheduler](apps/scheduler/README.md) - Scheduled task management
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
make scheduler-build    # Build scheduler only
make web-build          # Build web interface

# Test commands
make test               # Run all tests
make test-coverage      # Generate coverage reports
make controller-test    # Test controller only
make scraper-test       # Test scraper only
make textanalyzer-test  # Test textanalyzer only
make scheduler-test     # Test scheduler only
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
purpletab/
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
│   ├── scheduler/        # Scheduled task service
│   │   ├── cmd/          # Application entry point
│   │   ├── models/       # Data models
│   │   ├── db/           # Database layer
│   │   ├── api/          # API server
│   │   └── README.md     # Service documentation
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

### Continuous Integration

GitHub Actions workflows automatically run tests on every commit:

- **CI Workflow** - Detects changed services and runs only affected unit tests
- **Integration Tests Workflow** - Runs full end-to-end tests with all services
- **Per-Service Workflows** - Individual workflows for each service with coverage reporting
- Workflows trigger on push/PR to main/master/develop branches
- Matrix testing: Go 1.21, Node.js 18-20

Workflows are located in `.github/workflows/`:
- `ci.yml` - Smart monorepo CI that runs unit tests for changed apps
- `integration-tests.yml` - End-to-end integration tests with Docker Compose
- `test-controller.yml` - Controller tests (Go)
- `test-scraper.yml` - Scraper tests (Go)
- `test-textanalyzer.yml` - TextAnalyzer tests (Go)
- `test-web.yml` - Web interface tests (Node.js/React)

**Test Strategy:**
- **Unit tests** run for each changed service independently (fast feedback)
- **Integration tests** run in a separate workflow that:
  - Builds all services with Docker
  - Starts services with Docker Compose
  - Runs comprehensive end-to-end tests
  - Includes health checks and proper service orchestration
  - Only runs benchmarks on main branch

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

## Project Stats

### Code Metrics

![GitHub code size](https://img.shields.io/github/languages/code-size/zombar/purpletab)
![Lines of code](https://img.shields.io/tokei/lines/github/zombar/purpletab)
![GitHub last commit](https://img.shields.io/github/last-commit/zombar/purpletab)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/zombar/purpletab)

### Service Test Coverage

All services include comprehensive test suites:

| Service | Unit Tests | E2E/Integration Tests | Test Types |
|---------|-----------|----------------------|------------|
| Controller | ✓ Go tests | ✓ Integration suite | API handlers, storage, orchestration |
| Scraper | ✓ Go tests | ✓ Integration suite | Content extraction, AI features, storage |
| TextAnalyzer | ✓ Go tests | ✓ Integration suite | Analysis algorithms, AI detection, scoring |
| Scheduler | ✓ Go tests | ✓ Integration suite | Cron scheduling, task management |
| Web | ✓ Vitest (unit) | ✓ Playwright (195 E2E) | Components, hooks, integration |
| **Integration** | - | ✓ Full stack tests | Service orchestration, benchmarks |

Run `make test` for unit tests, `make test-integration` for integration tests, or `make test-all` for complete test suite.

### Language Breakdown

![Top Languages](https://github-readme-stats.vercel.app/api/top-langs/?username=zombar&repo=purpletab&layout=compact&theme=dark)

### Repository Activity

![GitHub Activity Graph](https://github-readme-activity-graph.vercel.app/graph?username=zombar&repo=purpletab&theme=github-compact)

### Contributors

[![Contributors](https://contrib.rocks/image?repo=zombar/purpletab)](https://github.com/zombar/purpletab/graphs/contributors)

## Production Considerations

- **Database**: Migrate to PostgreSQL for multi-instance deployments
- **Service Discovery**: Use environment variables or service mesh
- **Monitoring**: Add structured logging and metrics collection
- **Security**: Implement authentication, rate limiting, and HTTPS
- **Scaling**: Deploy services independently based on load

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
