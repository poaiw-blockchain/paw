#!/bin/bash

################################################################################
# Manual Certificate Renewal Script
#
# This script manually renews Let's Encrypt certificates and reloads NGINX
#
# Usage:
#   ./renew-cert.sh [--force] [--cert-name NAME]
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
FORCE=false
CERT_NAME=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE=true
            shift
            ;;
        --cert-name)
            CERT_NAME="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [--force] [--cert-name NAME]"
            echo ""
            echo "Options:"
            echo "  --force           Force renewal even if not due"
            echo "  --cert-name NAME  Renew specific certificate"
            echo "  -h, --help        Show this help"
            exit 0
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${GREEN}PAW Chain - Certificate Renewal${NC}"
echo "================================="
echo ""

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} certbot is not installed"
    exit 1
fi

# Build renewal command
RENEW_CMD="sudo certbot renew"

if [ "$FORCE" = true ]; then
    RENEW_CMD="$RENEW_CMD --force-renewal"
    echo -e "${YELLOW}[WARNING]${NC} Force renewal enabled"
fi

if [ -n "$CERT_NAME" ]; then
    RENEW_CMD="$RENEW_CMD --cert-name $CERT_NAME"
    echo "Certificate: $CERT_NAME"
fi

echo ""
echo -e "${YELLOW}[INFO]${NC} Running certificate renewal..."
echo ""

# Run renewal
if eval "$RENEW_CMD"; then
    echo ""
    echo -e "${GREEN}[SUCCESS]${NC} Certificate renewal completed successfully"

    # Reload NGINX
    echo ""
    echo -e "${YELLOW}[INFO]${NC} Reloading NGINX..."

    if command -v systemctl &> /dev/null; then
        sudo systemctl reload nginx
    elif command -v service &> /dev/null; then
        sudo service nginx reload
    else
        sudo nginx -s reload
    fi

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[SUCCESS]${NC} NGINX reloaded successfully"
    else
        echo -e "${RED}[ERROR]${NC} Failed to reload NGINX"
        exit 1
    fi

    # Show certificate info
    echo ""
    echo -e "${BLUE}Certificate Information:${NC}"
    sudo certbot certificates

else
    echo ""
    echo -e "${RED}[ERROR]${NC} Certificate renewal failed"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check certbot logs: /var/log/letsencrypt/letsencrypt.log"
    echo "  - Verify DNS records"
    echo "  - Ensure webroot is accessible"
    echo "  - Check port 80 is open"
    exit 1
fi

echo ""
echo -e "${GREEN}[COMPLETE]${NC} Certificate renewal process finished"
