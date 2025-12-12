#!/bin/bash
# Check status of PAW Block Explorer and related services
#
# This script checks:
# - PAW node RPC availability
# - Explorer web server status
# - Current block height
# - Network sync status

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}PAW Explorer Status Check${NC}"
echo -e "${BLUE}=================================${NC}"
echo ""

# Check RPC endpoint
echo -e "${YELLOW}Checking RPC endpoint (localhost:26657)...${NC}"
if curl -s http://localhost:26657/status > /dev/null 2>&1; then
    STATUS=$(curl -s http://localhost:26657/status)
    BLOCK_HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height')
    CHAIN_ID=$(echo "$STATUS" | jq -r '.result.node_info.network')
    CATCHING_UP=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up')
    NODE_VERSION=$(echo "$STATUS" | jq -r '.result.node_info.version')

    echo -e "${GREEN}✓ RPC is available${NC}"
    echo -e "  Chain ID: ${GREEN}$CHAIN_ID${NC}"
    echo -e "  Block Height: ${GREEN}$BLOCK_HEIGHT${NC}"
    echo -e "  Node Version: ${GREEN}$NODE_VERSION${NC}"

    if [ "$CATCHING_UP" == "true" ]; then
        echo -e "  Sync Status: ${YELLOW}Syncing...${NC}"
    else
        echo -e "  Sync Status: ${GREEN}Synced${NC}"
    fi
else
    echo -e "${RED}✗ RPC is not available${NC}"
    echo -e "  ${YELLOW}Make sure the PAW node is running${NC}"
fi
echo ""

# Check Explorer web server
echo -e "${YELLOW}Checking Explorer web server (localhost:11080)...${NC}"
if curl -s http://localhost:11080/ > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Explorer is running${NC}"
    echo -e "  URL: ${GREEN}http://localhost:11080${NC}"

    # Try to get homepage title
    TITLE=$(curl -s http://localhost:11080/ | grep -o '<title>.*</title>' | sed 's/<[^>]*>//g')
    if [ -n "$TITLE" ]; then
        echo -e "  Page: ${GREEN}$TITLE${NC}"
    fi
else
    echo -e "${RED}✗ Explorer is not running${NC}"
    echo -e "  ${YELLOW}Start with: ./scripts/start-explorer.sh${NC}"
fi
echo ""

# Check Docker container
echo -e "${YELLOW}Checking Docker container...${NC}"
if docker ps --format '{{.Names}}' | grep -q "paw-explorer"; then
    echo -e "${GREEN}✓ Docker container is running${NC}"
    CONTAINER_STATUS=$(docker inspect paw-explorer --format='{{.State.Status}}')
    echo -e "  Status: ${GREEN}$CONTAINER_STATUS${NC}"

    # Show recent logs
    echo -e "  Recent logs:"
    docker logs paw-explorer --tail 5 2>&1 | sed 's/^/    /'
else
    echo -e "${YELLOW}✗ Docker container is not running${NC}"
    echo -e "  ${YELLOW}Start with: cd docker && docker-compose up -d explorer${NC}"
fi
echo ""

# Check validators
echo -e "${YELLOW}Checking validator set...${NC}"
if curl -s http://localhost:26657/validators > /dev/null 2>&1; then
    VALIDATORS=$(curl -s http://localhost:26657/validators)
    VAL_COUNT=$(echo "$VALIDATORS" | jq -r '.result.validators | length')
    TOTAL_POWER=$(echo "$VALIDATORS" | jq -r '[.result.validators[].voting_power | tonumber] | add')

    echo -e "${GREEN}✓ Validator data available${NC}"
    echo -e "  Total Validators: ${GREEN}$VAL_COUNT${NC}"
    echo -e "  Total Voting Power: ${GREEN}$TOTAL_POWER${NC}"
else
    echo -e "${RED}✗ Cannot fetch validator data${NC}"
fi
echo ""

# Summary
echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}Quick Access URLs${NC}"
echo -e "${BLUE}=================================${NC}"
echo -e "Explorer:     ${GREEN}http://localhost:11080${NC}"
echo -e "Grafana:      ${GREEN}http://localhost:11030${NC}"
echo -e "Prometheus:   ${GREEN}http://localhost:11090${NC}"
echo -e "RPC:          ${GREEN}http://localhost:26657${NC}"
echo ""
