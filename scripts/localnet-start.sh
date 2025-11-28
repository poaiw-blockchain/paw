#!/usr/bin/env bash

set -e

# Configuration
CHAIN_ID="paw-localnet-1"
MONIKER="localnet-node"
KEYRING_BACKEND="test"
KEY_NAME="validator"
DENOM="upaw"

# Clean up old data
echo "Cleaning up old blockchain data..."
rm -rf ~/.paw

# Initialize node
echo "Initializing node..."
pawd init $MONIKER --chain-id $CHAIN_ID

# Create a key for the validator
echo "Creating validator key..."
pawd keys add $KEY_NAME --keyring-backend $KEYRING_BACKEND

# Add genesis account
echo "Adding genesis account..."
pawd genesis add-genesis-account $KEY_NAME 1000000000000${DENOM} --keyring-backend $KEYRING_BACKEND

# Create genesis transaction
echo "Creating genesis transaction..."
pawd genesis gentx $KEY_NAME 100000000000${DENOM} \
  --chain-id $CHAIN_ID \
  --moniker $MONIKER \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --keyring-backend $KEYRING_BACKEND

# Collect genesis transactions
echo "Collecting genesis transactions..."
pawd genesis collect-gentxs

# Validate genesis
echo "Validating genesis..."
pawd genesis validate-genesis

# Update configuration for local development
echo "Updating configuration..."
sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/g' ~/.paw/config/config.toml

# Start the node
echo "Starting PAW blockchain..."
pawd start
