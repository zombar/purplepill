.PHONY: help build test clean install fmt lint docker-build docker-up docker-down docker-logs \
        docker-staging-% controller-% scraper-% textanalyzer-% web-% scheduler-%

# Submodule directories
CONTROLLER_DIR=apps/controller
SCRAPER_DIR=apps/scraper
TEXTANALYZER_DIR=apps/textanalyzer
WEB_DIR=apps/web
SCHEDULER_DIR=apps/scheduler

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
	@echo "  docker-build           - Build all Docker images"
	@echo "  docker-up              - Start all services with docker compose"
	@echo "  docker-down            - Stop all services"
	@echo "  docker-logs            - View logs from all services"
	@echo "  docker-ps              - Show running containers"
	@echo "  docker-clean           - Remove all containers, volumes, and images"
	@echo "  docker-restart         - Restart all services"
	@echo "  docker-rebuild         - Rebuild and restart all services"
	@echo "  docker-health          - Check health of all services"
	@echo ""
	@echo "Docker staging commands (dev machine):"
	@echo "  docker-staging-build   - Build all service images for staging"
	@echo "  docker-staging-push    - Build and push all images to ghcr.io/zombar"
	@echo "  docker-staging-deploy  - Full local deploy: build and start services"
	@echo ""
	@echo "Docker staging commands (server):"
	@echo "  docker-staging-pull    - Pull latest images and start services"
	@echo "  docker-staging-up      - Start services (without pulling)"
	@echo "  docker-staging-down    - Stop all staging services"
	@echo "  docker-staging-logs    - View logs from staging services"
	@echo ""
	@echo "Docker service commands (per service):"
	@echo "  docker-logs-<service>  - View logs for specific service (e.g., docker-logs-controller)"
	@echo "  docker-restart-<service> - Restart specific service"
	@echo "  docker-shell-<service> - Open shell in specific service"
	@echo "  docker-exec-<service>  - Execute command in service (use CMD='...')"
	@echo ""
	@echo "Docker management:"
	@echo "  docker-volumes         - List all volumes"
	@echo "  docker-volumes-inspect - Inspect volume usage"
	@echo "  docker-volumes-clean   - Remove unused volumes"
	@echo "  docker-images          - List all images"
	@echo "  docker-prune           - Remove unused containers and images"
	@echo "  docker-prune-all       - Remove ALL unused resources (destructive)"
	@echo "  docker-stats           - Show resource usage of containers"
	@echo ""
	@echo "Docker database access:"
	@echo "  docker-db-<service>    - Access SQLite database for service"
	@echo "  docker-backup          - Backup all databases and storage"
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
	@echo "  scheduler-build    - Build scheduler service"
	@echo "  scheduler-test     - Test scheduler service"
	@echo "  scheduler-run      - Run scheduler service"
	@echo "  web-build          - Build web interface"
	@echo "  web-test           - Test web interface"
	@echo "  web-test-coverage  - Test web interface with coverage"
	@echo "  web-lint           - Lint web interface"
	@echo "  web-dev            - Run web dev server"
	@echo "  web-install        - Install web dependencies"
	@echo ""
	@echo "For more service-specific commands (SEO tests, benchmarks, etc.):"
	@echo "  cd apps/controller && make help"
	@echo "  cd apps/scraper && make help"
	@echo "  cd apps/textanalyzer && make help"
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
	@$(MAKE) -C $(SCHEDULER_DIR) build
	@cd $(WEB_DIR) && npm run build
	@echo "All services built successfully!"

test: ## Run tests for all services
	@echo "Running tests for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) test
	@$(MAKE) -C $(SCRAPER_DIR) test
	@$(MAKE) -C $(TEXTANALYZER_DIR) test
	@$(MAKE) -C $(SCHEDULER_DIR) test
	@cd $(WEB_DIR) && npm test -- --run
	@echo "All tests completed!"

test-coverage: ## Run tests with coverage for all services
	@echo "Running tests with coverage for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) test-coverage
	@$(MAKE) -C $(SCRAPER_DIR) test-coverage
	@$(MAKE) -C $(TEXTANALYZER_DIR) test-coverage
	@$(MAKE) -C $(SCHEDULER_DIR) test
	@cd $(WEB_DIR) && npm run test:coverage -- --run
	@echo "All coverage reports generated!"

clean: ## Clean build artifacts for all services
	@echo "Cleaning all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) clean
	@$(MAKE) -C $(SCRAPER_DIR) clean
	@$(MAKE) -C $(TEXTANALYZER_DIR) clean
	@$(MAKE) -C $(SCHEDULER_DIR) clean
	@cd $(WEB_DIR) && rm -rf dist
	@echo "All services cleaned!"

install: ## Install dependencies for all services
	@echo "Installing dependencies for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) install
	@$(MAKE) -C $(SCRAPER_DIR) deps
	@$(MAKE) -C $(TEXTANALYZER_DIR) install
	@$(MAKE) -C $(SCHEDULER_DIR) deps
	@cd $(WEB_DIR) && npm install
	@echo "All dependencies installed!"

fmt: ## Format code for all services
	@echo "Formatting code for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) fmt
	@$(MAKE) -C $(SCRAPER_DIR) fmt
	@$(MAKE) -C $(TEXTANALYZER_DIR) fmt
	@$(MAKE) -C $(SCHEDULER_DIR) fmt
	@echo "All code formatted!"

lint: ## Lint code for all services
	@echo "Linting code for all services..."
	@$(MAKE) -C $(CONTROLLER_DIR) lint
	@$(MAKE) -C $(SCRAPER_DIR) vet
	@$(MAKE) -C $(TEXTANALYZER_DIR) lint
	@$(MAKE) -C $(SCHEDULER_DIR) fmt
	@cd $(WEB_DIR) && npm run lint
	@echo "All linting completed!"

# ==================== Docker Commands ====================

docker-build: ## Build all Docker images
	@echo "Building all Docker images..."
	@docker compose build
	@echo "All Docker images built!"

docker-up: ## Start all services with docker compose
	@echo "Starting all services..."
	@docker compose up -d
	@echo "All services started!"
	@echo "Web Interface:  http://localhost:3000"
	@echo "Controller:     http://localhost:8080"
	@echo "Scraper API:    http://localhost:8081"
	@echo "Text Analyzer:  http://localhost:8082"
	@echo "Scheduler:      http://localhost:8083"

docker-down: ## Stop all services
	@echo "Stopping all services..."
	@docker compose down
	@echo "All services stopped!"

docker-logs: ## View logs from all services
	@docker compose logs -f

docker-ps: ## Show running containers
	@docker compose ps

docker-clean: ## Remove all containers, volumes, and images
	@echo "Cleaning Docker resources..."
	@docker compose down -v --rmi all
	@echo "Docker cleanup complete!"

docker-restart: ## Restart all services
	@$(MAKE) docker-down
	@$(MAKE) docker-up

docker-rebuild: ## Rebuild and restart all services
	@echo "Rebuilding and restarting all services..."
	@docker compose down
	@docker compose build --no-cache
	@docker compose up -d
	@echo "All services rebuilt and restarted!"

docker-health: ## Check health of all services
	@echo "Checking health of all services..."
	@docker compose ps
	@echo ""
	@echo "Testing endpoints..."
	@echo -n "Controller (9080): " && curl -sf http://localhost:9080/health > /dev/null && echo "✓ Healthy" || echo "✗ Unhealthy"
	@echo -n "Scraper (9081): " && curl -sf http://localhost:9081/health > /dev/null && echo "✓ Healthy" || echo "✗ Unhealthy"
	@echo -n "TextAnalyzer (9082): " && curl -sf http://localhost:9082/health > /dev/null && echo "✓ Healthy" || echo "✗ Unhealthy"
	@echo -n "Scheduler (9083): " && curl -sf http://localhost:9083/health > /dev/null && echo "✓ Healthy" || echo "✗ Unhealthy"
	@echo -n "Web UI (3001): " && curl -sf http://localhost:3001 > /dev/null && echo "✓ Healthy" || echo "✗ Unhealthy"

# ==================== Docker Staging Commands ====================

docker-staging-build: ## Build all service images for staging
	@./build-staging.sh

docker-staging-push: ## Build and push all images to ghcr.io/zombar
	@./build-staging.sh push

docker-staging-deploy: ## Full local deploy: build and start all services (dev machine)
	@echo "🚀 Deploying to staging..."
	@./build-staging.sh
	@docker compose -f docker-compose.yml -f docker-compose.staging.yml up -d
	@echo ""
	@echo "✅ Staging deployment complete!"
	@echo "   Staging URL:   https://purpletab.honker (via reverse proxy)"
	@echo "   Local Web:     http://localhost:3001"
	@echo "   Local API:     http://localhost:9080"

docker-staging-up: ## Start all services in staging mode
	@docker compose -f docker-compose.yml -f docker-compose.staging.yml up -d

docker-staging-down: ## Stop all staging services
	@docker compose -f docker-compose.yml -f docker-compose.staging.yml down

docker-staging-logs: ## View logs from staging services
	@docker compose -f docker-compose.yml -f docker-compose.staging.yml logs -f

docker-staging-pull: ## Pull latest images and start services (for server)
	@./deploy-staging.sh

# ==================== Docker Per-Service Commands ====================

docker-logs-controller: ## View logs for controller service
	@docker compose logs -f controller

docker-logs-scraper: ## View logs for scraper service
	@docker compose logs -f scraper

docker-logs-textanalyzer: ## View logs for textanalyzer service
	@docker compose logs -f textanalyzer

docker-logs-scheduler: ## View logs for scheduler service
	@docker compose logs -f scheduler

docker-logs-web: ## View logs for web service
	@docker compose logs -f web

docker-restart-controller: ## Restart controller service
	@docker compose restart controller

docker-restart-scraper: ## Restart scraper service
	@docker compose restart scraper

docker-restart-textanalyzer: ## Restart textanalyzer service
	@docker compose restart textanalyzer

docker-restart-scheduler: ## Restart scheduler service
	@docker compose restart scheduler

docker-restart-web: ## Restart web service
	@docker compose restart web

docker-shell-controller: ## Open shell in controller container
	@docker compose exec controller sh

docker-shell-scraper: ## Open shell in scraper container
	@docker compose exec scraper sh

docker-shell-textanalyzer: ## Open shell in textanalyzer container
	@docker compose exec textanalyzer sh

docker-shell-scheduler: ## Open shell in scheduler container
	@docker compose exec scheduler sh

docker-shell-web: ## Open shell in web container
	@docker compose exec web sh

docker-exec-controller: ## Execute command in controller (use CMD='...')
	@docker compose exec controller $(CMD)

docker-exec-scraper: ## Execute command in scraper (use CMD='...')
	@docker compose exec scraper $(CMD)

docker-exec-textanalyzer: ## Execute command in textanalyzer (use CMD='...')
	@docker compose exec textanalyzer $(CMD)

docker-exec-scheduler: ## Execute command in scheduler (use CMD='...')
	@docker compose exec scheduler $(CMD)

docker-exec-web: ## Execute command in web (use CMD='...')
	@docker compose exec web $(CMD)

# ==================== Docker Management Commands ====================

docker-volumes: ## List all volumes
	@echo "Docker volumes:"
	@docker volume ls | grep purpletab || echo "No purpletab volumes found"

docker-volumes-inspect: ## Inspect volume usage
	@echo "Volume details:"
	@docker compose exec controller du -sh /app/data 2>/dev/null || echo "Controller volume not mounted"
	@docker compose exec scraper du -sh /app/data /app/storage 2>/dev/null || echo "Scraper volume not mounted"
	@docker compose exec textanalyzer du -sh /data 2>/dev/null || echo "TextAnalyzer volume not mounted"
	@docker compose exec scheduler du -sh /app/data 2>/dev/null || echo "Scheduler volume not mounted"

docker-volumes-clean: ## Remove unused volumes (WARNING: destructive)
	@echo "⚠️  WARNING: This will remove unused Docker volumes!"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read
	@docker volume prune -f
	@echo "Unused volumes removed!"

docker-images: ## List all purpletab images
	@echo "PurpleTab Docker images:"
	@docker images | grep -E "(REPOSITORY|purpletab)" || echo "No purpletab images found"

docker-prune: ## Remove unused containers, networks, and images
	@echo "Pruning unused Docker resources..."
	@docker system prune -f
	@echo "Docker prune complete!"

docker-prune-all: ## Remove ALL unused Docker resources including volumes (WARNING: destructive)
	@echo "⚠️  WARNING: This will remove ALL unused Docker resources including volumes!"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read
	@docker system prune -af --volumes
	@echo "All unused Docker resources removed!"

# Database access shortcuts
docker-db-controller: ## Access controller SQLite database
	@docker compose exec controller sqlite3 /app/data/controller.db

docker-db-scraper: ## Access scraper SQLite database
	@docker compose exec scraper sqlite3 /app/data/scraper.db

docker-db-textanalyzer: ## Access textanalyzer SQLite database
	@docker compose exec textanalyzer sqlite3 /data/textanalyzer.db

docker-db-scheduler: ## Access scheduler SQLite database
	@docker compose exec scheduler sqlite3 /app/data/scheduler.db

# Backup commands
docker-backup: ## Backup all databases and storage
	@echo "Creating backup..."
	@mkdir -p backups/$(shell date +%Y%m%d-%H%M%S)
	@docker cp $$(docker compose ps -q controller):/app/data/controller.db backups/$(shell date +%Y%m%d-%H%M%S)/
	@docker cp $$(docker compose ps -q scraper):/app/data/scraper.db backups/$(shell date +%Y%m%d-%H%M%S)/
	@docker cp $$(docker compose ps -q textanalyzer):/data/textanalyzer.db backups/$(shell date +%Y%m%d-%H%M%S)/
	@docker cp $$(docker compose ps -q scheduler):/app/data/scheduler.db backups/$(shell date +%Y%m%d-%H%M%S)/
	@echo "Backup complete! Location: backups/$(shell date +%Y%m%d-%H%M%S)/"

docker-stats: ## Show resource usage of running containers
	@docker stats --no-stream $$(docker compose ps -q)

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

# Scheduler commands
scheduler-build:
	@$(MAKE) -C $(SCHEDULER_DIR) build

scheduler-test:
	@$(MAKE) -C $(SCHEDULER_DIR) test

scheduler-clean:
	@$(MAKE) -C $(SCHEDULER_DIR) clean

scheduler-run:
	@$(MAKE) -C $(SCHEDULER_DIR) run

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
	@echo "  Terminal 4: make scheduler-run"
	@echo "  Terminal 5: make web-dev (optional - for UI)"
