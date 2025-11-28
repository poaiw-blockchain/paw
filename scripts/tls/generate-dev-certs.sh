#!/bin/bash

################################################################################
# Generate Self-Signed Development Certificates for PAW Chain
#
# This script creates self-signed TLS/SSL certificates for local development
# and testing. DO NOT use these certificates in production!
#
# Usage:
#   ./generate-dev-certs.sh [domain]
#
# Default domain: localhost
################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="${1:-localhost}"
CERTS_DIR="${CERTS_DIR:-./certs}"
VALIDITY_DAYS=365
KEY_SIZE=2048
COUNTRY="US"
STATE="California"
CITY="San Francisco"
ORG="PAW Chain"
ORG_UNIT="Development"

echo -e "${GREEN}PAW Chain - Development Certificate Generator${NC}"
echo "=============================================="
echo ""

# Create certificates directory
mkdir -p "$CERTS_DIR"
echo -e "${YELLOW}[INFO]${NC} Certificates directory: $CERTS_DIR"

# Generate private key
echo -e "${YELLOW}[INFO]${NC} Generating private key..."
openssl genrsa -out "$CERTS_DIR/privkey.pem" $KEY_SIZE 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Private key generated"
else
    echo -e "${RED}[ERROR]${NC} Failed to generate private key"
    exit 1
fi

# Create OpenSSL configuration file with SAN
cat > "$CERTS_DIR/openssl.cnf" <<EOF
[req]
default_bits = $KEY_SIZE
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext
x509_extensions = v3_ca

[dn]
C = $COUNTRY
ST = $STATE
L = $CITY
O = $ORG
OU = $ORG_UNIT
CN = $DOMAIN

[req_ext]
subjectAltName = @alt_names

[v3_ca]
subjectAltName = @alt_names
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = *.$DOMAIN
DNS.3 = localhost
DNS.4 = 127.0.0.1
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate self-signed certificate
echo -e "${YELLOW}[INFO]${NC} Generating self-signed certificate..."
openssl req -new -x509 -sha256 \
    -key "$CERTS_DIR/privkey.pem" \
    -out "$CERTS_DIR/fullchain.pem" \
    -days $VALIDITY_DAYS \
    -config "$CERTS_DIR/openssl.cnf" 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Certificate generated"
else
    echo -e "${RED}[ERROR]${NC} Failed to generate certificate"
    exit 1
fi

# Create chain file (same as fullchain for self-signed)
cp "$CERTS_DIR/fullchain.pem" "$CERTS_DIR/chain.pem"

# Generate DH parameters for enhanced security
echo -e "${YELLOW}[INFO]${NC} Generating Diffie-Hellman parameters (this may take a while)..."
openssl dhparam -out "$CERTS_DIR/dhparam.pem" 2048 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} DH parameters generated"
else
    echo -e "${RED}[ERROR]${NC} Failed to generate DH parameters"
    exit 1
fi

# Set proper permissions
chmod 600 "$CERTS_DIR/privkey.pem"
chmod 644 "$CERTS_DIR/fullchain.pem"
chmod 644 "$CERTS_DIR/chain.pem"
chmod 644 "$CERTS_DIR/dhparam.pem"

# Display certificate information
echo ""
echo -e "${GREEN}Certificate Details:${NC}"
echo "===================="
openssl x509 -in "$CERTS_DIR/fullchain.pem" -noout -text | grep -E "(Subject:|Issuer:|Not Before|Not After|DNS:|IP Address:)"

echo ""
echo -e "${GREEN}[SUCCESS]${NC} All certificates generated successfully!"
echo ""
echo "Certificate files created:"
echo "  - Private key:  $CERTS_DIR/privkey.pem"
echo "  - Certificate:  $CERTS_DIR/fullchain.pem"
echo "  - Chain:        $CERTS_DIR/chain.pem"
echo "  - DH Params:    $CERTS_DIR/dhparam.pem"
echo ""
echo -e "${YELLOW}[WARNING]${NC} These are self-signed certificates for development only!"
echo "         Do NOT use in production. Use Let's Encrypt or a trusted CA."
echo ""
echo "To trust this certificate locally:"
echo "  - macOS: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERTS_DIR/fullchain.pem"
echo "  - Linux: sudo cp $CERTS_DIR/fullchain.pem /usr/local/share/ca-certificates/ && sudo update-ca-certificates"
echo "  - Windows: Import $CERTS_DIR/fullchain.pem to Trusted Root Certification Authorities"
echo ""
