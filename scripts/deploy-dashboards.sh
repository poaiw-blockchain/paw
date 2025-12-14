#!/bin/bash
# Deploy PAW Dashboards
# This script deploys the staking, validator, and governance dashboards

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/compose/docker-compose.dashboards.yml"

echo "========================================="
echo "PAW Dashboard Deployment"
echo "========================================="
echo ""

# Check if docker-compose exists
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "Error: docker-compose or docker compose not found"
    exit 1
fi

# Determine docker compose command
if docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo "Using: $DOCKER_COMPOSE"
echo ""

# Stop existing containers
echo "Stopping existing dashboard containers..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" down 2>/dev/null || true
echo ""

# Build and start containers
echo "Building dashboard images..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" build --no-cache
echo ""

echo "Starting dashboard services..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d
echo ""

# Wait for services to be healthy
echo "Waiting for dashboards to be ready..."
sleep 10

# Check health
echo ""
echo "========================================="
echo "Dashboard Status:"
echo "========================================="

$DOCKER_COMPOSE -f "$COMPOSE_FILE" ps

echo ""
echo "========================================="
echo "Dashboard URLs:"
echo "========================================="
echo "Staking Dashboard:    http://localhost:11100"
echo "Validator Dashboard:  http://localhost:11110"
echo "Governance Portal:    http://localhost:11120"
echo "========================================="
echo ""
echo "To view logs:"
echo "  $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f"
echo ""
echo "To stop dashboards:"
echo "  $SCRIPT_DIR/stop-dashboards.sh"
echo ""
echo "Deployment complete!"
