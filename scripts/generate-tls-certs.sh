#!/bin/bash

# PAW Blockchain TLS Certificate Generation Script
# This script generates TLS certificates for the API server
#
# Usage:
#   ./scripts/generate-tls-certs.sh [environment]
#
# Arguments:
#   environment: "dev" (default) or "staging"
#
# For PRODUCTION, use certificates from a trusted CA like Let's Encrypt
# See PRODUCTION_CERTIFICATES.md for instructions

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CERTS_DIR="$PROJECT_ROOT/certs"

# Default values
ENVIRONMENT="${1:-dev}"
DAYS_VALID=365
KEY_SIZE=2048
COUNTRY="US"
STATE="California"
LOCALITY="San Francisco"
ORGANIZATION="PAW Blockchain"
ORGANIZATIONAL_UNIT="Engineering"
COMMON_NAME="localhost"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${GREEN}========================================${NC}"
}

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if OpenSSL is installed
check_dependencies() {
    if ! command -v openssl &> /dev/null; then
        print_error "OpenSSL is not installed. Please install it first."
        echo "  - On Ubuntu/Debian: sudo apt-get install openssl"
        echo "  - On macOS: brew install openssl"
        echo "  - On Windows: Use  Bash or WSL with OpenSSL installed"
        exit 1
    fi
    print_info "OpenSSL version: $(openssl version)"
}

# Create certs directory if it doesn't exist
setup_directories() {
    if [ ! -d "$CERTS_DIR" ]; then
        mkdir -p "$CERTS_DIR"
        print_info "Created directory: $CERTS_DIR"
    fi

    # Create environment-specific subdirectory
    ENV_CERTS_DIR="$CERTS_DIR/$ENVIRONMENT"
    if [ ! -d "$ENV_CERTS_DIR" ]; then
        mkdir -p "$ENV_CERTS_DIR"
        print_info "Created directory: $ENV_CERTS_DIR"
    fi
}

# Generate OpenSSL configuration file
generate_openssl_config() {
    local config_file="$ENV_CERTS_DIR/openssl.cnf"

    cat > "$config_file" <<EOF
[req]
default_bits = $KEY_SIZE
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C = $COUNTRY
ST = $STATE
L = $LOCALITY
O = $ORGANIZATION
OU = $ORGANIZATIONAL_UNIT
CN = $COMMON_NAME
emailAddress = admin@pawchain.io

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[v3_ca]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints = critical, CA:true
keyUsage = critical, digitalSignature, cRLSign, keyCertSign

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
DNS.3 = 127.0.0.1
DNS.4 = pawchain.local
DNS.5 = *.pawchain.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

    if [ "$ENVIRONMENT" == "staging" ]; then
        cat >> "$config_file" <<EOF
DNS.6 = staging.pawchain.io
DNS.7 = *.staging.pawchain.io
EOF
    fi

    print_info "Generated OpenSSL configuration: $config_file"
}

# Generate Certificate Authority (CA)
generate_ca() {
    print_header "Generating Certificate Authority (CA)"

    local ca_key="$ENV_CERTS_DIR/ca-key.pem"
    local ca_cert="$ENV_CERTS_DIR/ca-cert.pem"

    # Generate CA private key
    openssl genrsa -out "$ca_key" 4096
    print_info "Generated CA private key: $ca_key"

    # Generate CA certificate
    openssl req -new -x509 -sha256 \
        -key "$ca_key" \
        -out "$ca_cert" \
        -days $((DAYS_VALID * 2)) \
        -config "$ENV_CERTS_DIR/openssl.cnf" \
        -extensions v3_ca \
        -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT CA/CN=PAW Blockchain CA"

    print_info "Generated CA certificate: $ca_cert"

    # Set restrictive permissions
    chmod 600 "$ca_key"
    chmod 644 "$ca_cert"
}

# Generate server certificate
generate_server_cert() {
    print_header "Generating Server Certificate"

    local server_key="$ENV_CERTS_DIR/server-key.pem"
    local server_csr="$ENV_CERTS_DIR/server-csr.pem"
    local server_cert="$ENV_CERTS_DIR/server-cert.pem"
    local ca_key="$ENV_CERTS_DIR/ca-key.pem"
    local ca_cert="$ENV_CERTS_DIR/ca-cert.pem"

    # Generate server private key
    openssl genrsa -out "$server_key" $KEY_SIZE
    print_info "Generated server private key: $server_key"

    # Generate certificate signing request (CSR)
    openssl req -new -sha256 \
        -key "$server_key" \
        -out "$server_csr" \
        -config "$ENV_CERTS_DIR/openssl.cnf"

    print_info "Generated certificate signing request: $server_csr"

    # Sign the CSR with CA
    openssl x509 -req -sha256 \
        -in "$server_csr" \
        -CA "$ca_cert" \
        -CAkey "$ca_key" \
        -CAcreateserial \
        -out "$server_cert" \
        -days $DAYS_VALID \
        -extensions v3_req \
        -extfile "$ENV_CERTS_DIR/openssl.cnf"

    print_info "Generated server certificate: $server_cert"

    # Set restrictive permissions
    chmod 600 "$server_key"
    chmod 644 "$server_cert"

    # Clean up CSR
    rm -f "$server_csr"
}

# Verify certificates
verify_certificates() {
    print_header "Verifying Certificates"

    local server_cert="$ENV_CERTS_DIR/server-cert.pem"
    local ca_cert="$ENV_CERTS_DIR/ca-cert.pem"

    # Verify server certificate against CA
    if openssl verify -CAfile "$ca_cert" "$server_cert" > /dev/null 2>&1; then
        print_info "✓ Server certificate verification: PASSED"
    else
        print_error "✗ Server certificate verification: FAILED"
        exit 1
    fi

    # Display certificate information
    echo ""
    print_info "Server Certificate Details:"
    openssl x509 -in "$server_cert" -noout -subject -issuer -dates

    # Display Subject Alternative Names
    echo ""
    print_info "Subject Alternative Names (SANs):"
    openssl x509 -in "$server_cert" -noout -text | grep -A1 "Subject Alternative Name" || true
}

# Create configuration example
create_config_example() {
    print_header "Creating Configuration Example"

    local config_example="$ENV_CERTS_DIR/server-config-example.yaml"

    cat > "$config_example" <<EOF
# PAW API Server TLS Configuration Example
#
# Add these settings to your config.yaml or use environment variables

api:
  # Enable TLS
  tls_enabled: true

  # Path to server certificate
  tls_cert_file: "$ENV_CERTS_DIR/server-cert.pem"

  # Path to server private key
  tls_key_file: "$ENV_CERTS_DIR/server-key.pem"

  # Server address
  address: "0.0.0.0:8443"

# Environment Variables (alternative to YAML config):
#   export PAW_API_TLS_ENABLED=true
#   export PAW_API_TLS_CERT_FILE="$ENV_CERTS_DIR/server-cert.pem"
#   export PAW_API_TLS_KEY_FILE="$ENV_CERTS_DIR/server-key.pem"
#   export PAW_API_ADDRESS="0.0.0.0:8443"
EOF

    print_info "Created configuration example: $config_example"
}

# Display usage instructions
display_instructions() {
    print_header "TLS Certificates Generated Successfully!"

    echo ""
    echo -e "${GREEN}Certificate Files:${NC}"
    echo "  CA Certificate:     $ENV_CERTS_DIR/ca-cert.pem"
    echo "  Server Certificate: $ENV_CERTS_DIR/server-cert.pem"
    echo "  Server Private Key: $ENV_CERTS_DIR/server-key.pem"
    echo ""

    echo -e "${GREEN}Next Steps:${NC}"
    echo ""
    echo "1. Configure the API server to use these certificates:"
    echo "   See: $ENV_CERTS_DIR/server-config-example.yaml"
    echo ""
    echo "2. For development/testing, trust the CA certificate:"
    echo "   - macOS:   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $ENV_CERTS_DIR/ca-cert.pem"
    echo "   - Linux:   sudo cp $ENV_CERTS_DIR/ca-cert.pem /usr/local/share/ca-certificates/paw-ca.crt && sudo update-ca-certificates"
    echo "   - Windows: Import $ENV_CERTS_DIR/ca-cert.pem to 'Trusted Root Certification Authorities'"
    echo ""
    echo "3. Test the API server with curl:"
    echo "   curl --cacert $ENV_CERTS_DIR/ca-cert.pem https://localhost:8443/api/v1/health"
    echo ""

    print_warning "IMPORTANT: These are self-signed certificates for ${ENVIRONMENT} use only!"
    print_warning "For PRODUCTION, use certificates from a trusted CA (e.g., Let's Encrypt)"
    echo ""
    echo "See PRODUCTION_CERTIFICATES.md for production certificate instructions."
}

# Create gitignore for certs directory
create_gitignore() {
    local gitignore="$CERTS_DIR/ignore"

    if [ ! -f "$gitignore" ]; then
        cat > "$gitignore" <<EOF
# Ignore all certificate files for security
*.pem
*.key
*.crt
*.csr
*.srl

# Keep only documentation and scripts
!ignore
!README.md
EOF
        print_info "Created ignore: $gitignore"
    fi
}

# Main execution
main() {
    print_header "PAW Blockchain TLS Certificate Generator"
    echo ""
    print_info "Environment: $ENVIRONMENT"
    print_info "Certificate validity: $DAYS_VALID days"
    echo ""

    # Validate environment
    if [ "$ENVIRONMENT" != "dev" ] && [ "$ENVIRONMENT" != "staging" ]; then
        print_error "Invalid environment: $ENVIRONMENT"
        echo "Valid environments: dev, staging"
        echo ""
        echo "For PRODUCTION certificates, use a trusted CA like Let's Encrypt"
        exit 1
    fi

    check_dependencies
    setup_directories
    create_gitignore
    generate_openssl_config
    generate_ca
    generate_server_cert
    verify_certificates
    create_config_example
    display_instructions
}

# Run main function
main
