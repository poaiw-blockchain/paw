#!/bin/bash

# PAW Blockchain - Testnet Setup Script
# Initializes a testnet environment for development and testing

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
CHAIN_ID="paw-testnet-1"
HOME_DIR="$HOME/.paw-testnet"
MONIKER="paw-testnet-validator"
KEYRING_BACKEND="test"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_pawd() {
    if ! command -v pawd &> /dev/null; then
        log_error "pawd binary not found. Please build the project first:"
        log_error "  make build"
        exit 1
    fi
    log_info "pawd binary found: $(which pawd)"
}

clean_previous() {
    if [ -d "$HOME_DIR" ]; then
        log_warn "Previous testnet data found at: $HOME_DIR"
        read -p "Delete and start fresh? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$HOME_DIR"
            log_info "Previous data removed"
        else
            log_error "Cannot proceed with existing data. Exiting."
            exit 1
        fi
    fi
}

init_chain() {
    log_step "Initializing chain..."

    pawd init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"
    log_info "Chain initialized with chain-id: $CHAIN_ID"
}

create_validator_key() {
    log_step "Creating validator key..."

    # Create validator operator key
    pawd keys add validator \
        --keyring-backend "$KEYRING_BACKEND" \
        --home "$HOME_DIR" 2>&1 | tee /tmp/validator-key.txt

    # Extract address
    VALIDATOR_ADDR=$(pawd keys show validator -a --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")
    log_info "Validator address: $VALIDATOR_ADDR"
}

create_test_accounts() {
    log_step "Creating test accounts..."

    # Create test user accounts
    for i in {1..3}; do
        pawd keys add "user$i" \
            --keyring-backend "$KEYRING_BACKEND" \
            --home "$HOME_DIR" \
            2>&1 | tee "/tmp/user${i}-key.txt"
    done

    log_info "Test accounts created: user1, user2, user3"
}

add_genesis_accounts() {
    log_step "Adding genesis accounts..."

    # Add validator to genesis with initial balance
    pawd genesis add-genesis-account validator 100000000000upaw \
        --keyring-backend "$KEYRING_BACKEND" \
        --home "$HOME_DIR"

    # Add test users to genesis
    for i in {1..3}; do
        pawd genesis add-genesis-account "user$i" 10000000000upaw \
            --keyring-backend "$KEYRING_BACKEND" \
            --home "$HOME_DIR"
    done

    log_info "Genesis accounts configured"
}

create_genesis_tx() {
    log_step "Creating genesis transaction..."

    pawd genesis gentx validator 50000000000upaw \
        --chain-id "$CHAIN_ID" \
        --moniker "$MONIKER" \
        --commission-rate "0.10" \
        --commission-max-rate "0.20" \
        --commission-max-change-rate "0.01" \
        --min-self-delegation "1" \
        --keyring-backend "$KEYRING_BACKEND" \
        --home "$HOME_DIR"

    log_info "Genesis transaction created"
}

collect_gentxs() {
    log_step "Collecting genesis transactions..."

    pawd genesis collect-gentxs --home "$HOME_DIR"
    log_info "Genesis transactions collected"
}

validate_genesis() {
    log_step "Validating genesis..."

    pawd genesis validate-genesis --home "$HOME_DIR"
    log_info "Genesis validated successfully"
}

configure_app() {
    log_step "Configuring application..."

    APP_TOML="$HOME_DIR/config/app.toml"

    # Enable API
    sed -i 's/enable = false/enable = true/g' "$APP_TOML"

    # Enable Swagger
    sed -i 's/swagger = false/swagger = true/g' "$APP_TOML"

    # Set minimum gas prices
    sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001upaw"/g' "$APP_TOML"

    # Enable Prometheus metrics
    sed -i 's/prometheus = false/prometheus = true/g' "$APP_TOML"

    # Configure pruning for testnet (less aggressive)
    sed -i 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
    sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100"/g' "$APP_TOML"
    sed -i 's/pruning-keep-every = "0"/pruning-keep-every = "500"/g' "$APP_TOML"
    sed -i 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"

    log_info "Application configured"
}

configure_tendermint() {
    log_step "Configuring Tendermint..."

    CONFIG_TOML="$HOME_DIR/config/config.toml"

    # Set shorter block time for testnet
    sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/g' "$CONFIG_TOML"

    # Enable unsafe features for testnet
    sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/g' "$CONFIG_TOML"

    # Enable indexing
    sed -i 's/index_all_keys = false/index_all_keys = true/g' "$CONFIG_TOML"

    log_info "Tendermint configured"
}

display_info() {
    log_step "Testnet setup complete!"

    echo ""
    echo "========================================"
    echo "PAW Testnet Configuration"
    echo "========================================"
    echo ""
    echo "Chain ID:       $CHAIN_ID"
    echo "Home Directory: $HOME_DIR"
    echo "Moniker:        $MONIKER"
    echo ""
    echo "Validator Address: $VALIDATOR_ADDR"
    echo ""
    echo "Test Accounts:"
    echo "  user1: $(pawd keys show user1 -a --keyring-backend $KEYRING_BACKEND --home $HOME_DIR)"
    echo "  user2: $(pawd keys show user2 -a --keyring-backend $KEYRING_BACKEND --home $HOME_DIR)"
    echo "  user3: $(pawd keys show user3 -a --keyring-backend $KEYRING_BACKEND --home $HOME_DIR)"
    echo ""
    echo "Start the testnet with:"
    echo "  pawd start --home $HOME_DIR"
    echo ""
    echo "Or in the background:"
    echo "  nohup pawd start --home $HOME_DIR > testnet.log 2>&1 &"
    echo ""
    echo "Check status:"
    echo "  pawd status --home $HOME_DIR"
    echo ""
    echo "Query validator:"
    echo "  pawd q staking validators --home $HOME_DIR"
    echo ""
    echo "IMPORTANT: Save the key information from /tmp/*-key.txt files"
    echo "========================================"
}

# Main execution
main() {
    echo "========================================"
    echo "PAW Blockchain - Testnet Setup"
    echo "========================================"
    echo ""

    check_pawd
    clean_previous
    init_chain
    create_validator_key
    create_test_accounts
    add_genesis_accounts
    create_genesis_tx
    collect_gentxs
    validate_genesis
    configure_app
    configure_tendermint
    display_info

    log_info "Testnet setup completed successfully!"
}

main "$@"
