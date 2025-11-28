#!/bin/bash

###############################################################################
# PAW Blockchain - Connect to Network Example
#
# This example demonstrates how to connect to the PAW blockchain network
# and retrieve basic network information using curl.
#
# Usage:
#   ./connect.sh
#
# Environment Variables:
#   PAW_RPC_ENDPOINT - RPC endpoint URL (default: http://localhost:26657)
#
# Requirements:
#   - curl
#   - jq
###############################################################################

set -e

# Configuration
RPC_ENDPOINT="${PAW_RPC_ENDPOINT:-http://localhost:26657}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check requirements
command -v curl >/dev/null 2>&1 || {
    echo -e "${RED}✗ Error: curl is required but not installed${NC}" >&2
    exit 1
}

command -v jq >/dev/null 2>&1 || {
    echo -e "${RED}✗ Error: jq is required but not installed${NC}" >&2
    exit 1
}

echo "Connecting to PAW Network..."
echo "RPC Endpoint: $RPC_ENDPOINT"
echo ""

# Get node status
echo "Fetching node status..."
STATUS=$(curl -s "$RPC_ENDPOINT/status")

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to connect to node${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check if the RPC endpoint is correct"
    echo "  2. Ensure the node is running"
    echo "  3. Check firewall settings"
    exit 1
fi

echo -e "${GREEN}✓ Successfully connected to PAW network${NC}"
echo ""

# Extract information
CHAIN_ID=$(echo "$STATUS" | jq -r '.result.node_info.network')
HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height')
CATCHING_UP=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up')
VERSION=$(echo "$STATUS" | jq -r '.result.node_info.version')
MONIKER=$(echo "$STATUS" | jq -r '.result.node_info.moniker')

echo "Chain ID: $CHAIN_ID"
echo "Current Block Height: $HEIGHT"
echo "Node Version: $VERSION"
echo "Moniker: $MONIKER"
echo "Syncing: $([ "$CATCHING_UP" = "true" ] && echo "Yes" || echo "No")"

# Get latest block
echo ""
echo "Fetching latest block..."
BLOCK=$(curl -s "$RPC_ENDPOINT/block")

BLOCK_HASH=$(echo "$BLOCK" | jq -r '.result.block_id.hash')
BLOCK_TIME=$(echo "$BLOCK" | jq -r '.result.block.header.time')
NUM_TXS=$(echo "$BLOCK" | jq -r '.result.block.data.txs | length')
PROPOSER=$(echo "$BLOCK" | jq -r '.result.block.header.proposer_address')

echo ""
echo "Latest Block Info:"
echo "  Block Hash: $BLOCK_HASH"
echo "  Time: $BLOCK_TIME"
echo "  Num Transactions: $NUM_TXS"
echo "  Proposer: $PROPOSER"

# Calculate average block time
if [ "$HEIGHT" -gt 5 ]; then
    PREV_HEIGHT=$((HEIGHT - 5))
    PREV_BLOCK=$(curl -s "$RPC_ENDPOINT/block?height=$PREV_HEIGHT")
    PREV_TIME=$(echo "$PREV_BLOCK" | jq -r '.result.block.header.time')

    # Convert times to seconds (simplified - requires date command)
    if command -v date >/dev/null 2>&1; then
        CURR_SECONDS=$(date -d "$BLOCK_TIME" +%s 2>/dev/null || echo "0")
        PREV_SECONDS=$(date -d "$PREV_TIME" +%s 2>/dev/null || echo "0")

        if [ "$CURR_SECONDS" != "0" ] && [ "$PREV_SECONDS" != "0" ]; then
            TIME_DIFF=$((CURR_SECONDS - PREV_SECONDS))
            AVG_BLOCK_TIME=$(echo "scale=2; $TIME_DIFF / 5" | bc 2>/dev/null || echo "N/A")
            if [ "$AVG_BLOCK_TIME" != "N/A" ]; then
                echo "  Average Block Time: ${AVG_BLOCK_TIME}s"
            fi
        fi
    fi
fi

echo ""
echo -e "${GREEN}✓ Network information retrieved successfully${NC}"
