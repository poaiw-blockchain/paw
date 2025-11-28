#!/bin/bash

# Generate mTLS certificates for P2P communication
# Creates a Certificate Authority and node certificates for mutual TLS authentication

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${CERT_DIR:-$SCRIPT_DIR/mtls}"

# Configuration
CA_DAYS="${CA_DAYS:-3650}"  # 10 years
CERT_DAYS="${CERT_DAYS:-365}"  # 1 year
KEY_SIZE="${KEY_SIZE:-2048}"
NODE_NAME="${NODE_NAME:-paw-node-1}"

echo "==================================================================="
echo "Generating mTLS Certificates for P2P Communication"
echo "==================================================================="
echo ""
echo "Configuration:"
echo "  Node name: $NODE_NAME"
echo "  CA valid for: $CA_DAYS days"
echo "  Certificates valid for: $CERT_DAYS days"
echo "  Key size: $KEY_SIZE bits"
echo "  Output directory: $CERT_DIR"
echo ""

# Create output directory
mkdir -p "$CERT_DIR"

# ============================================================================
# Step 1: Generate Certificate Authority (CA)
# ============================================================================

echo "Step 1: Generating Certificate Authority..."

# Generate CA private key
openssl genrsa -out "$CERT_DIR/ca-key.pem" $KEY_SIZE

# Generate CA certificate
openssl req -new -x509 \
    -key "$CERT_DIR/ca-key.pem" \
    -out "$CERT_DIR/ca-cert.pem" \
    -days $CA_DAYS \
    -subj "/C=US/ST=California/L=San Francisco/O=PAW Blockchain/OU=P2P Network/CN=PAW P2P CA"

echo "✓ CA certificate generated"

# ============================================================================
# Step 2: Generate Node Certificate
# ============================================================================

echo ""
echo "Step 2: Generating node certificate for $NODE_NAME..."

# Generate node private key
openssl genrsa -out "$CERT_DIR/p2p-key.pem" $KEY_SIZE

# Create certificate signing request
openssl req -new \
    -key "$CERT_DIR/p2p-key.pem" \
    -out "$CERT_DIR/p2p-csr.pem" \
    -subj "/C=US/ST=California/L=San Francisco/O=PAW Blockchain/OU=Validator Node/CN=$NODE_NAME"

# Create certificate extensions
cat > "$CERT_DIR/p2p-ext.conf" <<EOF
subjectAltName = DNS:$NODE_NAME,DNS:localhost,IP:127.0.0.1
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth,clientAuth
EOF

# Sign the certificate with CA
openssl x509 -req \
    -in "$CERT_DIR/p2p-csr.pem" \
    -CA "$CERT_DIR/ca-cert.pem" \
    -CAkey "$CERT_DIR/ca-key.pem" \
    -CAcreateserial \
    -out "$CERT_DIR/p2p-cert.pem" \
    -days $CERT_DAYS \
    -extfile "$CERT_DIR/p2p-ext.conf"

echo "✓ Node certificate generated and signed"

# ============================================================================
# Step 3: Set Permissions
# ============================================================================

echo ""
echo "Step 3: Setting secure file permissions..."

chmod 600 "$CERT_DIR/ca-key.pem"
chmod 644 "$CERT_DIR/ca-cert.pem"
chmod 600 "$CERT_DIR/p2p-key.pem"
chmod 644 "$CERT_DIR/p2p-cert.pem"

echo "✓ Permissions set"

# ============================================================================
# Step 4: Verify Certificates
# ============================================================================

echo ""
echo "Step 4: Verifying certificates..."

# Verify node certificate
openssl verify -CAfile "$CERT_DIR/ca-cert.pem" "$CERT_DIR/p2p-cert.pem"

echo "✓ Certificate verification successful"

# ============================================================================
# Summary
# ============================================================================

echo ""
echo "==================================================================="
echo "mTLS Certificate Generation Complete!"
echo "==================================================================="
echo ""
echo "Generated files:"
echo "  CA Certificate:   $CERT_DIR/ca-cert.pem (0644) - Distribute to all nodes"
echo "  CA Private Key:   $CERT_DIR/ca-key.pem (0600) - Keep secure!"
echo "  Node Certificate: $CERT_DIR/p2p-cert.pem (0644)"
echo "  Node Private Key: $CERT_DIR/p2p-key.pem (0600)"
echo ""
echo "Certificate details:"
echo ""
echo "CA Certificate:"
openssl x509 -in "$CERT_DIR/ca-cert.pem" -noout -subject -dates
echo ""
echo "Node Certificate:"
openssl x509 -in "$CERT_DIR/p2p-cert.pem" -noout -subject -dates
echo ""
echo "==================================================================="
echo "Configuration Instructions"
echo "==================================================================="
echo ""
echo "1. Copy files to each validator node:"
echo "   - ca-cert.pem (same on all nodes)"
echo "   - p2p-cert.pem (unique per node)"
echo "   - p2p-key.pem (unique per node)"
echo ""
echo "2. Update config.toml on each node:"
echo ""
echo "   [p2p]"
echo "   p2p_tls_cert_file = \"$CERT_DIR/p2p-cert.pem\""
echo "   p2p_tls_key_file = \"$CERT_DIR/p2p-key.pem\""
echo "   p2p_tls_ca_file = \"$CERT_DIR/ca-cert.pem\""
echo "   p2p_tls_require_client_cert = true"
echo ""
echo "3. Generate separate certificates for each validator:"
echo "   NODE_NAME=paw-node-2 ./generate-mtls-certs.sh"
echo "   NODE_NAME=paw-node-3 ./generate-mtls-certs.sh"
echo ""
echo "4. Restart all nodes after configuration"
echo ""
echo "Security Notes:"
echo "  - Keep CA private key (ca-key.pem) extremely secure"
echo "  - Each node should have unique certificate/key pair"
echo "  - CA certificate (ca-cert.pem) must be identical on all nodes"
echo "  - Rotate node certificates annually"
echo "  - Monitor certificate expiration dates"
echo ""
