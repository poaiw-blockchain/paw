#!/bin/bash
# PAW Genesis Generation and Signing Script
# This script generates a genesis file with cryptographic verification

set -e

# Configuration
CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"
GENESIS_TIME="${GENESIS_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
WORK_DIR="${WORK_DIR:-./genesis-work}"
OUTPUT_DIR="${OUTPUT_DIR:-./genesis-output}"
GPG_KEY_EMAIL="${GPG_KEY_EMAIL:-genesis@paw-chain.org}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}PAW Genesis Generation Script${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Create working directory
mkdir -p "$WORK_DIR"
mkdir -p "$OUTPUT_DIR"

echo -e "${GREEN}Configuration:${NC}"
echo "  Chain ID: $CHAIN_ID"
echo "  Genesis Time: $GENESIS_TIME"
echo "  Work Directory: $WORK_DIR"
echo "  Output Directory: $OUTPUT_DIR"
echo "  GPG Key: $GPG_KEY_EMAIL"
echo ""

# Check if pawd binary exists
if ! command -v pawd &> /dev/null; then
    echo -e "${RED}Error: pawd binary not found${NC}"
    echo "Please install pawd or add it to PATH"
    exit 1
fi

# Check if GPG is available
if ! command -v gpg &> /dev/null; then
    echo -e "${RED}Error: gpg not found${NC}"
    echo "Please install GPG: sudo apt-get install gnupg"
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq not found${NC}"
    echo "Please install jq: sudo apt-get install jq"
    exit 1
fi

echo -e "${YELLOW}[1/10] Initializing chain...${NC}"
pawd init validator \
    --chain-id "$CHAIN_ID" \
    --home "$WORK_DIR" \
    --overwrite

echo -e "${GREEN}✓ Chain initialized${NC}"
echo ""

echo -e "${YELLOW}[2/10] Configuring genesis parameters...${NC}"

# Set genesis time
jq --arg time "$GENESIS_TIME" '.genesis_time = $time' \
    "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"

# Configure consensus parameters
jq '.consensus_params.block.max_bytes = "2097152" |
    .consensus_params.block.max_gas = "100000000" |
    .consensus_params.evidence.max_age_num_blocks = "500000" |
    .consensus_params.evidence.max_age_duration = "1814400000000000"' \
    "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"

echo -e "${GREEN}✓ Genesis parameters configured${NC}"
echo ""

echo -e "${YELLOW}[3/10] Adding genesis accounts...${NC}"

# Example accounts - REPLACE WITH ACTUAL ADDRESSES
# These should be provided via environment variables or config file

if [ -n "$GENESIS_ACCOUNTS_FILE" ] && [ -f "$GENESIS_ACCOUNTS_FILE" ]; then
    echo "Loading accounts from: $GENESIS_ACCOUNTS_FILE"
    while IFS=, read -r address amount; do
        echo "  Adding: $address ($amount upaw)"
        pawd add-genesis-account "$address" "$amount" \
            --home "$WORK_DIR" \
            --keyring-backend test
    done < "$GENESIS_ACCOUNTS_FILE"
else
    echo -e "${YELLOW}⚠ Warning: No genesis accounts file provided${NC}"
    echo "  Set GENESIS_ACCOUNTS_FILE to add accounts"
    echo "  Format: address,amount (one per line)"
fi

echo -e "${GREEN}✓ Genesis accounts added${NC}"
echo ""

echo -e "${YELLOW}[4/10] Configuring modules...${NC}"

# Configure staking module
jq '.app_state.staking.params.unbonding_time = "1814400s" |
    .app_state.staking.params.max_validators = 125 |
    .app_state.staking.params.bond_denom = "upaw"' \
    "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"

# Configure governance module
jq '.app_state.gov.voting_params.voting_period = "1209600s" |
    .app_state.gov.deposit_params.min_deposit[0].denom = "upaw" |
    .app_state.gov.deposit_params.min_deposit[0].amount = "10000000000"' \
    "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"

# Configure DEX module (custom module)
if jq -e '.app_state.dex' "$WORK_DIR/config/genesis.json" > /dev/null; then
    jq '.app_state.dex.params.swap_fee = "0.003" |
        .app_state.dex.params.max_slippage = "0.05"' \
        "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
    mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"
    echo "  ✓ DEX module configured"
fi

# Configure Oracle module (custom module)
if jq -e '.app_state.oracle' "$WORK_DIR/config/genesis.json" > /dev/null; then
    jq '.app_state.oracle.params.min_validators = 4 |
        .app_state.oracle.params.update_interval = "60s"' \
        "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
    mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"
    echo "  ✓ Oracle module configured"
fi

# Configure Compute module (custom module)
if jq -e '.app_state.compute' "$WORK_DIR/config/genesis.json" > /dev/null; then
    jq '.app_state.compute.params.min_stake = "1000000000" |
        .app_state.compute.params.max_execution_time = "300s"' \
        "$WORK_DIR/config/genesis.json" > "$WORK_DIR/config/genesis.tmp.json"
    mv "$WORK_DIR/config/genesis.tmp.json" "$WORK_DIR/config/genesis.json"
    echo "  ✓ Compute module configured"
fi

echo -e "${GREEN}✓ Modules configured${NC}"
echo ""

echo -e "${YELLOW}[5/10] Collecting genesis transactions...${NC}"

# If gentxs directory exists with transactions
if [ -d "$WORK_DIR/config/gentx" ] && [ -n "$(ls -A $WORK_DIR/config/gentx)" ]; then
    pawd collect-gentxs --home "$WORK_DIR"
    echo -e "${GREEN}✓ Genesis transactions collected${NC}"
else
    echo -e "${YELLOW}⚠ No genesis transactions found${NC}"
    echo "  Validators should submit gentx files to $WORK_DIR/config/gentx/"
fi
echo ""

echo -e "${YELLOW}[6/10] Validating genesis file...${NC}"

# Validate genesis structure
pawd validate-genesis --home "$WORK_DIR" || {
    echo -e "${RED}❌ Genesis validation failed!${NC}"
    exit 1
}

echo -e "${GREEN}✓ Genesis validation passed${NC}"
echo ""

echo -e "${YELLOW}[7/10] Copying genesis to output...${NC}"

GENESIS_FILE="$OUTPUT_DIR/genesis.json"
cp "$WORK_DIR/config/genesis.json" "$GENESIS_FILE"

echo -e "${GREEN}✓ Genesis copied to: $GENESIS_FILE${NC}"
echo ""

echo -e "${YELLOW}[8/10] Generating SHA256 checksum...${NC}"

CHECKSUM=$(sha256sum "$GENESIS_FILE" | awk '{print $1}')
echo "$CHECKSUM  genesis.json" > "$OUTPUT_DIR/genesis.json.sha256"

echo -e "${GREEN}✓ Checksum generated${NC}"
echo "  SHA256: $CHECKSUM"
echo "  File: $OUTPUT_DIR/genesis.json.sha256"
echo ""

echo -e "${YELLOW}[9/10] Signing genesis with GPG...${NC}"

# Check if GPG key exists
if ! gpg --list-secret-keys "$GPG_KEY_EMAIL" &> /dev/null; then
    echo -e "${YELLOW}⚠ Warning: GPG key not found for $GPG_KEY_EMAIL${NC}"
    echo ""
    echo "To generate a GPG key:"
    echo "  gpg --full-generate-key"
    echo ""
    echo "Or specify existing key:"
    echo "  export GPG_KEY_EMAIL=your-email@example.com"
    echo ""
    read -p "Skip GPG signing? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
    SKIP_GPG=true
fi

if [ "$SKIP_GPG" != "true" ]; then
    gpg --armor --detach-sign --local-user "$GPG_KEY_EMAIL" \
        --output "$OUTPUT_DIR/genesis.json.sig" \
        "$GENESIS_FILE"

    echo -e "${GREEN}✓ Genesis signed with GPG${NC}"
    echo "  Signature: $OUTPUT_DIR/genesis.json.sig"
    echo "  Key: $GPG_KEY_EMAIL"

    # Export public key
    gpg --armor --export "$GPG_KEY_EMAIL" > "$OUTPUT_DIR/gpg-public-key.asc"
    echo "  Public Key: $OUTPUT_DIR/gpg-public-key.asc"
else
    echo -e "${YELLOW}⚠ GPG signing skipped${NC}"
fi
echo ""

echo -e "${YELLOW}[10/10] Verifying generated files...${NC}"

# Verify checksum
echo "Verifying checksum..."
cd "$OUTPUT_DIR"
sha256sum -c genesis.json.sha256 || {
    echo -e "${RED}❌ Checksum verification failed!${NC}"
    exit 1
}
cd - > /dev/null

# Verify GPG signature
if [ "$SKIP_GPG" != "true" ]; then
    echo "Verifying GPG signature..."
    gpg --verify "$OUTPUT_DIR/genesis.json.sig" "$GENESIS_FILE" || {
        echo -e "${RED}❌ GPG signature verification failed!${NC}"
        exit 1
    }
fi

echo -e "${GREEN}✓ All verifications passed${NC}"
echo ""

# Display genesis information
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Genesis Information${NC}"
echo -e "${BLUE}=========================================${NC}"

CHAIN_ID_ACTUAL=$(jq -r '.chain_id' "$GENESIS_FILE")
GENESIS_TIME_ACTUAL=$(jq -r '.genesis_time' "$GENESIS_FILE")
VALIDATORS_COUNT=$(jq '.validators | length' "$GENESIS_FILE")
TOTAL_SUPPLY=$(jq '[.app_state.bank.balances[].coins[].amount | tonumber] | add // 0' "$GENESIS_FILE")

echo "Chain ID: $CHAIN_ID_ACTUAL"
echo "Genesis Time: $GENESIS_TIME_ACTUAL"
echo "Validators: $VALIDATORS_COUNT"
echo "Total Supply: $TOTAL_SUPPLY upaw ($((TOTAL_SUPPLY / 1000000)) PAW)"
echo "SHA256: $CHECKSUM"
echo ""

# List output files
echo -e "${GREEN}Generated Files:${NC}"
echo "  1. $GENESIS_FILE"
echo "  2. $OUTPUT_DIR/genesis.json.sha256"
if [ "$SKIP_GPG" != "true" ]; then
    echo "  3. $OUTPUT_DIR/genesis.json.sig"
    echo "  4. $OUTPUT_DIR/gpg-public-key.asc"
fi
echo ""

# Create distribution package
echo -e "${YELLOW}Creating distribution package...${NC}"
PACKAGE_NAME="paw-${CHAIN_ID}-genesis-$(date +%Y%m%d-%H%M%S).tar.gz"
tar -czf "$OUTPUT_DIR/$PACKAGE_NAME" -C "$OUTPUT_DIR" \
    genesis.json \
    genesis.json.sha256 \
    $([ "$SKIP_GPG" != "true" ] && echo "genesis.json.sig gpg-public-key.asc" || echo "")

echo -e "${GREEN}✓ Distribution package created${NC}"
echo "  Package: $OUTPUT_DIR/$PACKAGE_NAME"
echo ""

# Generate deployment instructions
cat > "$OUTPUT_DIR/DEPLOYMENT_INSTRUCTIONS.txt" << EOF
========================================
PAW Genesis Deployment Instructions
========================================

Chain ID: $CHAIN_ID_ACTUAL
Genesis Time: $GENESIS_TIME_ACTUAL
SHA256: $CHECKSUM

DISTRIBUTION CHECKLIST:

□ Upload genesis.json to  release
□ Upload genesis.json.sha256 to  release
$([ "$SKIP_GPG" != "true" ] && echo "□ Upload genesis.json.sig to  release" || echo "")
$([ "$SKIP_GPG" != "true" ] && echo "□ Upload gpg-public-key.asc to  release" || echo "")
□ Update k8s/genesis-config.yaml with:
  - GENESIS_URL
  - GENESIS_CHECKSUM_URL
  $([ "$SKIP_GPG" != "true" ] && echo "- GENESIS_SIG_URL" || echo "")
  - GENESIS_CHECKSUM
□ Update k8s/genesis-secret.yaml with GPG public key
□ Announce to validators via official channels
□ Share checksum through multiple channels
□ Coordinate genesis time with all validators
□ Verify checksum with at least 3 other validators

VALIDATOR VERIFICATION:

All validators MUST independently verify:
1. Download genesis.json
2. Verify SHA256: $CHECKSUM
$([ "$SKIP_GPG" != "true" ] && echo "3. Verify GPG signature" || echo "")
3. Verify chain_id: $CHAIN_ID_ACTUAL
4. Compare with other validators
5. Only proceed if ALL checks pass

KUBERNETES DEPLOYMENT:

1. Update genesis-config.yaml:
   GENESIS_CHECKSUM: "$CHECKSUM"

2. Deploy to cluster:
   kubectl apply -f k8s/genesis-config.yaml
   kubectl apply -f k8s/genesis-secret.yaml

3. Verify deployment:
   kubectl get pods -n paw-blockchain -w

EMERGENCY CONTACTS:
- Security: security@paw-chain.org
- Validators: validators@paw-chain.org

========================================
EOF

echo -e "${GREEN}✓ Deployment instructions created${NC}"
echo "  Instructions: $OUTPUT_DIR/DEPLOYMENT_INSTRUCTIONS.txt"
echo ""

echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}✅ Genesis Generation Complete!${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""
echo -e "${YELLOW}NEXT STEPS:${NC}"
echo "1. Review all generated files"
echo "2. Distribute to validators for verification"
echo "3. Update Kubernetes configuration"
echo "4. Follow DEPLOYMENT_INSTRUCTIONS.txt"
echo "5. Coordinate network launch"
echo ""
echo -e "${RED}CRITICAL: All validators MUST verify checksum before launch!${NC}"
echo ""

exit 0
