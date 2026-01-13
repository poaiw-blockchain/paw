#!/bin/bash
# Enable sentry mode on a PAW validator
# This reconfigures a validator to connect ONLY through sentry nodes
#
# Usage: ./enable-sentry-mode.sh <validator_number>
# Example: ./enable-sentry-mode.sh 1  (for val1)
#
# WARNING: This will restart the validator service!

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <validator_number>"
    echo "Example: $0 1  (for val1)"
    exit 1
fi

VAL_NUM="$1"
VAL_HOME="$HOME/.paw-val${VAL_NUM}"
VAL_SERVICE="pawd-val@${VAL_NUM}"

# Sentry node information (on services-testnet)
# These will be populated after sentry deployment
SENTRY_NODE_ID="${SENTRY_NODE_ID:-}"
SENTRY_IP="10.10.0.4"  # VPN IP for services-testnet
SENTRY_P2P_PORT="12056"

if [[ -z "$SENTRY_NODE_ID" ]]; then
    echo "ERROR: SENTRY_NODE_ID not set"
    echo "Set it first: export SENTRY_NODE_ID=<node_id_from_sentry>"
    echo ""
    echo "Get sentry node ID:"
    echo "  ssh services-testnet 'pawd tendermint show-node-id --home ~/.paw-sentry'"
    exit 1
fi

if [[ ! -d "$VAL_HOME" ]]; then
    echo "ERROR: Validator home not found: $VAL_HOME"
    exit 1
fi

CONFIG_FILE="$VAL_HOME/config/config.toml"
BACKUP_FILE="$VAL_HOME/config/config.toml.pre-sentry"

echo "=== Enabling Sentry Mode for val${VAL_NUM} ==="
echo "Validator Home: $VAL_HOME"
echo "Sentry Node ID: $SENTRY_NODE_ID"
echo "Sentry Address: ${SENTRY_IP}:${SENTRY_P2P_PORT}"
echo ""

# Backup existing config
echo "Step 1: Backing up current config..."
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "  Backup saved to: $BACKUP_FILE"

echo "Step 2: Updating config for sentry mode..."

# Update persistent_peers to ONLY connect to sentry
sed -i "s|^persistent_peers = .*|persistent_peers = \"${SENTRY_NODE_ID}@${SENTRY_IP}:${SENTRY_P2P_PORT}\"|" "$CONFIG_FILE"

# Disable peer exchange - validator should NOT discover peers
sed -i "s|^pex = true|pex = false|" "$CONFIG_FILE"

# Set unconditional peers to sentry
sed -i "s|^unconditional_peer_ids = .*|unconditional_peer_ids = \"${SENTRY_NODE_ID}\"|" "$CONFIG_FILE"

# Enable addr_book_strict = false to allow private IPs
sed -i "s|^addr_book_strict = true|addr_book_strict = false|" "$CONFIG_FILE"

echo "  Config updated"

echo "Step 3: Verifying changes..."
echo "  persistent_peers: $(grep '^persistent_peers' "$CONFIG_FILE")"
echo "  pex: $(grep '^pex' "$CONFIG_FILE")"
echo "  unconditional_peer_ids: $(grep '^unconditional_peer_ids' "$CONFIG_FILE")"

echo ""
read -p "Restart validator service now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Step 4: Restarting validator..."
    sudo systemctl restart "$VAL_SERVICE"
    sleep 3
    if systemctl is-active --quiet "$VAL_SERVICE"; then
        echo "  Validator restarted successfully"
    else
        echo "  ERROR: Validator failed to start!"
        echo "  Check logs: sudo journalctl -u $VAL_SERVICE -n 50"
        echo "  Restore backup: cp $BACKUP_FILE $CONFIG_FILE"
    fi
else
    echo "Skipping restart. Restart manually:"
    echo "  sudo systemctl restart $VAL_SERVICE"
fi

echo ""
echo "=== Sentry Mode Enabled ==="
echo ""
echo "The validator now connects ONLY through the sentry node."
echo "Ensure the sentry is running before the validator!"
echo ""
echo "To verify connectivity:"
echo "  curl -s http://127.0.0.1:<RPC_PORT>/net_info | jq '.result.peers[].node_info.moniker'"
echo ""
echo "To rollback:"
echo "  cp $BACKUP_FILE $CONFIG_FILE"
echo "  sudo systemctl restart $VAL_SERVICE"
