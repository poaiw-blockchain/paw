.PHONY: all build test test-unit test-integration run clean docker-build docker-up docker-down

# Variables
BINARY_NAME=status-server
GO=go
GOFLAGS=-v
DOCKER_COMPOSE=docker-compose

all: test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	cd backend && $(GO) build $(GOFLAGS) -o ../$(BINARY_NAME) .

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	cd tests/unit && $(GO) mod download && $(GO) test $(GOFLAGS) -cover -timeout 30s

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	cd tests/integration && $(GO) mod download && $(GO) test $(GOFLAGS) -cover -timeout 60s

# Run the application
run:
	@echo "Starting $(BINARY_NAME)..."
	cd backend && $(GO) run main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	cd backend && $(GO) clean
	rm -f coverage.out

# Docker commands
docker-build:
	@echo "Building Docker image..."
	$(DOCKER_COMPOSE) build

docker-up:
	@echo "Starting Docker containers..."
	$(DOCKER_COMPOSE) up -d

docker-down:
	@echo "Stopping Docker containers..."
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Development
dev:
	@echo "Running in development mode..."
	cd backend && $(GO) run main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	cd backend && $(GO) mod download
	cd tests/unit && $(GO) mod download
	cd tests/integration && $(GO) mod download

# Format code
fmt:
	@echo "Formatting code..."
	cd backend && $(GO) fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	cd backend && golangci-lint run

# Coverage report
coverage:
	@echo "Generating coverage report..."
	cd backend && $(GO) test ./... -coverprofile=../coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
