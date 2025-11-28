#!/bin/bash
# PAW Mainnet Genesis Verification Script
# This script validates the genesis file before network launch

set -e

GENESIS_FILE="${1:-config/genesis-mainnet.json}"
EXPECTED_CHAIN_ID="paw-mainnet-1"
EXPECTED_SUPPLY="50000000000000"

echo "========================================="
echo "PAW Mainnet Genesis Verification"
echo "========================================="
echo ""

# Check if genesis file exists
if [ ! -f "$GENESIS_FILE" ]; then
    echo "❌ Error: Genesis file not found at $GENESIS_FILE"
    exit 1
fi

echo "✓ Genesis file found: $GENESIS_FILE"
echo ""

# Validate JSON syntax
echo "Checking JSON syntax..."
if command -v jq &> /dev/null; then
    if jq empty "$GENESIS_FILE" 2>/dev/null; then
        echo "✓ Valid JSON syntax"
    else
        echo "❌ Invalid JSON syntax"
        exit 1
    fi
elif command -v node &> /dev/null; then
    if node -e "JSON.parse(require('fs').readFileSync('$GENESIS_FILE', 'utf8'))" 2>/dev/null; then
        echo "✓ Valid JSON syntax"
    else
        echo "❌ Invalid JSON syntax"
        exit 1
    fi
else
    echo "⚠ Warning: Cannot validate JSON (install jq or node.js)"
fi
echo ""

# Extract and verify chain ID
echo "Verifying chain configuration..."
if command -v jq &> /dev/null; then
    CHAIN_ID=$(jq -r '.chain_id' "$GENESIS_FILE")
    GENESIS_TIME=$(jq -r '.genesis_time' "$GENESIS_FILE")
    TOTAL_SUPPLY=$(jq -r '.app_state.bank.supply[0].amount' "$GENESIS_FILE")

    echo "Chain ID: $CHAIN_ID"
    if [ "$CHAIN_ID" = "$EXPECTED_CHAIN_ID" ]; then
        echo "✓ Chain ID matches expected: $EXPECTED_CHAIN_ID"
    else
        echo "❌ Chain ID mismatch! Expected: $EXPECTED_CHAIN_ID, Got: $CHAIN_ID"
        exit 1
    fi

    echo ""
    echo "Genesis Time: $GENESIS_TIME"

    echo ""
    echo "Total Supply: $TOTAL_SUPPLY upaw ($((TOTAL_SUPPLY / 1000000)) PAW)"
    if [ "$TOTAL_SUPPLY" = "$EXPECTED_SUPPLY" ]; then
        echo "✓ Total supply matches expected: $EXPECTED_SUPPLY upaw"
    else
        echo "❌ Total supply mismatch! Expected: $EXPECTED_SUPPLY, Got: $TOTAL_SUPPLY"
        exit 1
    fi
else
    echo "⚠ Warning: jq not installed, skipping detailed verification"
fi
echo ""

# Compute genesis hash
echo "Computing genesis hash..."
if command -v sha256sum &> /dev/null; then
    HASH=$(sha256sum "$GENESIS_FILE" | awk '{print $1}')
    echo "Genesis SHA256: $HASH"
elif command -v shasum &> /dev/null; then
    HASH=$(shasum -a 256 "$GENESIS_FILE" | awk '{print $1}')
    echo "Genesis SHA256: $HASH"
else
    echo "⚠ Warning: Cannot compute hash (install sha256sum or shasum)"
fi
echo ""

# Verify token distribution if jq is available
if command -v jq &> /dev/null; then
    echo "Verifying token distribution..."

    BALANCES=$(jq -r '.app_state.bank.balances[] | "\(.address):\(.coins[0].amount)"' "$GENESIS_FILE")

    TOTAL=0
    while IFS=: read -r ADDRESS AMOUNT; do
        TOTAL=$((TOTAL + AMOUNT))
        PAW_AMOUNT=$((AMOUNT / 1000000))
        ADDR_SHORT="${ADDRESS:0:30}..."
        printf "  %-35s %15s PAW\n" "$ADDR_SHORT" "$(echo $PAW_AMOUNT | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
    done <<< "$BALANCES"

    echo ""
    echo "Total Allocated: $((TOTAL / 1000000)) PAW"

    if [ "$TOTAL" = "$TOTAL_SUPPLY" ]; then
        echo "✓ Balances sum matches total supply"
    else
        echo "❌ Balance sum mismatch! Sum: $TOTAL, Supply: $TOTAL_SUPPLY"
        exit 1
    fi
fi
echo ""

# Verify consensus parameters
if command -v jq &> /dev/null; then
    echo "Consensus Parameters:"
    MAX_BLOCK_SIZE=$(jq -r '.consensus_params.block.max_bytes' "$GENESIS_FILE")
    MAX_GAS=$(jq -r '.consensus_params.block.max_gas' "$GENESIS_FILE")
    MAX_VALIDATORS=$(jq -r '.app_state.staking.params.max_validators' "$GENESIS_FILE")
    UNBONDING_TIME=$(jq -r '.app_state.staking.params.unbonding_time' "$GENESIS_FILE")

    echo "  Max Block Size: $((MAX_BLOCK_SIZE / 1024 / 1024)) MB"
    echo "  Max Gas: $MAX_GAS"
    echo "  Max Validators: $MAX_VALIDATORS"
    echo "  Unbonding Time: $UNBONDING_TIME"
    echo ""
fi

# Verify custom modules
if command -v jq &> /dev/null; then
    echo "Custom Modules:"

    # DEX Module
    SWAP_FEE=$(jq -r '.app_state.dex.params.swap_fee' "$GENESIS_FILE")
    DEX_POOLS=$(jq -r '.app_state.dex.pools | length' "$GENESIS_FILE")
    echo "  DEX: Swap Fee=$SWAP_FEE, Initial Pools=$DEX_POOLS"

    # Oracle Module
    MIN_VALIDATORS=$(jq -r '.app_state.oracle.params.min_validators' "$GENESIS_FILE")
    PRICE_FEEDS=$(jq -r '.app_state.oracle.price_feeds | length' "$GENESIS_FILE")
    echo "  Oracle: Min Validators=$MIN_VALIDATORS, Price Feeds=$PRICE_FEEDS"

    # Compute Module
    MIN_STAKE=$(jq -r '.app_state.compute.params.min_stake' "$GENESIS_FILE")
    echo "  Compute: Min Stake=$((MIN_STAKE / 1000000)) PAW"
    echo ""
fi

echo "========================================="
echo "✅ Genesis file verification complete!"
echo "========================================="
echo ""
echo "Next Steps:"
echo "1. Coordinate with other validators to verify hash"
echo "2. Copy genesis to node: cp $GENESIS_FILE ~/.paw/config/genesis.json"
echo "3. If genesis validator, create and submit gentx"
echo "4. Wait for all gentxs to be collected"
echo "5. Start node at genesis_time"
echo ""

exit 0
