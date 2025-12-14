#!/bin/bash
# Verify PAW Dashboards
# This script performs health checks on all deployed dashboards

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/compose/docker-compose.dashboards.yml"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "PAW Dashboard Health Check"
echo "========================================="
echo ""

# Determine docker compose command
if docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Check if containers are running
echo "Checking container status..."
STAKING_RUNNING=$($DOCKER_COMPOSE -f "$COMPOSE_FILE" ps -q staking-dashboard 2>/dev/null)
VALIDATOR_RUNNING=$($DOCKER_COMPOSE -f "$COMPOSE_FILE" ps -q validator-dashboard 2>/dev/null)
GOVERNANCE_RUNNING=$($DOCKER_COMPOSE -f "$COMPOSE_FILE" ps -q governance-portal 2>/dev/null)

if [ -z "$STAKING_RUNNING" ] && [ -z "$VALIDATOR_RUNNING" ] && [ -z "$GOVERNANCE_RUNNING" ]; then
    echo -e "${RED}Error: No dashboard containers are running${NC}"
    echo "Run: $SCRIPT_DIR/deploy-dashboards.sh"
    exit 1
fi

echo ""

# Function to check HTTP endpoint
check_endpoint() {
    local name=$1
    local url=$2

    if curl -sf "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name is accessible at $url"
        return 0
    else
        echo -e "${RED}✗${NC} $name is NOT accessible at $url"
        return 1
    fi
}

# Check endpoints
ALL_PASS=0

echo "Checking HTTP endpoints..."
check_endpoint "Staking Dashboard" "http://localhost:11100" || ALL_PASS=1
check_endpoint "Validator Dashboard" "http://localhost:11110" || ALL_PASS=1
check_endpoint "Governance Portal" "http://localhost:11120" || ALL_PASS=1

echo ""

# Check container health
echo "Checking container health..."
STAKING_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' paw-staking-dashboard 2>/dev/null || echo "unknown")
VALIDATOR_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' paw-validator-dashboard 2>/dev/null || echo "unknown")
GOVERNANCE_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' paw-governance-portal 2>/dev/null || echo "unknown")

if [ "$STAKING_HEALTH" = "healthy" ]; then
    echo -e "${GREEN}✓${NC} Staking Dashboard is healthy"
else
    echo -e "${YELLOW}⚠${NC} Staking Dashboard health: $STAKING_HEALTH"
    ALL_PASS=1
fi

if [ "$VALIDATOR_HEALTH" = "healthy" ]; then
    echo -e "${GREEN}✓${NC} Validator Dashboard is healthy"
else
    echo -e "${YELLOW}⚠${NC} Validator Dashboard health: $VALIDATOR_HEALTH"
    ALL_PASS=1
fi

if [ "$GOVERNANCE_HEALTH" = "healthy" ]; then
    echo -e "${GREEN}✓${NC} Governance Portal is healthy"
else
    echo -e "${YELLOW}⚠${NC} Governance Portal health: $GOVERNANCE_HEALTH"
    ALL_PASS=1
fi

echo ""

# Show resource usage
echo "Container Resource Usage:"
docker stats --no-stream paw-staking-dashboard paw-validator-dashboard paw-governance-portal 2>/dev/null || true

echo ""
echo "========================================="
if [ $ALL_PASS -eq 0 ]; then
    echo -e "${GREEN}All dashboards are healthy and accessible!${NC}"
else
    echo -e "${YELLOW}Some dashboards may have issues. Check logs:${NC}"
    echo "  $DOCKER_COMPOSE -f $COMPOSE_FILE logs"
fi
echo "========================================="

exit $ALL_PASS
