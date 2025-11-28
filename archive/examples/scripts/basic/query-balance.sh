#!/bin/bash

###############################################################################
# PAW Blockchain - Query Balance Example
#
# This example demonstrates how to query account balances using curl.
#
# Usage:
#   ./query-balance.sh <address>
#   ./query-balance.sh  # Uses PAW_ADDRESS from env
#
# Requirements:
#   - curl
#   - jq
###############################################################################

set -e

# Configuration
RPC_ENDPOINT="${PAW_RPC_ENDPOINT:-http://localhost:26657}"
REST_ENDPOINT="${PAW_REST_ENDPOINT:-http://localhost:1317}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check requirements
for cmd in curl jq; do
    command -v $cmd >/dev/null 2>&1 || {
        echo -e "${RED}✗ Error: $cmd is required${NC}" >&2
        exit 1
    }
done

# Get address
ADDRESS="${1:-$PAW_ADDRESS}"

if [ -z "$ADDRESS" ]; then
    echo -e "${RED}Error: No address provided${NC}"
    echo "Usage: ./query-balance.sh <address>"
    echo "   or: Set PAW_ADDRESS in environment"
    exit 1
fi

echo "Querying Account Balance..."
echo ""
echo "Address: $ADDRESS"
echo "REST API: $REST_ENDPOINT"
echo ""

# Query all balances
BALANCES=$(curl -s "$REST_ENDPOINT/cosmos/bank/v1beta1/balances/$ADDRESS")

# Check if account exists
if echo "$BALANCES" | jq -e '.code' >/dev/null 2>&1; then
    ERROR_MSG=$(echo "$BALANCES" | jq -r '.message')
    echo -e "${RED}✗ Error querying balance: $ERROR_MSG${NC}"
    exit 1
fi

# Get balance array
BALANCE_COUNT=$(echo "$BALANCES" | jq '.balances | length')

if [ "$BALANCE_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✓ Account exists but has no tokens${NC}"
    echo ""
    echo "To fund this account:"
    echo "  - Use the testnet faucet"
    echo "  - Request tokens from another address"
else
    echo -e "${GREEN}✓ Balances retrieved successfully:${NC}"
    echo ""

    # Display each balance
    i=0
    while [ $i -lt "$BALANCE_COUNT" ]; do
        DENOM=$(echo "$BALANCES" | jq -r ".balances[$i].denom")
        AMOUNT=$(echo "$BALANCES" | jq -r ".balances[$i].amount")

        # Format amount
        if [[ $DENOM == u* ]]; then
            BASE_DENOM=${DENOM:1}
            BASE_DENOM_UPPER=$(echo "$BASE_DENOM" | tr '[:lower:]' '[:upper:]')
            BASE_AMOUNT=$(echo "scale=6; $AMOUNT / 1000000" | bc)
            echo "$((i+1)). $BASE_AMOUNT $BASE_DENOM_UPPER ($AMOUNT $DENOM)"
        else
            echo "$((i+1)). $AMOUNT $DENOM"
        fi

        i=$((i+1))
    done

    echo ""
    echo "Total Balances:"
    echo "  Token Types: $BALANCE_COUNT"

    # Calculate total units
    TOTAL_AMOUNT=0
    i=0
    while [ $i -lt "$BALANCE_COUNT" ]; do
        AMOUNT=$(echo "$BALANCES" | jq -r ".balances[$i].amount")
        TOTAL_AMOUNT=$((TOTAL_AMOUNT + AMOUNT))
        i=$((i+1))
    done
    echo "  Total Units: $(printf "%'d" $TOTAL_AMOUNT)"
fi

# Query account info
echo ""
ACCOUNT=$(curl -s "$REST_ENDPOINT/cosmos/auth/v1beta1/accounts/$ADDRESS")

if echo "$ACCOUNT" | jq -e '.account' >/dev/null 2>&1; then
    ACC_NUM=$(echo "$ACCOUNT" | jq -r '.account.account_number // "N/A"')
    SEQUENCE=$(echo "$ACCOUNT" | jq -r '.account.sequence // "0"')
    ACC_TYPE=$(echo "$ACCOUNT" | jq -r '.account["@type"] // "base"')

    echo "Account Information:"
    echo "  Account Number: $ACC_NUM"
    echo "  Sequence: $SEQUENCE"
    echo "  Type: $ACC_TYPE"
else
    echo -e "${YELLOW}✓ Account not yet initialized (no transactions)${NC}"
fi

echo ""
echo -e "${GREEN}✓ Query completed successfully${NC}"
