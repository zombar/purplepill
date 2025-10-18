.PHONY: help build test clean install fmt lint docker-build docker-up docker-down docker-logs \
        controller-% scraper-% textanalyzer-% web-%

# Submodule directories
CONTROLLER_DIR=apps/controller
SCRAPER_DIR=apps/scraper
TEXTANALYZER_DIR=apps/textanalyzer
WEB_DIR=apps/web

# Default target
help: ## Display this help message
	@echo "PurplePill - Available targets:"
	@echo ""
	@echo "Aggregate commands (run across all submodules):"
	@echo "  build              - Build all services"
	@echo "  test               - Run tests for all services"
	@echo "  test-coverage      - Run tests with coverage for all services"
	@echo "  clean              - Clean build artifacts for all services"
	@echo "  install            - Install dependencies for all services"
	@echo "  fmt                - Format code for all services"
	@echo "  lint               - Lint code for all services"
	@echo ""
	@echo "Integration test commands:"
	@echo "  test-integration       - Run integration tests"
	@echo "  test-integration-short - Run integration tests (skip benchmarks)"
	@echo "  test-benchmark         - Run performance/load benchmarks"
	@echo "  test-all               - Run all tests (unit + integration)"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build       - Build all Docker images"
	@echo "  docker-up          - Start all services with docker-compose"
	@echo "  docker-down        - Stop all services"
	@echo "  docker-logs        - View logs from all services"
	@echo "  docker-ps          - Show running containers"
	@echo "  docker-clean       - Remove all containers, volumes, and images"
	@echo ""
	@echo "Per-service commands (use <service>-<command>):"
	@echo "  controller-build   - Build controller service"
	@echo "  controller-test    - Test controller service"
	@echo "  controller-run     - Run controller service"
	@echo "  scraper-build      - Build scraper service"
	@echo "  scraper-test       - Test scraper service"
	@echo "  scraper-run-api    - Run scraper API service"
	@echo "  textanalyzer-build - Build textanalyzer service"
	@echo "  textanalyzer-test  - Test textanalyzer service"
	@echo "  textanalyzer-run   - Run textanalyzer service"
	@echo "  web-build          - Build web interface"
	@echo "  web-test           - Test web interface"
	@echo "  web-test-coverage  - Test web interface with coverage"
	@echo "  web-lint           - Lint web interface"
	@echo "  web-dev            - Run web dev server"
	@echo "  web-install        - Install web dependencies"
	@echo ""
	@echo "Utility commands:"
	@echo "  submodule-update   - Update all submodules to latest"
	@echo "  submodule-status   - Show status of all submodules"

# ==================== Aggregate Commands ====================

build: ## Build all services
	@echo "Building all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) build
	@$(MAKE) -C $(SCRAPER_DIR) build-api
	@$(MAKE) -C $(TEXTANALYZER_DIR) build
	@cd $(WEB_DIR) && npm run build
	@echo "All services built successfully!"

test: ## Run tests for all services
	@echo "Running tests for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) test
	@$(MAKE) -C $(SCRAPER_DIR) test
	@$(MAKE) -C $(TEXTANALYZER_DIR) test
	@cd $(WEB_DIR) && npm test -- --run
	@echo "All tests completed!"

test-coverage: ## Run tests with coverage for all services
	@echo "Running tests with coverage for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) test-coverage
	@$(MAKE) -C $(SCRAPER_DIR) test-coverage
	@$(MAKE) -C $(TEXTANALYZER_DIR) test-coverage
	@cd $(WEB_DIR) && npm run test:coverage -- --run
	@echo "All coverage reports generated!"

clean: ## Clean build artifacts for all services
	@echo "Cleaning all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) clean
	@$(MAKE) -C $(SCRAPER_DIR) clean
	@$(MAKE) -C $(TEXTANALYZER_DIR) clean
	@cd $(WEB_DIR) && rm -rf dist
	@echo "All services cleaned!"

install: ## Install dependencies for all services
	@echo "Installing dependencies for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) install
	@$(MAKE) -C $(SCRAPER_DIR) deps
	@$(MAKE) -C $(TEXTANALYZER_DIR) install
	@cd $(WEB_DIR) && npm install
	@echo "All dependencies installed!"

fmt: ## Format code for all services
	@echo "Formatting code for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) fmt
	@$(MAKE) -C $(SCRAPER_DIR) fmt
	@$(MAKE) -C $(TEXTANALYZER_DIR) fmt
	@echo "All code formatted!"

lint: ## Lint code for all services
	@echo "Linting code for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) lint
	@$(MAKE) -C $(SCRAPER_DIR) vet
	@$(MAKE) -C $(TEXTANALYZER_DIR) lint
	@cd $(WEB_DIR) && npm run lint
	@echo "All linting completed!"

# ==================== Docker Commands ====================

docker-build: ## Build all Docker images
	@echo "Building all Docker images..."
	@docker-compose build
	@echo "All Docker images built!"

docker-up: ## Start all services with docker-compose
	@echo "Starting all services..."
	@docker-compose up -d
	@echo "All services started!"
	@echo "Web Interface:  http://localhost:3000"
	@echo "Controller:     http://localhost:8080"
	@echo "Scraper API:    http://localhost:8081"
	@echo "Text Analyzer:  http://localhost:8082"

docker-down: ## Stop all services
	@echo "Stopping all services..."
	@docker-compose down
	@echo "All services stopped!"

docker-logs: ## View logs from all services
	@docker-compose logs -f

docker-ps: ## Show running containers
	@docker-compose ps

docker-clean: ## Remove all containers, volumes, and images
	@echo "Cleaning Docker resources..."
	@docker-compose down -v --rmi all
	@echo "Docker cleanup complete!"

docker-restart: ## Restart all services
	@$(MAKE) docker-down
	@$(MAKE) docker-up

# ==================== Per-Service Commands ====================

# Controller commands
controller-build:
	@$(MAKE) -C $(CONTROLLER_DIR) build

controller-test:
	@$(MAKE) -C $(CONTROLLER_DIR) test

controller-clean:
	@$(MAKE) -C $(CONTROLLER_DIR) clean

controller-run:
	@$(MAKE) -C $(CONTROLLER_DIR) run

controller-dev:
	@$(MAKE) -C $(CONTROLLER_DIR) dev

# Scraper commands
scraper-build:
	@$(MAKE) -C $(SCRAPER_DIR) build-api

scraper-build-cli:
	@$(MAKE) -C $(SCRAPER_DIR) build-cli

scraper-test:
	@$(MAKE) -C $(SCRAPER_DIR) test

scraper-clean:
	@$(MAKE) -C $(SCRAPER_DIR) clean

scraper-run-api:
	@$(MAKE) -C $(SCRAPER_DIR) run-api

# TextAnalyzer commands
textanalyzer-build:
	@$(MAKE) -C $(TEXTANALYZER_DIR) build

textanalyzer-test:
	@$(MAKE) -C $(TEXTANALYZER_DIR) test

textanalyzer-clean:
	@$(MAKE) -C $(TEXTANALYZER_DIR) clean

textanalyzer-run:
	@$(MAKE) -C $(TEXTANALYZER_DIR) run

# Web commands
web-build:
	@cd $(WEB_DIR) && npm run build

web-test:
	@cd $(WEB_DIR) && npm test -- --run

web-test-coverage:
	@cd $(WEB_DIR) && npm run test:coverage -- --run

web-lint:
	@cd $(WEB_DIR) && npm run lint

web-install:
	@cd $(WEB_DIR) && npm install

web-dev:
	@cd $(WEB_DIR) && npm run dev

web-preview:
	@cd $(WEB_DIR) && npm run preview

web-clean:
	@cd $(WEB_DIR) && rm -rf dist

# ==================== Utility Commands ====================

submodule-update: ## Update all submodules to latest
	@echo "Updating submodules..."
	@git submodule update --remote --merge
	@echo "Submodules updated!"

submodule-status: ## Show status of all submodules
	@echo "Submodule status:"
	@git submodule status

submodule-init: ## Initialize submodules (first time setup)
	@echo "Initializing submodules..."
	@git submodule update --init --recursive
	@echo "Submodules initialized!"

# Check target - run all quality checks
check: fmt lint test ## Run all checks (fmt, lint, test)
	@echo "All checks passed!"

# ==================== Integration Tests ====================

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@cd tests/integration && go test -v -timeout 10m
	@echo "Integration tests completed!"

test-integration-short: ## Run integration tests without benchmarks
	@echo "Running integration tests (short mode)..."
	@cd tests/integration && go test -v -short -timeout 5m
	@echo "Integration tests completed!"

test-benchmark: ## Run load/performance benchmarks (requires BENCHMARK=true)
	@echo "Running benchmark tests..."
	@echo "This may take several minutes..."
	@cd tests/integration && BENCHMARK=true go test -v -timeout 15m -run TestBenchmark
	@echo "Benchmark tests completed!"

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "All tests completed!"

# Development workflow - build and start all services locally
dev: build ## Build and run all services locally (non-Docker)
	@echo "Starting development environment..."
	@echo "Note: Run each service in separate terminals:"
	@echo "  Terminal 1: make textanalyzer-run"
	@echo "  Terminal 2: make scraper-run-api"
	@echo "  Terminal 3: make controller-run"
	@echo "  Terminal 4: make web-dev (optional - for UI)"
