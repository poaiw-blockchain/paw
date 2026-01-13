#!/bin/bash
# Deploy nginx configuration to route public traffic through sentry nodes
# Run this script ON paw-testnet server (54.39.103.49)
#
# This script:
# 1. Backs up existing nginx configs
# 2. Deploys new sentry-routing configs
# 3. Tests nginx configuration
# 4. Reloads nginx if valid
#
# Prerequisites:
# - Sentry node must be running and synced on services-testnet
# - VPN connection to services-testnet (10.10.0.4) must be active

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NGINX_SITES="/etc/nginx/sites-enabled"
BACKUP_DIR="/etc/nginx/backup-$(date +%Y%m%d-%H%M%S)"

echo "=== PAW Testnet Nginx Sentry Routing Deployment ==="
echo "Date: $(date)"
echo ""

# Check if running on paw-testnet
if [[ "$(hostname)" != *"paw"* ]] && [[ ! -f "/etc/nginx/sites-enabled/testnet-rpc.poaiw.org" ]]; then
    echo "WARNING: This script should be run on paw-testnet server"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

# Verify sentry is reachable and synced
echo "Step 1: Verifying sentry node status..."
SENTRY_STATUS=$(curl -s --connect-timeout 5 http://10.10.0.4:12057/status 2>/dev/null || echo "")
if [[ -z "$SENTRY_STATUS" ]]; then
    echo "ERROR: Cannot reach sentry node at 10.10.0.4:12057"
    echo "Ensure sentry is running: ssh services-testnet 'systemctl status pawd-sentry'"
    exit 1
fi

SENTRY_HEIGHT=$(echo "$SENTRY_STATUS" | jq -r '.result.sync_info.latest_block_height')
SENTRY_CATCHING_UP=$(echo "$SENTRY_STATUS" | jq -r '.result.sync_info.catching_up')

if [[ "$SENTRY_CATCHING_UP" == "true" ]]; then
    echo "ERROR: Sentry is still catching up (height: $SENTRY_HEIGHT)"
    echo "Wait for sentry to sync before deploying"
    exit 1
fi

echo "  Sentry OK: height=$SENTRY_HEIGHT, synced=true"

# Verify sentry REST API
echo "Step 2: Verifying sentry REST API..."
REST_STATUS=$(curl -s --connect-timeout 5 http://10.10.0.4:12017/cosmos/base/tendermint/v1beta1/syncing 2>/dev/null || echo "")
if [[ -z "$REST_STATUS" ]]; then
    echo "ERROR: Cannot reach sentry REST API at 10.10.0.4:12017"
    exit 1
fi
echo "  REST API OK"

# Verify sentry gRPC (basic TCP check)
echo "Step 3: Verifying sentry gRPC..."
if timeout 3 bash -c "cat < /dev/null > /dev/tcp/10.10.0.4/12090" 2>/dev/null; then
    echo "  gRPC OK"
else
    echo "WARNING: Cannot reach sentry gRPC at 10.10.0.4:12090"
    echo "gRPC routing may not work. Continue anyway? (y/N)"
    read -p "" -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

# Backup existing configs
echo ""
echo "Step 4: Backing up existing nginx configs..."
sudo mkdir -p "$BACKUP_DIR"
sudo cp -r /etc/nginx/sites-enabled/* "$BACKUP_DIR/" 2>/dev/null || true
echo "  Backup saved to: $BACKUP_DIR"

# Deploy new configs
echo ""
echo "Step 5: Deploying sentry routing configs..."

if [[ -f "$SCRIPT_DIR/testnet-rpc.poaiw.org.conf" ]]; then
    sudo cp "$SCRIPT_DIR/testnet-rpc.poaiw.org.conf" "$NGINX_SITES/testnet-rpc.poaiw.org"
    echo "  Deployed: testnet-rpc.poaiw.org"
else
    echo "  WARNING: testnet-rpc.poaiw.org.conf not found"
fi

if [[ -f "$SCRIPT_DIR/testnet-api.poaiw.org.conf" ]]; then
    sudo cp "$SCRIPT_DIR/testnet-api.poaiw.org.conf" "$NGINX_SITES/testnet-api.poaiw.org"
    echo "  Deployed: testnet-api.poaiw.org"
else
    echo "  WARNING: testnet-api.poaiw.org.conf not found"
fi

if [[ -f "$SCRIPT_DIR/testnet-grpc.poaiw.org.conf" ]]; then
    sudo cp "$SCRIPT_DIR/testnet-grpc.poaiw.org.conf" "$NGINX_SITES/testnet-grpc.poaiw.org"
    echo "  Deployed: testnet-grpc.poaiw.org"
else
    echo "  WARNING: testnet-grpc.poaiw.org.conf not found"
fi

# Test nginx configuration
echo ""
echo "Step 6: Testing nginx configuration..."
if sudo nginx -t 2>&1; then
    echo "  Configuration valid"
else
    echo "ERROR: nginx configuration invalid!"
    echo "Restoring backup..."
    sudo cp "$BACKUP_DIR"/* "$NGINX_SITES/" 2>/dev/null || true
    exit 1
fi

# Reload nginx
echo ""
echo "Step 7: Reloading nginx..."
sudo systemctl reload nginx
echo "  nginx reloaded"

# Verify endpoints
echo ""
echo "Step 8: Verifying public endpoints..."
sleep 2

RPC_CHECK=$(curl -s --connect-timeout 5 https://testnet-rpc.poaiw.org/status | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "FAIL")
REST_CHECK=$(curl -s --connect-timeout 5 https://testnet-api.poaiw.org/cosmos/base/tendermint/v1beta1/syncing | jq -r '.syncing' 2>/dev/null || echo "FAIL")

echo "  RPC (testnet-rpc.poaiw.org): height=$RPC_CHECK"
echo "  REST (testnet-api.poaiw.org): syncing=$REST_CHECK"

if [[ "$RPC_CHECK" == "FAIL" ]] || [[ "$REST_CHECK" == "FAIL" ]]; then
    echo ""
    echo "WARNING: Some endpoints may not be responding correctly"
    echo "Check nginx logs: sudo tail -f /var/log/nginx/error.log"
fi

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Traffic is now routed through the sentry node:"
echo "  Public -> Cloudflare -> nginx -> Sentry (10.10.0.4) -> Validators"
echo ""
echo "Validator RPC endpoints are no longer directly exposed to public traffic."
echo ""
echo "To rollback: sudo cp $BACKUP_DIR/* $NGINX_SITES/ && sudo systemctl reload nginx"
