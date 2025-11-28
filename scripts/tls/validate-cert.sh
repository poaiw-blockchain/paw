#!/bin/bash

################################################################################
# Validate SSL/TLS Certificate
#
# This script performs comprehensive validation of SSL certificates including:
# - Certificate and key matching
# - Certificate chain validation
# - Expiry checks
# - Security configuration
#
# Usage:
#   ./validate-cert.sh [cert_dir]
#
# Default: /etc/letsencrypt/live/explorer.pawchain.network
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
CERT_DIR="${1:-/etc/letsencrypt/live/explorer.pawchain.network}"
CERT_FILE="$CERT_DIR/fullchain.pem"
KEY_FILE="$CERT_DIR/privkey.pem"
CHAIN_FILE="$CERT_DIR/chain.pem"

ERRORS=0
WARNINGS=0

echo -e "${GREEN}PAW Chain - Certificate Validation${NC}"
echo "===================================="
echo ""

# Function to print test result
print_result() {
    local status=$1
    local message=$2

    if [ "$status" == "ok" ]; then
        echo -e "${GREEN}[PASS]${NC} $message"
    elif [ "$status" == "warn" ]; then
        echo -e "${YELLOW}[WARN]${NC} $message"
        ((WARNINGS++))
    else
        echo -e "${RED}[FAIL]${NC} $message"
        ((ERRORS++))
    fi
}

# Check if files exist
echo -e "${BLUE}Checking file existence...${NC}"
if [ -f "$CERT_FILE" ]; then
    print_result "ok" "Certificate file exists: $CERT_FILE"
else
    print_result "fail" "Certificate file not found: $CERT_FILE"
fi

if [ -f "$KEY_FILE" ]; then
    print_result "ok" "Private key file exists: $KEY_FILE"
else
    print_result "fail" "Private key file not found: $KEY_FILE"
fi

if [ -f "$CHAIN_FILE" ]; then
    print_result "ok" "Chain file exists: $CHAIN_FILE"
else
    print_result "warn" "Chain file not found: $CHAIN_FILE"
fi

echo ""

# Exit if critical files don't exist
if [ ! -f "$CERT_FILE" ] || [ ! -f "$KEY_FILE" ]; then
    echo -e "${RED}[ERROR]${NC} Critical files missing. Cannot continue validation."
    exit 1
fi

# Check file permissions
echo -e "${BLUE}Checking file permissions...${NC}"
CERT_PERM=$(stat -c "%a" "$CERT_FILE" 2>/dev/null || stat -f "%A" "$CERT_FILE" 2>/dev/null)
KEY_PERM=$(stat -c "%a" "$KEY_FILE" 2>/dev/null || stat -f "%A" "$KEY_FILE" 2>/dev/null)

if [ "$CERT_PERM" == "644" ] || [ "$CERT_PERM" == "444" ]; then
    print_result "ok" "Certificate permissions are correct: $CERT_PERM"
else
    print_result "warn" "Certificate permissions should be 644, found: $CERT_PERM"
fi

if [ "$KEY_PERM" == "600" ] || [ "$KEY_PERM" == "400" ]; then
    print_result "ok" "Private key permissions are secure: $KEY_PERM"
else
    print_result "fail" "Private key permissions are insecure: $KEY_PERM (should be 600 or 400)"
fi

echo ""

# Validate certificate syntax
echo -e "${BLUE}Validating certificate syntax...${NC}"
if openssl x509 -in "$CERT_FILE" -noout 2>/dev/null; then
    print_result "ok" "Certificate syntax is valid"
else
    print_result "fail" "Certificate syntax is invalid"
fi

# Validate private key syntax
if openssl rsa -in "$KEY_FILE" -noout -check 2>/dev/null || openssl ec -in "$KEY_FILE" -noout 2>/dev/null; then
    print_result "ok" "Private key syntax is valid"
else
    print_result "fail" "Private key syntax is invalid"
fi

echo ""

# Check if certificate and key match
echo -e "${BLUE}Checking certificate and key match...${NC}"
CERT_MODULUS=$(openssl x509 -in "$CERT_FILE" -noout -modulus 2>/dev/null | openssl md5 | cut -d' ' -f2)
KEY_MODULUS=$(openssl rsa -in "$KEY_FILE" -noout -modulus 2>/dev/null | openssl md5 | cut -d' ' -f2)

if [ "$CERT_MODULUS" == "$KEY_MODULUS" ]; then
    print_result "ok" "Certificate and private key match"
else
    print_result "fail" "Certificate and private key do NOT match!"
fi

echo ""

# Check certificate expiry
echo -e "${BLUE}Checking certificate expiry...${NC}"
EXPIRY_DATE=$(openssl x509 -in "$CERT_FILE" -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY_DATE" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "$EXPIRY_DATE" +%s 2>/dev/null)
CURRENT_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( ($EXPIRY_EPOCH - $CURRENT_EPOCH) / 86400 ))

if [ $DAYS_UNTIL_EXPIRY -lt 0 ]; then
    print_result "fail" "Certificate has EXPIRED on $EXPIRY_DATE"
elif [ $DAYS_UNTIL_EXPIRY -lt 7 ]; then
    print_result "fail" "Certificate expires in $DAYS_UNTIL_EXPIRY days (critical)"
elif [ $DAYS_UNTIL_EXPIRY -lt 30 ]; then
    print_result "warn" "Certificate expires in $DAYS_UNTIL_EXPIRY days"
else
    print_result "ok" "Certificate valid for $DAYS_UNTIL_EXPIRY days (expires: $EXPIRY_DATE)"
fi

echo ""

# Check key algorithm and size
echo -e "${BLUE}Checking key algorithm and strength...${NC}"
KEY_SIZE=$(openssl x509 -in "$CERT_FILE" -noout -text | grep "Public-Key:" | sed 's/.*(\([0-9]*\) bit).*/\1/')
KEY_ALGO=$(openssl x509 -in "$CERT_FILE" -noout -text | grep "Public Key Algorithm:" | awk '{print $4}')

echo "  Algorithm: $KEY_ALGO"
echo "  Key Size: $KEY_SIZE bits"

if [ "$KEY_SIZE" -ge 2048 ]; then
    print_result "ok" "Key size is adequate ($KEY_SIZE bits)"
elif [ "$KEY_SIZE" -ge 1024 ]; then
    print_result "warn" "Key size is weak ($KEY_SIZE bits, recommend 2048+)"
else
    print_result "fail" "Key size is too weak ($KEY_SIZE bits)"
fi

echo ""

# Check signature algorithm
echo -e "${BLUE}Checking signature algorithm...${NC}"
SIG_ALGO=$(openssl x509 -in "$CERT_FILE" -noout -text | grep "Signature Algorithm:" | head -1 | awk '{print $3}')
echo "  Signature Algorithm: $SIG_ALGO"

if [[ "$SIG_ALGO" =~ sha256 ]] || [[ "$SIG_ALGO" =~ sha384 ]] || [[ "$SIG_ALGO" =~ sha512 ]]; then
    print_result "ok" "Signature algorithm is secure ($SIG_ALGO)"
elif [[ "$SIG_ALGO" =~ sha1 ]]; then
    print_result "warn" "Signature algorithm is weak ($SIG_ALGO)"
else
    print_result "warn" "Unknown signature algorithm ($SIG_ALGO)"
fi

echo ""

# Display subject and issuer
echo -e "${BLUE}Certificate Information:${NC}"
openssl x509 -in "$CERT_FILE" -noout -subject -issuer

echo ""

# Check SAN (Subject Alternative Names)
echo -e "${BLUE}Checking Subject Alternative Names...${NC}"
SAN=$(openssl x509 -in "$CERT_FILE" -noout -text | grep -A1 "Subject Alternative Name" | tail -1 | sed 's/^\s*//')
if [ -n "$SAN" ]; then
    print_result "ok" "SAN found: $SAN"
else
    print_result "warn" "No Subject Alternative Names found"
fi

echo ""

# Final summary
echo "=================================="
echo -e "${BLUE}Validation Summary:${NC}"
echo "  Errors: $ERRORS"
echo "  Warnings: $WARNINGS"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Certificate validation passed with no issues!"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}[WARNING]${NC} Certificate validation passed with $WARNINGS warning(s)"
    exit 1
else
    echo -e "${RED}[FAILURE]${NC} Certificate validation failed with $ERRORS error(s)"
    exit 2
fi
