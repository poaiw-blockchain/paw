#!/bin/bash

# PAW Blockchain - Docker Deployment Script
# This script deploys the PAW blockchain using Docker and Docker Compose

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_DIR="$PROJECT_ROOT/docker"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    log_info "Docker version: $(docker --version)"

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    log_info "Docker Compose version: $(docker-compose --version 2>/dev/null || docker compose version)"

    # Check available disk space (at least 100GB recommended)
    available_space=$(df -BG "$PROJECT_ROOT" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$available_space" -lt 100 ]; then
        log_warn "Less than 100GB of disk space available. Current: ${available_space}GB"
        log_warn "This may not be sufficient for a full node."
    else
        log_info "Available disk space: ${available_space}GB"
    fi
}

create_env_file() {
    log_step "Creating environment file..."

    ENV_FILE="$DOCKER_DIR/.env"

    if [ -f "$ENV_FILE" ]; then
        log_warn "Environment file already exists. Backing up..."
        cp "$ENV_FILE" "$ENV_FILE.backup.$(date +%Y%m%d_%H%M%S)"
    fi

    cat > "$ENV_FILE" << EOF
# PAW Blockchain Docker Configuration
# Generated on $(date)

# Chain Configuration
CHAIN_ID=${CHAIN_ID:-paw-1}
MONIKER=${MONIKER:-paw-docker-node}

# Network Configuration
PERSISTENT_PEERS=${PERSISTENT_PEERS:-}
SEEDS=${SEEDS:-}

# State Sync Configuration (Optional - enables faster initial sync)
STATE_SYNC_ENABLED=${STATE_SYNC_ENABLED:-false}
STATE_SYNC_RPC_SERVERS=${STATE_SYNC_RPC_SERVERS:-}
STATE_SYNC_TRUST_HEIGHT=${STATE_SYNC_TRUST_HEIGHT:-}
STATE_SYNC_TRUST_HASH=${STATE_SYNC_TRUST_HASH:-}

# Security
JWT_SECRET=${JWT_SECRET:-$(openssl rand -base64 32)}
TLS_ENABLED=${TLS_ENABLED:-false}

# Monitoring
GRAFANA_USER=${GRAFANA_USER:-admin}
GRAFANA_PASSWORD=${GRAFANA_PASSWORD:-$(openssl rand -base64 12)}
EOF

    log_info "Environment file created at: $ENV_FILE"
    log_warn "IMPORTANT: Save your JWT_SECRET and GRAFANA_PASSWORD from the .env file!"
}

download_genesis() {
    log_step "Downloading genesis file..."

    GENESIS_URL="${GENESIS_URL:-https://rawhubusercontent.com/paw-chain/networks/main/mainnet/genesis.json}"
    GENESIS_FILE="$DOCKER_DIR/genesis.json"

    if [ -f "$GENESIS_FILE" ]; then
        log_warn "Genesis file already exists. Skipping download."
        log_info "To re-download, delete: $GENESIS_FILE"
        return
    fi

    log_info "Downloading from: $GENESIS_URL"
    if curl -L "$GENESIS_URL" -o "$GENESIS_FILE" --fail --silent --show-error; then
        log_info "Genesis file downloaded successfully"

        # Validate genesis file
        if ! jq empty "$GENESIS_FILE" 2>/dev/null; then
            log_error "Downloaded genesis file is not valid JSON"
            rm -f "$GENESIS_FILE"
            exit 1
        fi

        log_info "Genesis file validated"
    else
        log_error "Failed to download genesis file"
        log_info "You can manually place genesis.json in: $DOCKER_DIR/"
        exit 1
    fi
}

build_images() {
    log_step "Building Docker images..."

    cd "$DOCKER_DIR"

    if docker-compose build --no-cache; then
        log_info "Docker images built successfully"
    else
        log_error "Failed to build Docker images"
        exit 1
    fi
}

start_services() {
    log_step "Starting services..."

    cd "$DOCKER_DIR"

    # Start services in detached mode
    if docker-compose up -d; then
        log_info "Services started successfully"
    else
        log_error "Failed to start services"
        exit 1
    fi

    # Wait for services to be healthy
    log_info "Waiting for services to become healthy..."
    sleep 10

    # Check service status
    docker-compose ps
}

display_info() {
    log_step "Deployment complete!"

    echo ""
    echo "========================================"
    echo "PAW Blockchain - Access Information"
    echo "========================================"
    echo ""
    echo "RPC Endpoint:        http://localhost:26657"
    echo "REST API:            http://localhost:1317"
    echo "Custom API:          http://localhost:5000"
    echo "gRPC:                localhost:9090"
    echo ""
    echo "Monitoring:"
    echo "  Prometheus:        http://localhost:9091"
    echo "  Grafana:           http://localhost:3001"
    echo "  AlertManager:      http://localhost:9093"
    echo ""
    echo "Credentials (from .env file):"
    echo "  Grafana User:      ${GRAFANA_USER:-admin}"
    echo "  Grafana Password:  (check $DOCKER_DIR/.env)"
    echo ""
    echo "Useful Commands:"
    echo "  View logs:         cd $DOCKER_DIR && docker-compose logs -f paw-node"
    echo "  Stop services:     cd $DOCKER_DIR && docker-compose down"
    echo "  Restart services:  cd $DOCKER_DIR && docker-compose restart"
    echo "  Check status:      cd $DOCKER_DIR && docker-compose ps"
    echo ""
    echo "========================================"
}

show_logs() {
    log_step "Showing recent logs..."
    cd "$DOCKER_DIR"
    docker-compose logs --tail=50 paw-node
}

# Main execution
main() {
    echo "========================================"
    echo "PAW Blockchain - Docker Deployment"
    echo "========================================"
    echo ""

    # Parse command line arguments
    SKIP_BUILD=false
    SKIP_GENESIS=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-genesis)
                SKIP_GENESIS=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-build      Skip Docker image build"
                echo "  --skip-genesis    Skip genesis file download"
                echo "  --help            Show this help message"
                echo ""
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    check_prerequisites
    create_env_file

    if [ "$SKIP_GENESIS" = false ]; then
        download_genesis
    fi

    if [ "$SKIP_BUILD" = false ]; then
        build_images
    fi

    start_services
    display_info

    # Ask if user wants to see logs
    read -p "Would you like to view the logs? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        show_logs
    fi
}

# Run main function
main "$@"
