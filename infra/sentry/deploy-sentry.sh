#!/bin/bash
# Deploy PAW Testnet Sentry Node
# Run this script ON the services-testnet server (139.99.149.160)
#
# Prerequisites:
#   - pawd binary available at ~/.paw/cosmovisor/genesis/bin/pawd
#   - Genesis file from existing validator
#   - WireGuard VPN configured (wg0 interface with 10.10.0.4)

set -euo pipefail

SENTRY_HOME="${SENTRY_HOME:-$HOME/.paw-sentry}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHAIN_ID="paw-testnet-1"

# Port configuration for sentry
P2P_PORT=12056
RPC_PORT=12057
GRPC_PORT=12090
REST_PORT=12017

echo "=== PAW Testnet Sentry Node Deployment ==="
echo "Sentry Home: $SENTRY_HOME"
echo "Chain ID: $CHAIN_ID"
echo ""

# Check prerequisites
if [[ ! -f "$HOME/.paw/cosmovisor/genesis/bin/pawd" ]]; then
    echo "ERROR: pawd binary not found at ~/.paw/cosmovisor/genesis/bin/pawd"
    exit 1
fi

PAWD="$HOME/.paw/cosmovisor/genesis/bin/pawd"

# Check if sentry already exists
if [[ -d "$SENTRY_HOME" ]]; then
    echo "WARNING: Sentry home directory already exists at $SENTRY_HOME"
    read -p "Do you want to remove it and start fresh? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$SENTRY_HOME"
    else
        echo "Aborting. Remove $SENTRY_HOME manually if needed."
        exit 1
    fi
fi

echo "Step 1: Initializing sentry node..."
$PAWD init paw-sentry-1 --chain-id "$CHAIN_ID" --home "$SENTRY_HOME"

echo "Step 2: Copying genesis from validator..."
# Try val3 first (local), then val4
if [[ -f "$HOME/.paw-val3/config/genesis.json" ]]; then
    cp "$HOME/.paw-val3/config/genesis.json" "$SENTRY_HOME/config/genesis.json"
    echo "  Copied genesis from val3"
elif [[ -f "$HOME/.paw-val4/config/genesis.json" ]]; then
    cp "$HOME/.paw-val4/config/genesis.json" "$SENTRY_HOME/config/genesis.json"
    echo "  Copied genesis from val4"
else
    echo "ERROR: No genesis file found. Copy from a validator manually."
    exit 1
fi

echo "Step 3: Installing sentry configuration..."
if [[ -f "$SCRIPT_DIR/config-sentry.toml" ]]; then
    cp "$SCRIPT_DIR/config-sentry.toml" "$SENTRY_HOME/config/config.toml"
    echo "  Installed config.toml"
else
    echo "WARNING: config-sentry.toml not found, using default config"
fi

if [[ -f "$SCRIPT_DIR/app-sentry.toml" ]]; then
    cp "$SCRIPT_DIR/app-sentry.toml" "$SENTRY_HOME/config/app.toml"
    echo "  Installed app.toml"
else
    echo "WARNING: app-sentry.toml not found, using default config"
fi

echo "Step 4: Getting sentry node ID..."
SENTRY_NODE_ID=$($PAWD tendermint show-node-id --home "$SENTRY_HOME")
echo "  Sentry Node ID: $SENTRY_NODE_ID"

echo "Step 5: Installing systemd service..."
if [[ -f "$SCRIPT_DIR/pawd-sentry.service" ]]; then
    sudo cp "$SCRIPT_DIR/pawd-sentry.service" /etc/systemd/system/
    sudo systemctl daemon-reload
    echo "  Installed pawd-sentry.service"
else
    echo "WARNING: pawd-sentry.service not found, skipping systemd setup"
fi

echo ""
echo "=== Sentry Node Deployment Complete ==="
echo ""
echo "Sentry Node ID: $SENTRY_NODE_ID"
echo "P2P Address:    139.99.149.160:$P2P_PORT"
echo "RPC Address:    127.0.0.1:$RPC_PORT"
echo "gRPC Address:   0.0.0.0:$GRPC_PORT"
echo "REST Address:   127.0.0.1:$REST_PORT"
echo ""
echo "Next steps:"
echo "  1. Start the sentry:  sudo systemctl enable --now pawd-sentry"
echo "  2. Check logs:        sudo journalctl -u pawd-sentry -f"
echo "  3. Wait for sync:     curl -s http://127.0.0.1:$RPC_PORT/status | jq .result.sync_info"
echo ""
echo "To add this sentry to nginx (optional):"
echo "  upstream paw_sentry {"
echo "      server 127.0.0.1:$RPC_PORT;"
echo "  }"
echo ""
echo "Sentry peer string for external nodes:"
echo "  ${SENTRY_NODE_ID}@139.99.149.160:$P2P_PORT"
