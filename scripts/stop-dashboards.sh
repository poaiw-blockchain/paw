#!/bin/bash
# Stop PAW Dashboards
# This script stops all dashboard services

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/compose/docker-compose.dashboards.yml"

echo "========================================="
echo "Stopping PAW Dashboards"
echo "========================================="
echo ""

# Determine docker compose command
if docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Stop and remove containers
echo "Stopping dashboard services..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" down

echo ""
echo "Dashboard services stopped."
echo ""
echo "To start dashboards again:"
echo "  $SCRIPT_DIR/deploy-dashboards.sh"
echo ""
