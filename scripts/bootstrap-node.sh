#!/bin/bash
# bootstrap-node.sh
# Enhanced bootstrap script for PAW controller + compute test node
# Calls pawd init, generates keys, and configures from node-config.yaml

set -euo pipefail

# Configuration paths
BASE_DIR="infra/node"
DATA_DIR="$BASE_DIR/data"
CONFIG_FILE="infra/node-config.yaml"
CHAIN_DIR="${HOME}/.paw"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if pawd is available
if ! command -v pawd &> /dev/null; then
    log_warn "pawd binary not found. You may need to build it first."
    log_info "Creating placeholder configuration for development..."
    SKIP_INIT=true
else
    SKIP_INIT=false
fi

# Create directories
mkdir -p "$DATA_DIR"
mkdir -p "$BASE_DIR/config"
mkdir -p "$BASE_DIR/keyring"

log_info "Bootstrapping PAW node..."

# Parse configuration from node-config.yaml
if [ -f "$CONFIG_FILE" ]; then
    log_info "Reading configuration from $CONFIG_FILE"

    CHAIN_ID=$(grep "chain_id:" "$CONFIG_FILE" | awk '{print $2}')
    TOTAL_SUPPLY=$(grep "total_supply:" "$CONFIG_FILE" | awk '{print $2}')
    RESERVE=$(grep "reserve:" "$CONFIG_FILE" | awk '{print $2}')
    YEAR1_EMISSION=$(grep "year1_per_day:" "$CONFIG_FILE" | awk '{print $2}')
    MAX_VALIDATORS=$(grep "max_validators:" "$CONFIG_FILE" | awk '{print $2}')
    GENESIS_VALIDATORS=$(grep "genesis_validators:" "$CONFIG_FILE" | awk '{print $2}')
    BLOCK_TIME=$(grep "block_time_seconds:" "$CONFIG_FILE" | awk '{print $2}')
    MIN_GAS_PRICE=$(grep "min_gas_price:" "$CONFIG_FILE" | awk '{print $2}')

    log_info "Chain ID: $CHAIN_ID"
    log_info "Total supply: ${TOTAL_SUPPLY}M PAW"
    log_info "Genesis validators: $GENESIS_VALIDATORS"
    log_info "Max validators: $MAX_VALIDATORS"
else
    log_error "Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# Initialize node if pawd is available
if [ "$SKIP_INIT" = false ]; then
    log_info "Initializing node with pawd..."

    # Initialize chain data
    pawd init "paw-controller" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"

    # Generate validator keys
    log_info "Generating validator keys..."
    for i in 1 2; do
        VALIDATOR_NAME="validator-${i}"
        log_info "Creating key for ${VALIDATOR_NAME}..."
        pawd keys add "$VALIDATOR_NAME" --keyring-backend test --home "$CHAIN_DIR" 2>&1 | tee "${BASE_DIR}/keyring/${VALIDATOR_NAME}.key"
    done

    # Copy configuration
    cp "${CHAIN_DIR}/config/genesis.json" "$BASE_DIR/genesis.json"
    cp "${CHAIN_DIR}/config/config.toml" "$BASE_DIR/config/config.toml"
    cp "${CHAIN_DIR}/config/app.toml" "$BASE_DIR/config/app.toml"

    log_info "Node initialized successfully!"
else
    # Create placeholder genesis for development
    log_info "Creating placeholder genesis file..."
    cat > "$BASE_DIR/genesis.json" <<EOF
{
  "chain_id": "${CHAIN_ID}",
  "genesis_time": "2025-12-01T00:00:00Z",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "2097152",
      "max_gas": "100000000",
      "time_iota_ms": "1000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": ["ed25519"]
    },
    "version": {}
  },
  "app_state": {
    "bank": {
      "supply": [
        {
          "denom": "upaw",
          "amount": "${TOTAL_SUPPLY}000000"
        }
      ]
    },
    "staking": {
      "params": {
        "unbonding_time": "1814400s",
        "max_validators": ${MAX_VALIDATORS},
        "max_entries": 7,
        "historical_entries": 10000,
        "bond_denom": "upaw"
      }
    },
    "gov": {
      "params": {
        "min_deposit": [
          {
            "denom": "upaw",
            "amount": "10000000000"
          }
        ],
        "max_deposit_period": "604800s",
        "voting_period": "1209600s",
        "quorum": "0.400000000000000000",
        "threshold": "0.667000000000000000",
        "veto_threshold": "0.333000000000000000"
      }
    }
  }
}
EOF
fi

# Generate node environment file
log_info "Generating node environment configuration..."

# Generate Fernet salt for wallet encryption
FERNET_SALT=$(python3 - <<'PY' 2>/dev/null || echo "dGVzdHNhbHQ=")
import os, base64
print(base64.urlsafe_b64encode(os.urandom(8)).decode())
PY

cat > "$BASE_DIR/node.env" <<EOF
# PAW Node Configuration
PAW_CHAIN_ID=${CHAIN_ID}
PAW_DATA_DIR=${DATA_DIR}
PAW_EMISSION_SCHEDULE=${YEAR1_EMISSION},1435,717
PAW_FERNET_SALT=${FERNET_SALT}
PAW_TOTAL_SUPPLY=${TOTAL_SUPPLY}000000
PAW_RESERVE=${RESERVE}000000
PAW_BLOCK_TIME=${BLOCK_TIME}
PAW_MIN_GAS_PRICE=${MIN_GAS_PRICE}
PAW_MAX_VALIDATORS=${MAX_VALIDATORS}
PAW_GENESIS_VALIDATORS=${GENESIS_VALIDATORS}
EOF

# Create validator addresses file
log_info "Creating validator configuration..."
cat > "$BASE_DIR/validators.json" <<EOF
{
  "validators": [
    {
      "name": "validator-1",
      "moniker": "paw-validator-1",
      "commission_rate": "0.10",
      "max_commission_rate": "0.15",
      "max_change_rate": "0.01"
    },
    {
      "name": "validator-2",
      "moniker": "paw-validator-2",
      "commission_rate": "0.08",
      "max_commission_rate": "0.15",
      "max_change_rate": "0.02"
    }
  ]
}
EOF

# Create README for next steps
cat > "$BASE_DIR/README.md" <<'EOF'
# PAW Node Configuration

This directory contains the bootstrapped configuration for the PAW test node.

## Files

- `genesis.json` - Genesis state configuration
- `node.env` - Environment variables for node operation
- `validators.json` - Validator configuration
- `config/` - Tendermint configuration files
- `keyring/` - Validator key backups

## Next Steps

1. Initialize genesis state:
   ```bash
   ./scripts/init-genesis.sh
   ```

2. Start the node:
   ```bash
   ./infra/start-test-node.sh
   ```

3. Or use pawd directly:
   ```bash
   pawd start --home ~/.paw
   ```

## Validator Keys

Validator keys are stored in `keyring/` directory.
**Important**: Keep these keys secure in production!

## Network Parameters

- Chain ID: paw-testnet
- Block Time: 4 seconds
- Max Validators: 125
- Genesis Validators: 25
- Total Supply: 50M PAW

For more details, see `infra/node-config.yaml`
EOF

log_info ""
log_info "Bootstrap complete!"
log_info ""
log_info "Configuration files created in: $BASE_DIR"
log_info "  - genesis.json"
log_info "  - node.env"
log_info "  - validators.json"
log_info "  - README.md"
log_info ""
log_info "Next steps:"
log_info "  1. Run './scripts/init-genesis.sh' to initialize full genesis state"
log_info "  2. Run './infra/start-test-node.sh' to launch the controller node"
log_info ""
