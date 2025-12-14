# PAW Flask Explorer Makefile

.PHONY: help build up down restart logs clean test health metrics

# Default target
.DEFAULT_GOAL := help

# Variables
COMPOSE := docker-compose
FLASK_CONTAINER := paw-flask
NGINX_CONTAINER := paw-nginx

help: ## Show this help message
	@echo "PAW Flask Explorer - Available Commands"
	@echo "========================================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build Docker images
	$(COMPOSE) build --no-cache

up: ## Start all services
	$(COMPOSE) up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@make health

down: ## Stop all services
	$(COMPOSE) down

restart: ## Restart all services
	$(COMPOSE) restart
	@sleep 3
	@make health

logs: ## Show logs (use 'make logs-follow' to follow)
	$(COMPOSE) logs --tail=100

logs-follow: ## Follow logs in real-time
	$(COMPOSE) logs -f

logs-flask: ## Show Flask logs
	$(COMPOSE) logs --tail=100 $(FLASK_CONTAINER)

logs-nginx: ## Show nginx logs
	$(COMPOSE) logs --tail=100 $(NGINX_CONTAINER)

ps: ## Show running containers
	$(COMPOSE) ps

health: ## Check health status
	@echo "Checking health status..."
	@curl -sf http://localhost:11080/health || echo "Health check failed"
	@echo ""
	@echo "Checking readiness..."
	@curl -sf http://localhost:11080/health/ready || echo "Readiness check failed"

metrics: ## Show Prometheus metrics
	@curl -s http://localhost:11080/metrics

stats: ## Show network statistics
	@curl -s http://localhost:11080/api/v1/stats | python3 -m json.tool

test-endpoints: ## Test all major endpoints
	@echo "Testing endpoints..."
	@echo "Health: " && curl -s http://localhost:11080/health | python3 -m json.tool
	@echo "\nStats: " && curl -s http://localhost:11080/api/v1/stats | python3 -m json.tool
	@echo "\nBlocks: " && curl -s http://localhost:11080/api/v1/blocks | python3 -m json.tool | head -20

shell-flask: ## Open shell in Flask container
	$(COMPOSE) exec $(FLASK_CONTAINER) /bin/bash

shell-nginx: ## Open shell in nginx container
	$(COMPOSE) exec $(NGINX_CONTAINER) /bin/sh

clean: ## Remove containers, volumes, and cache
	$(COMPOSE) down -v
	@echo "Cleaned up containers and volumes"

clean-cache: ## Clear nginx cache
	$(COMPOSE) exec $(NGINX_CONTAINER) rm -rf /var/cache/nginx/* || true
	$(COMPOSE) restart $(NGINX_CONTAINER)
	@echo "Nginx cache cleared"

rebuild: clean build up ## Clean, rebuild, and start services

dev: ## Run in development mode (local Python)
	@echo "Running in development mode..."
	export FLASK_ENV=development FLASK_DEBUG=true && python3 app.py

install: ## Install Python dependencies locally
	pip3 install -r requirements.txt

lint: ## Run Python linter
	@echo "Running flake8..."
	@flake8 app.py || echo "Install flake8: pip install flake8"

format: ## Format Python code
	@echo "Running black..."
	@black app.py || echo "Install black: pip install black"

backup: ## Backup configuration files
	@echo "Creating backup..."
	@tar -czf flask-explorer-backup-$$(date +%Y%m%d-%H%M%S).tar.gz \
		.env docker-compose.yml nginx.conf requirements.txt Makefile README.md
	@echo "Backup created"

open: ## Open explorer in browser
	@python3 -m webbrowser http://localhost:11080

monitor: ## Monitor container resources
	docker stats $(FLASK_CONTAINER) $(NGINX_CONTAINER)

update: ## Pull latest changes and restart
	git pull
	$(COMPOSE) pull
	make rebuild

version: ## Show versions
	@echo "Docker Compose version:"
	@$(COMPOSE) version
	@echo "\nPython version:"
	@python3 --version
	@echo "\nFlask container version:"
	@$(COMPOSE) exec $(FLASK_CONTAINER) python --version 2>/dev/null || echo "Container not running"
