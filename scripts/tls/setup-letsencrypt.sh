#!/bin/bash

################################################################################
# Setup Let's Encrypt SSL/TLS Certificates
#
# This script automates the process of obtaining and configuring Let's Encrypt
# SSL certificates for PAW Chain services.
#
# Usage:
#   ./setup-letsencrypt.sh --domain explorer.pawchain.network --email admin@pawchain.network
#
# Requirements:
#   - certbot installed
#   - DNS records pointing to this server
#   - Ports 80 and 443 accessible
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default configuration
DOMAIN=""
EMAIL=""
WEBROOT="/var/www/certbot"
CERT_NAME=""
STAGING=false
DRY_RUN=false

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Setup Let's Encrypt SSL/TLS certificates for PAW Chain

OPTIONS:
    -d, --domain DOMAIN       Domain name (required)
    -e, --email EMAIL         Email for important notifications (required)
    -n, --cert-name NAME      Certificate name (default: domain)
    -w, --webroot PATH        Webroot path (default: /var/www/certbot)
    -s, --staging             Use Let's Encrypt staging server
    --dry-run                 Test without making actual changes
    -h, --help                Show this help message

EXAMPLES:
    $0 -d explorer.pawchain.network -e admin@pawchain.network
    $0 -d api.pawchain.network -e admin@pawchain.network --staging

EOF
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--domain)
            DOMAIN="$2"
            shift 2
            ;;
        -e|--email)
            EMAIL="$2"
            shift 2
            ;;
        -n|--cert-name)
            CERT_NAME="$2"
            shift 2
            ;;
        -w|--webroot)
            WEBROOT="$2"
            shift 2
            ;;
        -s|--staging)
            STAGING=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Unknown option: $1"
            usage
            ;;
    esac
done

# Validate required parameters
if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
    echo -e "${RED}[ERROR]${NC} Domain and email are required"
    usage
fi

# Set certificate name if not provided
if [ -z "$CERT_NAME" ]; then
    CERT_NAME="$DOMAIN"
fi

echo -e "${GREEN}PAW Chain - Let's Encrypt Setup${NC}"
echo "================================"
echo ""
echo "Domain:      $DOMAIN"
echo "Email:       $EMAIL"
echo "Cert Name:   $CERT_NAME"
echo "Webroot:     $WEBROOT"
echo "Staging:     $STAGING"
echo "Dry Run:     $DRY_RUN"
echo ""

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} certbot is not installed"
    echo ""
    echo "Install certbot:"
    echo "  Ubuntu/Debian: sudo apt-get install certbot"
    echo "  CentOS/RHEL:   sudo yum install certbot"
    echo "  macOS:         brew install certbot"
    exit 1
fi

echo -e "${GREEN}[INFO]${NC} certbot version: $(certbot --version)"
echo ""

# Create webroot directory
echo -e "${YELLOW}[INFO]${NC} Creating webroot directory..."
sudo mkdir -p "$WEBROOT"
sudo chown -R www-data:www-data "$WEBROOT" 2>/dev/null || sudo chown -R nginx:nginx "$WEBROOT" 2>/dev/null || true

# Build certbot command
CERTBOT_CMD="sudo certbot certonly --webroot"
CERTBOT_CMD="$CERTBOT_CMD -w $WEBROOT"
CERTBOT_CMD="$CERTBOT_CMD -d $DOMAIN"
CERTBOT_CMD="$CERTBOT_CMD --cert-name $CERT_NAME"
CERTBOT_CMD="$CERTBOT_CMD --email $EMAIL"
CERTBOT_CMD="$CERTBOT_CMD --agree-tos"
CERTBOT_CMD="$CERTBOT_CMD --non-interactive"

if [ "$STAGING" = true ]; then
    CERTBOT_CMD="$CERTBOT_CMD --staging"
    echo -e "${YELLOW}[WARNING]${NC} Using Let's Encrypt STAGING environment"
fi

if [ "$DRY_RUN" = true ]; then
    CERTBOT_CMD="$CERTBOT_CMD --dry-run"
    echo -e "${YELLOW}[INFO]${NC} DRY RUN MODE - No actual changes will be made"
fi

echo ""
echo -e "${YELLOW}[INFO]${NC} Obtaining certificate..."
echo "Command: $CERTBOT_CMD"
echo ""

# Run certbot
if eval "$CERTBOT_CMD"; then
    echo ""
    echo -e "${GREEN}[SUCCESS]${NC} Certificate obtained successfully!"

    if [ "$DRY_RUN" = false ]; then
        CERT_PATH="/etc/letsencrypt/live/$CERT_NAME"

        echo ""
        echo "Certificate files:"
        echo "  - Certificate:      $CERT_PATH/fullchain.pem"
        echo "  - Private Key:      $CERT_PATH/privkey.pem"
        echo "  - Chain:            $CERT_PATH/chain.pem"
        echo "  - Certificate Only: $CERT_PATH/cert.pem"

        # Generate DH parameters if they don't exist
        DHPARAM_FILE="/etc/nginx/ssl/dhparam.pem"
        if [ ! -f "$DHPARAM_FILE" ]; then
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Generating DH parameters (this may take a while)..."
            sudo mkdir -p /etc/nginx/ssl
            sudo openssl dhparam -out "$DHPARAM_FILE" 2048
            echo -e "${GREEN}[SUCCESS]${NC} DH parameters generated: $DHPARAM_FILE"
        fi

        echo ""
        echo -e "${YELLOW}[INFO]${NC} Next steps:"
        echo "  1. Update your NGINX configuration to use the certificate"
        echo "  2. Test NGINX configuration: sudo nginx -t"
        echo "  3. Reload NGINX: sudo systemctl reload nginx"
        echo "  4. Setup automatic renewal (see setup-cert-renewal.sh)"
        echo ""
        echo "Certificate will expire in 90 days"
        echo "Auto-renewal will be attempted by certbot timer/cron"
    fi
else
    echo ""
    echo -e "${RED}[ERROR]${NC} Failed to obtain certificate"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Ensure DNS records point to this server"
    echo "  2. Check if port 80 is accessible from the internet"
    echo "  3. Verify webroot path is correct and accessible"
    echo "  4. Check certbot logs: /var/log/letsencrypt/letsencrypt.log"
    exit 1
fi
