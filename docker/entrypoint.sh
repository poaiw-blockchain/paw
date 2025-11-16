#!/bin/bash
set -e

# PAW Node Entrypoint Script
# Handles initialization and startup of PAW blockchain node

CHAIN_ID="${CHAIN_ID:-paw-1}"
MONIKER="${MONIKER:-paw-node}"
HOME_DIR="${HOME:-.paw}"
CONFIG_DIR="$HOME_DIR/config"
DATA_DIR="$HOME_DIR/data"

# Color output
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

# Function to check if node is initialized
is_initialized() {
    if [ -f "$CONFIG_DIR/genesis.json" ] && [ -f "$CONFIG_DIR/config.toml" ]; then
        return 0
    fi
    return 1
}

# Function to initialize node
init_node() {
    log_info "Initializing PAW node with chain-id: $CHAIN_ID, moniker: $MONIKER"

    pawd init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

    if [ $? -ne 0 ]; then
        log_error "Failed to initialize node"
        exit 1
    fi

    log_info "Node initialized successfully"
}

# Function to configure node
configure_node() {
    log_info "Configuring node..."

    # Update config.toml
    if [ -n "$PERSISTENT_PEERS" ]; then
        log_info "Setting persistent peers: $PERSISTENT_PEERS"
        sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PERSISTENT_PEERS\"/g" "$CONFIG_DIR/config.toml"
    fi

    if [ -n "$SEEDS" ]; then
        log_info "Setting seeds: $SEEDS"
        sed -i "s/seeds = \"\"/seeds = \"$SEEDS\"/g" "$CONFIG_DIR/config.toml"
    fi

    # Enable prometheus metrics
    sed -i 's/prometheus = false/prometheus = true/g' "$CONFIG_DIR/config.toml"

    # Configure RPC server to listen on all interfaces
    sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' "$CONFIG_DIR/config.toml"

    # Enable API server
    sed -i 's/enable = false/enable = true/g' "$CONFIG_DIR/app.toml"
    sed -i 's/swagger = false/swagger = true/g' "$CONFIG_DIR/app.toml"

    # Configure API to listen on all interfaces
    sed -i 's/address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' "$CONFIG_DIR/app.toml"

    # Configure gRPC to listen on all interfaces
    sed -i 's/address = "localhost:9090"/address = "0.0.0.0:9090"/g' "$CONFIG_DIR/app.toml"

    # Set minimum gas prices if provided
    if [ -n "$MINIMUM_GAS_PRICES" ]; then
        log_info "Setting minimum gas prices: $MINIMUM_GAS_PRICES"
        sed -i "s/minimum-gas-prices = \"\"/minimum-gas-prices = \"$MINIMUM_GAS_PRICES\"/g" "$CONFIG_DIR/app.toml"
    else
        # Default minimum gas prices
        sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.025upaw"/g' "$CONFIG_DIR/app.toml"
    fi

    # Configure pruning
    PRUNING="${PRUNING:-default}"
    log_info "Setting pruning strategy: $PRUNING"
    sed -i "s/pruning = \"default\"/pruning = \"$PRUNING\"/g" "$CONFIG_DIR/app.toml"

    # Configure state sync if enabled
    if [ "$STATE_SYNC_ENABLED" = "true" ]; then
        log_info "Enabling state sync..."
        sed -i 's/enable = false/enable = true/g' "$CONFIG_DIR/config.toml"

        if [ -n "$STATE_SYNC_RPC_SERVERS" ]; then
            sed -i "s/rpc_servers = \"\"/rpc_servers = \"$STATE_SYNC_RPC_SERVERS\"/g" "$CONFIG_DIR/config.toml"
        fi

        if [ -n "$STATE_SYNC_TRUST_HEIGHT" ] && [ -n "$STATE_SYNC_TRUST_HASH" ]; then
            sed -i "s/trust_height = 0/trust_height = $STATE_SYNC_TRUST_HEIGHT/g" "$CONFIG_DIR/config.toml"
            sed -i "s/trust_hash = \"\"/trust_hash = \"$STATE_SYNC_TRUST_HASH\"/g" "$CONFIG_DIR/config.toml"
        fi
    fi

    log_info "Node configuration completed"
}

# Function to download genesis file
download_genesis() {
    if [ -n "$GENESIS_URL" ]; then
        log_info "Downloading genesis file from $GENESIS_URL"
        curl -L "$GENESIS_URL" -o "$CONFIG_DIR/genesis.json"

        if [ $? -ne 0 ]; then
            log_error "Failed to download genesis file"
            exit 1
        fi

        log_info "Genesis file downloaded successfully"
    fi
}

# Function to validate genesis
validate_genesis() {
    log_info "Validating genesis file..."

    pawd validate-genesis --home "$HOME_DIR"

    if [ $? -ne 0 ]; then
        log_error "Genesis validation failed"
        exit 1
    fi

    log_info "Genesis file is valid"
}

# Function to create validator
create_validator() {
    if [ "$CREATE_VALIDATOR" = "true" ]; then
        log_warn "Validator creation requested. Make sure the node is synced first!"

        # Wait for user confirmation in manual mode
        if [ "$AUTO_CREATE_VALIDATOR" != "true" ]; then
            log_warn "Set AUTO_CREATE_VALIDATOR=true to automatically create validator"
            return
        fi

        if [ -z "$VALIDATOR_KEYNAME" ]; then
            log_error "VALIDATOR_KEYNAME not set"
            return
        fi

        # Note: This is a placeholder. Actual validator creation requires
        # the node to be synced and tokens to be available
        log_info "Validator creation command (run manually when synced):"
        echo "pawd tx staking create-validator \\"
        echo "  --amount=1000000upaw \\"
        echo "  --pubkey=\$(pawd tendermint show-validator) \\"
        echo "  --moniker=\"$MONIKER\" \\"
        echo "  --chain-id=\"$CHAIN_ID\" \\"
        echo "  --commission-rate=\"0.10\" \\"
        echo "  --commission-max-rate=\"0.20\" \\"
        echo "  --commission-max-change-rate=\"0.01\" \\"
        echo "  --min-self-delegation=\"1\" \\"
        echo "  --gas=\"auto\" \\"
        echo "  --gas-adjustment=\"1.5\" \\"
        echo "  --gas-prices=\"0.025upaw\" \\"
        echo "  --from=\"$VALIDATOR_KEYNAME\""
    fi
}

# Function to wait for sync
wait_for_sync() {
    if [ "$WAIT_FOR_SYNC" = "true" ]; then
        log_info "Waiting for node to sync..."

        while true; do
            SYNC_STATUS=$(curl -s http://localhost:26657/status | jq -r .result.sync_info.catching_up)

            if [ "$SYNC_STATUS" = "false" ]; then
                log_info "Node is synced!"
                break
            fi

            log_info "Node is syncing... (catching_up: $SYNC_STATUS)"
            sleep 30
        done
    fi
}

# Main execution
main() {
    log_info "Starting PAW node..."
    log_info "Chain ID: $CHAIN_ID"
    log_info "Moniker: $MONIKER"
    log_info "Home directory: $HOME_DIR"

    # Check if already initialized
    if ! is_initialized; then
        log_info "Node not initialized. Initializing..."
        init_node
        configure_node
        download_genesis

        # Validate genesis if it exists
        if [ -f "$CONFIG_DIR/genesis.json" ]; then
            validate_genesis
        fi
    else
        log_info "Node already initialized. Skipping initialization."

        # Still apply configuration updates
        configure_node
    fi

    # Handle different commands
    case "$1" in
        start)
            log_info "Starting PAW blockchain node..."
            exec pawd start --home "$HOME_DIR" ${@:2}
            ;;

        init)
            log_info "Node initialized. Exiting."
            exit 0
            ;;

        validate-genesis)
            validate_genesis
            exit 0
            ;;

        create-validator)
            create_validator
            exit 0
            ;;

        version)
            pawd version
            exit 0
            ;;

        *)
            # Pass through any other commands to pawd
            exec pawd "$@" --home "$HOME_DIR"
            ;;
    esac
}

# Run main function
main "$@"
