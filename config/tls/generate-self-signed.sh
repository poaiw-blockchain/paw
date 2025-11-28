#!/bin/bash

# Generate self-signed TLS certificates for development/testing
# WARNING: Do NOT use self-signed certificates in production!

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${CERT_DIR:-$SCRIPT_DIR/certs}"

# Configuration
DOMAIN="${DOMAIN:-localhost}"
DAYS_VALID="${DAYS_VALID:-365}"
KEY_SIZE="${KEY_SIZE:-2048}"

echo "==================================================================="
echo "Generating Self-Signed TLS Certificates"
echo "==================================================================="
echo ""
echo "WARNING: Self-signed certificates should ONLY be used for"
echo "development and testing. Use Let's Encrypt or a commercial CA"
echo "for production deployments."
echo ""
echo "Configuration:"
echo "  Domain: $DOMAIN"
echo "  Valid for: $DAYS_VALID days"
echo "  Key size: $KEY_SIZE bits"
echo "  Output directory: $CERT_DIR"
echo ""

# Create output directory
mkdir -p "$CERT_DIR"

# Generate private key
echo "Generating private key..."
openssl genrsa -out "$CERT_DIR/server.key" $KEY_SIZE

# Create certificate signing request configuration
cat > "$CERT_DIR/csr.conf" <<EOF
[req]
default_bits = $KEY_SIZE
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[dn]
C = US
ST = California
L = San Francisco
O = PAW Blockchain
OU = Development
CN = $DOMAIN

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = *.$DOMAIN
DNS.3 = localhost
DNS.4 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate certificate signing request
echo "Generating certificate signing request..."
openssl req -new -key "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.csr" \
    -config "$CERT_DIR/csr.conf"

# Generate self-signed certificate
echo "Generating self-signed certificate..."
openssl x509 -req \
    -in "$CERT_DIR/server.csr" \
    -signkey "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.crt" \
    -days $DAYS_VALID \
    -sha256 \
    -extensions req_ext \
    -extfile "$CERT_DIR/csr.conf"

# Set proper permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo ""
echo "==================================================================="
echo "Certificate Generation Complete!"
echo "==================================================================="
echo ""
echo "Generated files:"
echo "  Private key: $CERT_DIR/server.key (0600)"
echo "  Certificate: $CERT_DIR/server.crt (0644)"
echo ""
echo "Certificate details:"
openssl x509 -in "$CERT_DIR/server.crt" -noout -text | grep -A 3 "Subject:"
openssl x509 -in "$CERT_DIR/server.crt" -noout -text | grep -A 2 "Validity"
echo ""
echo "To verify the certificate:"
echo "  openssl x509 -in $CERT_DIR/server.crt -noout -text"
echo ""
echo "To use with PAW node, update your config files:"
echo "  RPC:  tls_cert_file = \"$CERT_DIR/server.crt\""
echo "        tls_key_file = \"$CERT_DIR/server.key\""
echo ""
echo "  gRPC: tls-cert-path = \"$CERT_DIR/server.crt\""
echo "        tls-key-path = \"$CERT_DIR/server.key\""
echo ""
echo "  API:  tls-cert-path = \"$CERT_DIR/server.crt\""
echo "        tls-key-path = \"$CERT_DIR/server.key\""
echo ""
echo "REMEMBER: These are self-signed certificates for testing only!"
echo "Use Let's Encrypt (certbot) for production deployments."
echo ""
