#!/bin/bash
# init-genesis.sh
# Initialize PAW blockchain genesis state with validators, accounts, and parameters

set -euo pipefail

# Configuration
CHAIN_ID="paw-testnet"
CHAIN_DIR="${HOME}/.paw"
CONFIG_DIR="${CHAIN_DIR}/config"
DATA_DIR="${CHAIN_DIR}/data"
GENESIS_FILE="${CONFIG_DIR}/genesis.json"
NODE_CONFIG="infra/node-config.yaml"

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

# Check if pawd binary exists
if ! command -v pawd &> /dev/null; then
    log_error "pawd binary not found. Please build it first: make install"
    exit 1
fi

# Check if node-config.yaml exists
if [ ! -f "$NODE_CONFIG" ]; then
    log_error "Configuration file not found: $NODE_CONFIG"
    exit 1
fi

log_info "Initializing PAW genesis state..."
log_info "Chain ID: $CHAIN_ID"
log_info "Chain directory: $CHAIN_DIR"

# Clean up existing data if present
if [ -d "$CHAIN_DIR" ]; then
    log_warn "Existing chain data found at $CHAIN_DIR"
    read -p "Do you want to remove it? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CHAIN_DIR"
        log_info "Removed existing chain data"
    else
        log_error "Aborted. Please backup or remove existing data manually."
        exit 1
    fi
fi

# Initialize chain data directory
log_info "Initializing chain data directory..."
pawd init "paw-controller" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"

# Parse node-config.yaml for parameters
log_info "Reading configuration from $NODE_CONFIG..."

# Extract parameters using yq or grep/sed
TOTAL_SUPPLY=$(grep "total_supply:" "$NODE_CONFIG" | awk '{print $2}')
RESERVE=$(grep "reserve:" "$NODE_CONFIG" | awk '{print $2}')
MAX_VALIDATORS=$(grep "max_validators:" "$NODE_CONFIG" | awk '{print $2}')
GENESIS_VALIDATORS=$(grep "genesis_validators:" "$NODE_CONFIG" | awk '{print $2}')
BLOCK_TIME=$(grep "block_time_seconds:" "$NODE_CONFIG" | awk '{print $2}')
MIN_GAS_PRICE=$(grep "min_gas_price:" "$NODE_CONFIG" | awk '{print $2}')
MIN_VALIDATOR_STAKE=$(grep "min_validator_stake:" "$NODE_CONFIG" | awk '{print $2}')
UNBONDING_PERIOD=$(grep "unbonding_period_seconds:" "$NODE_CONFIG" | awk '{print $2}')

log_info "Total supply: ${TOTAL_SUPPLY} PAW"
log_info "Reserve: ${RESERVE} PAW"
log_info "Max validators: ${MAX_VALIDATORS}"
log_info "Genesis validators: ${GENESIS_VALIDATORS}"

# Generate validator keys
log_info "Generating validator keys..."
VALIDATOR_KEYS=()

for i in $(seq 1 2); do
    VALIDATOR_NAME="validator-${i}"
    log_info "Generating keys for ${VALIDATOR_NAME}..."

    # Generate validator operator key
    pawd keys add "${VALIDATOR_NAME}" --keyring-backend test --home "$CHAIN_DIR" 2>&1 | tee "/tmp/${VALIDATOR_NAME}.key"

    # Extract address
    VALIDATOR_ADDR=$(pawd keys show "${VALIDATOR_NAME}" -a --keyring-backend test --home "$CHAIN_DIR")
    VALIDATOR_KEYS+=("${VALIDATOR_NAME}:${VALIDATOR_ADDR}")

    log_info "${VALIDATOR_NAME} address: ${VALIDATOR_ADDR}"
done

# Generate genesis accounts
log_info "Creating genesis accounts..."

# Treasury account
TREASURY_NAME="treasury"
pawd keys add "$TREASURY_NAME" --keyring-backend test --home "$CHAIN_DIR"
TREASURY_ADDR=$(pawd keys show "$TREASURY_NAME" -a --keyring-backend test --home "$CHAIN_DIR")

# Foundation account
FOUNDATION_NAME="foundation"
pawd keys add "$FOUNDATION_NAME" --keyring-backend test --home "$CHAIN_DIR"
FOUNDATION_ADDR=$(pawd keys show "$FOUNDATION_NAME" -a --keyring-backend test --home "$CHAIN_DIR")

log_info "Treasury address: ${TREASURY_ADDR}"
log_info "Foundation address: ${FOUNDATION_ADDR}"

# Add genesis accounts with initial balances
log_info "Adding genesis accounts with initial balances..."

# Validator accounts (10M PAW each for staking)
for key_info in "${VALIDATOR_KEYS[@]}"; do
    ADDR=$(echo "$key_info" | cut -d':' -f2)
    pawd add-genesis-account "$ADDR" 10000000000000upaw --home "$CHAIN_DIR"
done

# Treasury (15M PAW - reserve)
pawd add-genesis-account "$TREASURY_ADDR" 15000000000000upaw --home "$CHAIN_DIR"

# Foundation (15M PAW - ecosystem development)
pawd add-genesis-account "$FOUNDATION_ADDR" 15000000000000upaw --home "$CHAIN_DIR"

# Configure chain parameters in genesis
log_info "Configuring chain parameters..."

# Update staking parameters
pawd genesis set-staking-param \
    --min-validator-stake "${MIN_VALIDATOR_STAKE}000000upaw" \
    --unbonding-time "${UNBONDING_PERIOD}s" \
    --max-validators "$MAX_VALIDATORS" \
    --home "$CHAIN_DIR"

# Update consensus parameters
pawd genesis set-consensus-param \
    --block-time "${BLOCK_TIME}s" \
    --timeout-propose "3000ms" \
    --timeout-prevote "1000ms" \
    --timeout-precommit "1000ms" \
    --timeout-commit "4000ms" \
    --max-block-size "2097152" \
    --max-gas "100000000" \
    --home "$CHAIN_DIR"

# Update slashing parameters
pawd genesis set-slashing-param \
    --double-sign-penalty "0.05" \
    --downtime-threshold "500" \
    --downtime-window "10000" \
    --downtime-penalty "0.001" \
    --downtime-jail-duration "86400s" \
    --home "$CHAIN_DIR"

# Update governance parameters
pawd genesis set-gov-param \
    --min-deposit "10000000000upaw" \
    --deposit-period "604800s" \
    --voting-period "1209600s" \
    --quorum "0.40" \
    --threshold "0.667" \
    --veto-threshold "0.333" \
    --home "$CHAIN_DIR"

# Update fee parameters
pawd genesis set-fee-param \
    --min-gas-price "${MIN_GAS_PRICE}upaw" \
    --burn-percentage "0.50" \
    --validator-percentage "0.30" \
    --treasury-percentage "0.20" \
    --home "$CHAIN_DIR"

# Generate gentx for validators
log_info "Generating genesis transactions for validators..."

for i in $(seq 1 2); do
    VALIDATOR_NAME="validator-${i}"
    STAKE_AMOUNT="${MIN_VALIDATOR_STAKE}000000upaw"

    log_info "Creating gentx for ${VALIDATOR_NAME} with stake: ${STAKE_AMOUNT}"

    pawd gentx "${VALIDATOR_NAME}" "$STAKE_AMOUNT" \
        --chain-id "$CHAIN_ID" \
        --moniker "${VALIDATOR_NAME}" \
        --commission-rate "0.10" \
        --commission-max-rate "0.20" \
        --commission-max-change-rate "0.01" \
        --min-self-delegation "1000000" \
        --keyring-backend test \
        --home "$CHAIN_DIR"
done

# Collect genesis transactions
log_info "Collecting genesis transactions..."
pawd collect-gentxs --home "$CHAIN_DIR"

# Validate genesis file
log_info "Validating genesis file..."
pawd validate-genesis --home "$CHAIN_DIR"

log_info "Genesis state initialization complete!"
log_info "Genesis file: $GENESIS_FILE"
log_info ""
log_info "Validator keys:"
for key_info in "${VALIDATOR_KEYS[@]}"; do
    echo "  - $key_info"
done
log_info ""
log_info "To start the node, run: pawd start --home $CHAIN_DIR"
