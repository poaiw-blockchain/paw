#!/bin/bash
# Disable sentry mode on a PAW validator (rollback to direct peering)
#
# Usage: ./disable-sentry-mode.sh <validator_number>
# Example: ./disable-sentry-mode.sh 1  (for val1)
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
CONFIG_FILE="$VAL_HOME/config/config.toml"
BACKUP_FILE="$VAL_HOME/config/config.toml.pre-sentry"

if [[ ! -d "$VAL_HOME" ]]; then
    echo "ERROR: Validator home not found: $VAL_HOME"
    exit 1
fi

echo "=== Disabling Sentry Mode for val${VAL_NUM} ==="

if [[ -f "$BACKUP_FILE" ]]; then
    echo "Found pre-sentry backup, restoring..."
    cp "$BACKUP_FILE" "$CONFIG_FILE"
    echo "  Restored from: $BACKUP_FILE"
else
    echo "No backup found. Manually restoring direct peer configuration..."

    # Restore direct peer connections based on validator number
    case "$VAL_NUM" in
        1)
            # val1 connects to val2 (local), val3 and val4 (via VPN)
            PEERS="1780e068618ca0ffcba81574d62ab170c2ee3c8b@127.0.0.1:11756,a2b9ab78b0be7f006466131b44ede9a02fc140c4@10.10.0.4:11856,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda@10.10.0.4:11956"
            UNCONDITIONAL="1780e068618ca0ffcba81574d62ab170c2ee3c8b,a2b9ab78b0be7f006466131b44ede9a02fc140c4,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda"
            ;;
        2)
            # val2 connects to val1 (local), val3 and val4 (via VPN)
            PEERS="72c594a424bfc156381860feaca3a2586173eead@127.0.0.1:11656,a2b9ab78b0be7f006466131b44ede9a02fc140c4@10.10.0.4:11856,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda@10.10.0.4:11956"
            UNCONDITIONAL="72c594a424bfc156381860feaca3a2586173eead,a2b9ab78b0be7f006466131b44ede9a02fc140c4,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda"
            ;;
        3)
            # val3 connects to val4 (local), val1 and val2 (via VPN)
            PEERS="f8187d5bafe58b78b00d73b0563b65ad8c0d5fda@127.0.0.1:11956,72c594a424bfc156381860feaca3a2586173eead@10.10.0.2:11656,1780e068618ca0ffcba81574d62ab170c2ee3c8b@10.10.0.2:11756"
            UNCONDITIONAL="72c594a424bfc156381860feaca3a2586173eead,1780e068618ca0ffcba81574d62ab170c2ee3c8b,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda"
            ;;
        4)
            # val4 connects to val3 (local), val1 and val2 (via VPN)
            PEERS="a2b9ab78b0be7f006466131b44ede9a02fc140c4@127.0.0.1:11856,72c594a424bfc156381860feaca3a2586173eead@10.10.0.2:11656,1780e068618ca0ffcba81574d62ab170c2ee3c8b@10.10.0.2:11756"
            UNCONDITIONAL="72c594a424bfc156381860feaca3a2586173eead,1780e068618ca0ffcba81574d62ab170c2ee3c8b,a2b9ab78b0be7f006466131b44ede9a02fc140c4"
            ;;
        *)
            echo "ERROR: Unknown validator number: $VAL_NUM"
            exit 1
            ;;
    esac

    sed -i "s|^persistent_peers = .*|persistent_peers = \"$PEERS\"|" "$CONFIG_FILE"
    sed -i "s|^pex = false|pex = true|" "$CONFIG_FILE"
    sed -i "s|^unconditional_peer_ids = .*|unconditional_peer_ids = \"$UNCONDITIONAL\"|" "$CONFIG_FILE"
    sed -i "s|^addr_book_strict = false|addr_book_strict = true|" "$CONFIG_FILE"

    echo "  Restored direct peer configuration"
fi

echo ""
echo "Current config:"
echo "  persistent_peers: $(grep '^persistent_peers' "$CONFIG_FILE" | head -1)"
echo "  pex: $(grep '^pex' "$CONFIG_FILE" | head -1)"

echo ""
read -p "Restart validator service now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Restarting validator..."
    sudo systemctl restart "$VAL_SERVICE"
    sleep 3
    if systemctl is-active --quiet "$VAL_SERVICE"; then
        echo "  Validator restarted successfully"
    else
        echo "  ERROR: Validator failed to start!"
        echo "  Check logs: sudo journalctl -u $VAL_SERVICE -n 50"
    fi
else
    echo "Skipping restart. Restart manually:"
    echo "  sudo systemctl restart $VAL_SERVICE"
fi

echo ""
echo "=== Sentry Mode Disabled ==="
