.PHONY: help setup clean build run dev test db-start db-stop db-migrate-up db-migrate-down db-shell sqlc-generate web-build web-dev server-dev server-build docker-build docker-run

# Variables
BINARY_NAME=journal
DOCKER_IMAGE=journal-app
DB_URL=postgresql://journal:journaldev@localhost:5432/journal?sslmode=disable
PORT=8080

# Colors for output
GREEN=\033[0;32m
BLUE=\033[0;34m
NC=\033[0m

help: ## Show this help message
	@echo "$(BLUE)Journal Web App - Makefile Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

## Setup & Installation

setup: ## Initial setup (first time only)
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@$(MAKE) db-start
	@sleep 3
	@$(MAKE) db-migrate-up
	@podman exec journal-postgres psql -U journal -d journal -c \
		"INSERT INTO users (email, display_name) VALUES ('test@example.com', 'Test User') ON CONFLICT DO NOTHING;" || true
	@$(MAKE) sqlc-generate
	@cd web && npm install
	@go mod download
	@echo "$(GREEN)Setup complete! Run 'make run' to start the app.$(NC)"

## Database Commands

db-start: ## Start PostgreSQL container
	@echo "$(GREEN)Starting PostgreSQL container...$(NC)"
	@podman start journal-postgres 2>/dev/null || \
		podman run -d --name journal-postgres \
		-e POSTGRES_PASSWORD=journaldev \
		-e POSTGRES_USER=journal \
		-e POSTGRES_DB=journal \
		-p 5432:5432 \
		postgres:16-alpine
	@sleep 2
	@podman exec journal-postgres pg_isready -U journal

db-stop: ## Stop PostgreSQL container
	@echo "$(GREEN)Stopping PostgreSQL container...$(NC)"
	@podman stop journal-postgres

db-migrate-up: ## Run database migrations
	@echo "$(GREEN)Running migrations...$(NC)"
	@migrate -path db/migrations -database "$(DB_URL)" up

db-migrate-down: ## Rollback last migration
	@echo "$(GREEN)Rolling back migration...$(NC)"
	@migrate -path db/migrations -database "$(DB_URL)" down 1

db-shell: ## Open PostgreSQL shell
	@podman exec -it journal-postgres psql -U journal -d journal

db-reset: db-migrate-down db-migrate-up ## Reset database (down then up)

## Code Generation

sqlc-generate: ## Generate type-safe database code
	@echo "$(GREEN)Generating sqlc code...$(NC)"
	@cd db && sqlc generate

## Frontend Commands

web-build: ## Build frontend for production
	@echo "$(GREEN)Building frontend...$(NC)"
	@cd web && npm run build

web-dev: ## Start frontend dev server (Vite)
	@echo "$(GREEN)Starting Vite dev server...$(NC)"
	@cd web && npm run dev

web-install: ## Install frontend dependencies
	@cd web && npm install

## Backend Commands

server-dev: ## Start backend server (development mode)
	@echo "$(GREEN)Starting backend server...$(NC)"
	@SPA_MODE=fs go run ./cmd/server

server-build: ## Build backend binary
	@echo "$(GREEN)Building backend binary...$(NC)"
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)Binary created: bin/$(BINARY_NAME)$(NC)"

server-run: server-build ## Build and run backend binary
	@./bin/$(BINARY_NAME)

## Main Commands

build: web-build server-build ## Build both frontend and backend

run: web-build ## Build frontend and run server
	@echo "$(GREEN)Starting application on http://localhost:$(PORT)$(NC)"
	@SPA_MODE=fs go run ./cmd/server

run-once: web-build ## Build frontend and run server (alias for run)
	@$(MAKE) run

dev: ## Run in development mode (auto-reload recommended with air)
	@$(MAKE) -j2 web-dev server-dev

## Testing

test: ## Run Go tests
	@go test -v ./...

test-api: ## Test API endpoints
	@echo "$(GREEN)Testing API endpoints...$(NC)"
	@echo "Creating test entry..."
	@curl -s -X POST http://localhost:$(PORT)/api/entries \
		-H "Content-Type: application/json" \
		-d '{"title":"Test Entry","body_delta":{"ops":[{"insert":"Test\n"}]},"type":"notes","date":"2025-11-20"}' | jq .
	@echo "\nListing entries..."
	@curl -s http://localhost:$(PORT)/api/days/2025-11-20/entries | jq .

seed-data: ## Generate test data (3000 entries across 365 days)
	@echo "$(GREEN)Generating test data...$(NC)"
	@go run ./cmd/seed

## Cleanup

clean: ## Remove build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf web/dist web/node_modules bin generated cmd/server/web
	@go clean

clean-db: ## Remove database container (WARNING: destroys data)
	@echo "$(GREEN)Removing database container...$(NC)"
	@podman rm -f journal-postgres

## Docker Commands

docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):latest .

docker-run: ## Run Docker container
	@echo "$(GREEN)Running Docker container...$(NC)"
	@docker run -p $(PORT):$(PORT) \
		-e DATABASE_URL=$(DB_URL) \
		$(DOCKER_IMAGE):latest

## Utility Commands

fmt: ## Format Go code
	@go fmt ./...

lint: ## Run linter
	@golangci-lint run || go vet ./...

deps: ## Download Go dependencies
	@go mod download
	@go mod tidy

update-deps: ## Update Go dependencies
	@go get -u ./...
	@go mod tidy

## Quick Start Commands

all: clean setup build ## Clean, setup, and build everything

quick-start: db-start web-build run ## Quick start (assumes DB already set up)

restart: ## Restart the application
	@echo "$(GREEN)Restarting application...$(NC)"
	@pkill -f "go run ./cmd/server" || true
	@$(MAKE) run

## Information

info: ## Show application info
	@echo "$(BLUE)Journal Web App$(NC)"
	@echo "  Backend:  http://localhost:$(PORT)"
	@echo "  Database: localhost:5432"
	@echo "  User:     journal"
	@echo "  Password: journaldev"
	@echo ""
	@echo "Available routes:"
	@echo "  GET    /api/days/:date/entries"
	@echo "  POST   /api/entries"
	@echo "  PATCH  /api/entries/:id"
	@echo "  DELETE /api/entries/:id"
	@echo "  GET    /api/months/:yearmonth/entry-days"

status: ## Check service status
	@echo "$(GREEN)Checking service status...$(NC)"
	@echo -n "Database: "
	@podman ps | grep journal-postgres > /dev/null && echo "$(GREEN)Running$(NC)" || echo "$(BLUE)Stopped$(NC)"
	@echo -n "API: "
	@curl -s http://localhost:$(PORT)/api/months/2025-11/entry-days > /dev/null && echo "$(GREEN)Running$(NC)" || echo "$(BLUE)Stopped$(NC)"
