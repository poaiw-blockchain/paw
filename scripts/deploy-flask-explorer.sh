#!/bin/bash
#
# Deploy Flask Explorer for PAW Blockchain
#
# This script builds and starts the Flask blockchain explorer on port 11080.
# The explorer provides a web interface to view blocks, transactions, and validators.
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/compose/docker-compose.flask-explorer.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info &> /dev/null; then
    log_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if RPC node is accessible
RPC_URL="${RPC_URL:-http://localhost:26657}"
log_info "Checking RPC node at $RPC_URL..."
if curl -sf "$RPC_URL/status" &> /dev/null; then
    log_info "RPC node is accessible"
else
    log_warn "RPC node at $RPC_URL is not accessible. The explorer may not function correctly."
    log_warn "Make sure your PAW node is running before accessing the explorer."
fi

# Build and start Flask explorer
log_info "Building and starting Flask explorer..."
cd "$PROJECT_ROOT/compose"

docker compose -f docker-compose.flask-explorer.yml up -d --build

# Wait for service to be healthy
log_info "Waiting for Flask explorer to be ready..."
for i in {1..30}; do
    if curl -sf http://localhost:11080/ &> /dev/null; then
        log_info "Flask explorer is ready!"
        echo ""
        log_info "Flask Explorer deployed successfully!"
        echo ""
        log_info "Access the explorer at: http://localhost:11080"
        log_info "Dashboard: http://localhost:11080/"
        log_info "Validators: http://localhost:11080/validators"
        log_info "Search: http://localhost:11080/search"
        echo ""
        log_info "API Endpoints:"
        log_info "  Status: http://localhost:11080/api/status"
        log_info "  Block: http://localhost:11080/api/block/<height>"
        log_info "  Validators: http://localhost:11080/api/validators"
        echo ""
        log_info "To view logs: docker logs -f paw-flask-explorer"
        log_info "To stop: $SCRIPT_DIR/stop-flask-explorer.sh"
        exit 0
    fi
    sleep 2
done

log_error "Flask explorer did not become ready in time"
log_info "Check logs with: docker logs paw-flask-explorer"
exit 1
