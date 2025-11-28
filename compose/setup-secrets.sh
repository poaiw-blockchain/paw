#!/bin/bash

# ============================================================================
# PAW Blockchain - Secrets Setup Script
# ============================================================================
# This script generates secure random passwords and stores them in Docker
# secrets files with proper permissions.
#
# SECURITY: Never commit the secrets/ directory to !
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS_DIR="${SCRIPT_DIR}/secrets"

echo -e "${GREEN}============================================================================${NC}"
echo -e "${GREEN}PAW Blockchain - Secrets Setup${NC}"
echo -e "${GREEN}============================================================================${NC}"
echo ""

# Check if secrets directory exists
if [ -d "$SECRETS_DIR" ]; then
    echo -e "${YELLOW}Warning: Secrets directory already exists.${NC}"
    read -p "Do you want to overwrite existing secrets? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Aborted. No changes made.${NC}"
        exit 1
    fi
    echo -e "${YELLOW}Removing old secrets...${NC}"
    rm -rf "$SECRETS_DIR"
fi

# Create secrets directory with restricted permissions
echo -e "${GREEN}Creating secrets directory...${NC}"
mkdir -p "$SECRETS_DIR"
chmod 700 "$SECRETS_DIR"

# Function to generate a secure random password
generate_password() {
    local length=${1:-32}
    # Generate password using /dev/urandom and base64
    openssl rand -base64 "$length" | tr -d "=+/" | cut -c1-"$length"
}

# Function to create a secret file
create_secret() {
    local name=$1
    local value=$2
    local file="${SECRETS_DIR}/${name}.txt"

    echo -e "${GREEN}Creating secret: ${name}${NC}"
    echo -n "$value" > "$file"
    chmod 600 "$file"
    echo -e "  ${GREEN}✓${NC} Created: $file"
}

# Generate PostgreSQL password
echo ""
echo -e "${GREEN}Generating PostgreSQL password...${NC}"
POSTGRES_PASSWORD=$(generate_password 32)
create_secret "postgres_password" "$POSTGRES_PASSWORD"

# Generate pgAdmin password
echo ""
echo -e "${GREEN}Generating pgAdmin password...${NC}"
PGADMIN_PASSWORD=$(generate_password 24)
create_secret "pgadmin_password" "$PGADMIN_PASSWORD"

# Create README and gitignore in secrets directory
cat > "${SECRETS_DIR}/README.md" << 'EOF'
# Secrets Directory

This directory contains sensitive credentials for the PAW blockchain infrastructure.

## Security Notice

⚠️ **NEVER COMMIT THIS DIRECTORY TO !**

Generated secrets should be:
- Kept secure with file permissions (600 for files, 700 for directory)
- Backed up securely (encrypted backup only)
- Rotated regularly (at least every 90 days)
EOF
chmod 600 "${SECRETS_DIR}/README.md"

cat > "${SECRETS_DIR}/ignore" << 'EOF'
# Ignore all secrets
*.txt
*.key
*.pem
!README.md
EOF
chmod 644 "${SECRETS_DIR}/ignore"

echo ""
echo -e "${GREEN}============================================================================${NC}"
echo -e "${GREEN}Secrets Setup Complete!${NC}"
echo -e "${GREEN}============================================================================${NC}"
echo ""
