#!/bin/bash
#
# PAW Blockchain - Interactive Validator Registration Script
#
# This script guides you through creating a validator on PAW testnet/mainnet.
# It performs validation checks and creates the validator transaction.
#
# Usage:
#   ./register-validator.sh
#
# Prerequisites:
#   - pawd binary installed and in PATH
#   - Node fully synced
#   - Operator key created and funded
#   - priv_validator_key.json present (consensus key)
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
    exit 1
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

info() {
    echo -e "$1"
}

prompt() {
    read -p "$1: " response
    echo "$response"
}

# Check prerequisites
check_prerequisites() {
    info "=== Checking Prerequisites ==="

    # Check pawd binary
    if ! command -v pawd &> /dev/null; then
        error "pawd binary not found. Please install PAW blockchain."
    fi
    success "pawd binary found"

    # Check node is running and synced
    if ! pawd status &> /dev/null; then
        error "Cannot connect to pawd node. Is it running?"
    fi
    success "Connected to pawd node"

    # Check sync status
    CATCHING_UP=$(pawd status | jq -r '.SyncInfo.catching_up')
    if [ "$CATCHING_UP" = "true" ]; then
        error "Node is still syncing. Wait until catching_up = false"
    fi
    success "Node is fully synced"

    echo ""
}

# Get chain ID
get_chain_id() {
    CHAIN_ID=$(pawd status | jq -r '.NodeInfo.network')
    info "Chain ID: $CHAIN_ID"

    if [[ "$CHAIN_ID" == *"testnet"* ]]; then
        warning "Detected TESTNET. Use testnet tokens only."
    elif [[ "$CHAIN_ID" == *"mainnet"* ]]; then
        warning "Detected MAINNET. Real tokens at risk!"
    fi

    echo ""
}

# Interactive configuration
configure_validator() {
    info "=== Validator Configuration ==="

    # Moniker
    MONIKER=$(prompt "Enter validator moniker (public name)")
    if [ -z "$MONIKER" ]; then
        error "Moniker cannot be empty"
    fi

    # Operator key
    info ""
    info "Available keys:"
    pawd keys list --keyring-backend os
    echo ""

    OPERATOR_KEY=$(prompt "Enter operator key name (from list above)")
    if ! pawd keys show "$OPERATOR_KEY" --keyring-backend os &> /dev/null; then
        error "Operator key '$OPERATOR_KEY' not found"
    fi

    OPERATOR_ADDR=$(pawd keys show "$OPERATOR_KEY" -a --keyring-backend os)
    success "Operator address: $OPERATOR_ADDR"

    # Check balance
    BALANCE=$(pawd query bank balances "$OPERATOR_ADDR" --output json | jq -r '.balances[] | select(.denom=="upaw") | .amount')
    if [ -z "$BALANCE" ] || [ "$BALANCE" -lt 1000000 ]; then
        error "Insufficient balance. Need at least 1,000,000 upaw. Current: $BALANCE upaw"
    fi
    success "Balance: $BALANCE upaw"

    # Self-delegation amount
    echo ""
    MAX_DELEGATION=$((BALANCE - 10000))  # Reserve 10000 for fees
    info "Available for delegation: $MAX_DELEGATION upaw"
    SELF_DELEGATION=$(prompt "Enter self-delegation amount (upaw) [minimum 1000000]")

    if [ "$SELF_DELEGATION" -lt 1000000 ]; then
        error "Self-delegation must be at least 1,000,000 upaw"
    fi

    if [ "$SELF_DELEGATION" -gt "$MAX_DELEGATION" ]; then
        error "Self-delegation exceeds available balance (accounting for fees)"
    fi

    # Commission rates
    echo ""
    info "Commission Configuration:"
    info "  - commission-rate: Current rate (e.g., 0.10 = 10%)"
    info "  - commission-max-rate: Maximum you can ever charge"
    info "  - commission-max-change-rate: Max daily adjustment"

    COMMISSION_RATE=$(prompt "Enter commission rate [0.00-1.00, recommended: 0.10]")
    COMMISSION_MAX_RATE=$(prompt "Enter max commission rate [0.00-1.00, recommended: 0.20]")
    COMMISSION_MAX_CHANGE=$(prompt "Enter max commission change rate [0.00-1.00, recommended: 0.01]")

    # Validate commission
    if (( $(echo "$COMMISSION_RATE > $COMMISSION_MAX_RATE" | bc -l) )); then
        error "Commission rate cannot exceed max commission rate"
    fi

    # Minimum self-delegation
    MIN_SELF_DELEGATION=$(prompt "Enter minimum self-delegation (upaw) [recommended: same as initial]")

    if [ "$MIN_SELF_DELEGATION" -gt "$SELF_DELEGATION" ]; then
        error "Minimum self-delegation cannot exceed initial self-delegation"
    fi

    # Optional metadata
    echo ""
    info "Optional Metadata (press Enter to skip):"
    IDENTITY=$(prompt "Keybase.io identity (16-char PGP key)")
    WEBSITE=$(prompt "Website URL")
    SECURITY_CONTACT=$(prompt "Security contact email")
    DETAILS=$(prompt "Validator description")

    # Get consensus pubkey
    CONSENSUS_PUBKEY=$(pawd tendermint show-validator)

    echo ""
}

# Display configuration summary
show_summary() {
    info "=== Validator Configuration Summary ==="
    info "Chain ID: $CHAIN_ID"
    info "Moniker: $MONIKER"
    info "Operator: $OPERATOR_KEY ($OPERATOR_ADDR)"
    info "Self-Delegation: $SELF_DELEGATION upaw"
    info "Min Self-Delegation: $MIN_SELF_DELEGATION upaw"
    info "Commission Rate: $COMMISSION_RATE"
    info "Max Commission: $COMMISSION_MAX_RATE"
    info "Max Change Rate: $COMMISSION_MAX_CHANGE"
    info "Identity: ${IDENTITY:-<none>}"
    info "Website: ${WEBSITE:-<none>}"
    info "Security Contact: ${SECURITY_CONTACT:-<none>}"
    info "Details: ${DETAILS:-<none>}"
    info "Consensus Pubkey: $CONSENSUS_PUBKEY"
    echo ""

    read -p "Proceed with validator creation? (yes/no): " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        info "Aborted."
        exit 0
    fi
}

# Create validator transaction
create_validator() {
    info "=== Creating Validator ==="

    # Build command
    CMD="pawd tx staking create-validator \
      --from $OPERATOR_KEY \
      --amount ${SELF_DELEGATION}upaw \
      --pubkey \"$CONSENSUS_PUBKEY\" \
      --moniker \"$MONIKER\" \
      --chain-id $CHAIN_ID \
      --commission-rate $COMMISSION_RATE \
      --commission-max-rate $COMMISSION_MAX_RATE \
      --commission-max-change-rate $COMMISSION_MAX_CHANGE \
      --min-self-delegation $MIN_SELF_DELEGATION \
      --gas auto \
      --gas-adjustment 1.5 \
      --gas-prices 0.001upaw \
      --keyring-backend os \
      --broadcast-mode block"

    # Add optional fields
    [ -n "$IDENTITY" ] && CMD="$CMD --identity \"$IDENTITY\""
    [ -n "$WEBSITE" ] && CMD="$CMD --website \"$WEBSITE\""
    [ -n "$SECURITY_CONTACT" ] && CMD="$CMD --security-contact \"$SECURITY_CONTACT\""
    [ -n "$DETAILS" ] && CMD="$CMD --details \"$DETAILS\""

    # Execute transaction
    info "Submitting transaction..."
    eval $CMD

    if [ $? -eq 0 ]; then
        success "Validator created successfully!"
    else
        error "Validator creation failed. Check error message above."
    fi

    echo ""
}

# Verify validator
verify_validator() {
    info "=== Verifying Validator ==="

    sleep 5  # Wait for transaction to be processed

    VALOPER_ADDR=$(pawd keys show "$OPERATOR_KEY" --bech val -a --keyring-backend os)

    # Query validator
    if pawd query staking validator "$VALOPER_ADDR" &> /dev/null; then
        success "Validator found in validator set"

        info ""
        info "Validator Details:"
        pawd query staking validator "$VALOPER_ADDR" | jq '.'

        info ""
        info "Validator operator address: $VALOPER_ADDR"
        info "Monitor validator at: https://explorer.paw-testnet.io/validators/$VALOPER_ADDR"
    else
        warning "Validator not found yet. May take a few blocks to appear."
    fi

    echo ""
}

# Next steps
show_next_steps() {
    info "=== Next Steps ==="
    info "1. Monitor validator signing:"
    info "   pawd query slashing signing-info \$(pawd tendermint show-validator)"
    info ""
    info "2. Check if validator is bonded:"
    info "   pawd query staking validator $VALOPER_ADDR | jq '.status'"
    info ""
    info "3. Setup monitoring: docs/OBSERVABILITY.md and docs/DASHBOARDS_GUIDE.md"
    info ""
    info "4. Harden security: docs/VALIDATOR_KEY_MANAGEMENT.md and docs/SENTRY_ARCHITECTURE.md"
    info ""
    info "5. Join Discord: https://discord.gg/paw-blockchain"
    info ""
    success "Validator registration complete!"
}

# Main execution
main() {
    echo ""
    info "╔════════════════════════════════════════════════════════════╗"
    info "║     PAW Blockchain - Validator Registration Script        ║"
    info "╚════════════════════════════════════════════════════════════╝"
    echo ""

    check_prerequisites
    get_chain_id
    configure_validator
    show_summary
    create_validator
    verify_validator
    show_next_steps
}

main
