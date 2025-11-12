.PHONY: help build run test clean docker-build docker-up docker-down lint format

# Variables
BINARY_NAME=paw-api
DOCKER_IMAGE=paw-api:latest
DOCKER_COMPOSE=docker-compose

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the API server binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/main.go
	@echo "Build complete: $(BINARY_NAME)"

run: ## Run the API server
	@echo "Starting $(BINARY_NAME)..."
	@go run ./cmd/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) -f Dockerfile ..
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Containers started. API: http://localhost:5000"

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@$(DOCKER_COMPOSE) down
	@echo "Containers stopped"

docker-logs: ## View Docker container logs
	@$(DOCKER_COMPOSE) logs -f api

docker-restart: ## Restart Docker containers
	@echo "Restarting Docker containers..."
	@$(DOCKER_COMPOSE) restart
	@echo "Containers restarted"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

format: ## Format code
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .
	@echo "Code formatted"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies downloaded"

dev: ## Run in development mode with hot reload
	@echo "Starting development server..."
	@air || go run ./cmd/main.go

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/cosmtrek/air@latest
	@echo "Tools installed"

health-check: ## Check API health
	@curl -s http://localhost:5000/health | jq .

api-test: ## Run API integration tests
	@echo "Testing registration..."
	@curl -s -X POST http://localhost:5000/api/auth/register \
		-H "Content-Type: application/json" \
		-d '{"username":"testuser","password":"test123"}' | jq .
	@echo "\nTesting login..."
	@curl -s -X POST http://localhost:5000/api/auth/login \
		-H "Content-Type: application/json" \
		-d '{"username":"testuser","password":"test123"}' | jq .

.DEFAULT_GOAL := help
