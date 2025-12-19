#!/bin/bash
# PAW Flask Explorer - Staging Deployment Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

COMPOSE_CMD=""

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    else
        log_error "docker compose plugin is not installed"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Load environment variables
load_env() {
    log_info "Loading staging environment variables..."

    if [ -f .env.staging ]; then
        export $(cat .env.staging | grep -v '^#' | xargs)
        log_success "Loaded .env.staging"
    else
        log_error ".env.staging file not found"
        exit 1
    fi
}

# Build Docker images
build_images() {
    log_info "Building Docker images..."

    $COMPOSE_CMD -f docker-compose.staging.yml build --no-cache

    log_success "Docker images built successfully"
}

# Start services
start_services() {
    log_info "Starting staging services..."

    $COMPOSE_CMD -f docker-compose.staging.yml up -d

    log_success "Services started"
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to become healthy..."

    local max_attempts=60
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if $COMPOSE_CMD -f docker-compose.staging.yml ps | grep -q "(healthy)"; then
            local healthy_count=$($COMPOSE_CMD -f docker-compose.staging.yml ps | grep -c "(healthy)" || true)
            log_info "Healthy services: $healthy_count"

            if [ "$healthy_count" -ge 3 ]; then
                log_success "All services are healthy"
                return 0
            fi
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done

    log_error "Services did not become healthy in time"
    log_info "Checking service status..."
    $COMPOSE_CMD -f docker-compose.staging.yml ps
    return 1
}

# Run health checks
run_health_checks() {
    log_info "Running health checks..."

    # Check Flask health endpoint
    log_info "Checking Flask health..."
    if curl -f http://localhost:11083/health &> /dev/null; then
        log_success "Flask health check passed"
    else
        log_error "Flask health check failed"
        return 1
    fi

    # Check if indexer is accessible
    log_info "Checking indexer health..."
    if curl -f http://localhost:11081/health &> /dev/null; then
        log_success "Indexer health check passed"
    else
        log_warning "Indexer health check failed (may still be initializing)"
    fi

    # Check PostgreSQL
    log_info "Checking PostgreSQL..."
    if $COMPOSE_CMD -f docker-compose.staging.yml exec -T paw-staging-postgres pg_isready -U paw_staging &> /dev/null; then
        log_success "PostgreSQL is ready"
    else
        log_error "PostgreSQL is not ready"
        return 1
    fi

    log_success "Health checks completed"
}

# Show service URLs
show_urls() {
    log_info "Staging deployment URLs:"
    echo ""
    echo "  Explorer:    http://localhost:11083"
    echo "  Indexer:     http://localhost:11081"
    echo "  PostgreSQL:  localhost:11432"
    echo "  Prometheus:  http://localhost:11091"
    echo ""
    log_info "Use 'docker-compose -f docker-compose.staging.yml logs -f' to view logs"
}

# Main deployment flow
main() {
    log_info "Starting PAW Flask Explorer staging deployment..."
    echo ""

    check_prerequisites
    load_env

    # Stop existing staging deployment
    log_info "Stopping existing staging deployment (if any)..."
    $COMPOSE_CMD -f docker-compose.staging.yml down 2>/dev/null || true

    build_images
    start_services

    if wait_for_services; then
        run_health_checks
        show_urls
        log_success "Staging deployment completed successfully!"
        exit 0
    else
        log_error "Staging deployment failed"
        log_info "Check logs with: $COMPOSE_CMD -f docker-compose.staging.yml logs"
        exit 1
    fi
}

# Run main function
main "$@"
