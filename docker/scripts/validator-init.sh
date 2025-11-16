#!/bin/bash
# PAW Validator Initialization Script
# This script initializes a validator node with proper configuration

set -euo pipefail

# Configuration variables
CHAIN_ID="${CHAIN_ID:-paw-1}"
MONIKER="${MONIKER:-validator}"
HOME_DIR="${HOME:-/home/validator/.paw}"
KEYRING_BACKEND="${KEYRING_BACKEND:-file}"
MINIMUM_GAS_PRICES="${MINIMUM_GAS_PRICES:-0.001upaw}"

# Logging functions
log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check if node is already initialized
if [ -f "$HOME_DIR/config/genesis.json" ]; then
    log_info "Node already initialized at $HOME_DIR"
    exit 0
fi

log_info "Initializing PAW validator node..."
log_info "Chain ID: $CHAIN_ID"
log_info "Moniker: $MONIKER"
log_info "Home: $HOME_DIR"

# Initialize the node
pawd init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

# Configure app.toml
log_info "Configuring app.toml..."
sed -i "s/minimum-gas-prices = \"\"/minimum-gas-prices = \"$MINIMUM_GAS_PRICES\"/" "$HOME_DIR/config/app.toml"

# Enable API and gRPC
sed -i 's/enable = false/enable = true/g' "$HOME_DIR/config/app.toml"

# Configure Prometheus metrics
sed -i 's/prometheus = false/prometheus = true/g' "$HOME_DIR/config/config.toml"

# Configure pruning (keep 100 recent blocks, prune every 10 blocks)
sed -i 's/pruning = "default"/pruning = "custom"/' "$HOME_DIR/config/app.toml"
sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100"/' "$HOME_DIR/config/app.toml"
sed -i 's/pruning-interval = "0"/pruning-interval = "10"/' "$HOME_DIR/config/app.toml"

# Configure state sync
sed -i 's/snapshot-interval = 0/snapshot-interval = 1000/' "$HOME_DIR/config/app.toml"
sed -i 's/snapshot-keep-recent = 2/snapshot-keep-recent = 5/' "$HOME_DIR/config/app.toml"

# Configure logging
sed -i 's/log_level = "info"/log_level = "info"/' "$HOME_DIR/config/config.toml"
sed -i 's/log_format = "plain"/log_format = "json"/' "$HOME_DIR/config/config.toml"

log_info "Node initialized successfully"
log_info "Configuration files created in $HOME_DIR/config/"
log_info "Genesis file: $HOME_DIR/config/genesis.json"
log_info "Node key: $HOME_DIR/config/node_key.json"
log_info "Private validator key: $HOME_DIR/config/priv_validator_key.json"

log_warn "IMPORTANT: Backup the following files securely:"
log_warn "  - $HOME_DIR/config/priv_validator_key.json"
log_warn "  - $HOME_DIR/config/node_key.json"
log_warn "  - $HOME_DIR/data/priv_validator_state.json"

exit 0
