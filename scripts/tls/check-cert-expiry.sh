#!/bin/bash

################################################################################
# Check SSL/TLS Certificate Expiry
#
# This script checks the expiration date of SSL certificates and warns if
# they are about to expire.
#
# Usage:
#   ./check-cert-expiry.sh [cert_path] [warning_days]
#
# Default: /etc/letsencrypt/live/explorer.pawchain.network/fullchain.pem
# Default warning: 30 days
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
CERT_PATH="${1:-/etc/letsencrypt/live/explorer.pawchain.network/fullchain.pem}"
WARNING_DAYS="${2:-30}"

echo -e "${GREEN}PAW Chain - Certificate Expiry Checker${NC}"
echo "========================================"
echo ""

# Check if certificate file exists
if [ ! -f "$CERT_PATH" ]; then
    echo -e "${RED}[ERROR]${NC} Certificate file not found: $CERT_PATH"
    exit 1
fi

# Get certificate expiry date
EXPIRY_DATE=$(openssl x509 -in "$CERT_PATH" -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY_DATE" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "$EXPIRY_DATE" +%s 2>/dev/null)
CURRENT_EPOCH=$(date +%s)

# Calculate days until expiry
DAYS_UNTIL_EXPIRY=$(( ($EXPIRY_EPOCH - $CURRENT_EPOCH) / 86400 ))

echo "Certificate: $CERT_PATH"
echo "Expiry Date: $EXPIRY_DATE"
echo "Days Until Expiry: $DAYS_UNTIL_EXPIRY"
echo ""

# Display certificate details
echo -e "${YELLOW}Certificate Details:${NC}"
openssl x509 -in "$CERT_PATH" -noout -subject -issuer -dates

echo ""

# Check expiry status
if [ $DAYS_UNTIL_EXPIRY -lt 0 ]; then
    echo -e "${RED}[CRITICAL]${NC} Certificate has EXPIRED!"
    exit 2
elif [ $DAYS_UNTIL_EXPIRY -lt 7 ]; then
    echo -e "${RED}[CRITICAL]${NC} Certificate expires in $DAYS_UNTIL_EXPIRY days! Renew immediately!"
    exit 2
elif [ $DAYS_UNTIL_EXPIRY -lt $WARNING_DAYS ]; then
    echo -e "${YELLOW}[WARNING]${NC} Certificate expires in $DAYS_UNTIL_EXPIRY days. Consider renewing soon."
    exit 1
else
    echo -e "${GREEN}[OK]${NC} Certificate is valid for $DAYS_UNTIL_EXPIRY more days."
    exit 0
fi
