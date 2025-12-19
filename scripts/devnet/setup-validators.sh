#!/usr/bin/env bash
set -euo pipefail

# Parameterized validator setup script
# Usage: ./setup-validators.sh [NUM_VALIDATORS]
# Example: ./setup-validators.sh 2

NUM_VALIDATORS=${1:-2}
if [ "$NUM_VALIDATORS" -lt 2 ] || [ "$NUM_VALIDATORS" -gt 4 ]; then
  echo "Error: NUM_VALIDATORS must be between 2 and 4" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"
CHAIN_ID="${CHAIN_ID:-paw-devnet}"
KEYRING_BACKEND="test"
STATE_DIR="${SCRIPT_DIR}/.state"
GENESIS_DIR="/tmp/paw-multivalidator-genesis"
PAWD_BIN="${PAWD_BIN:-${PROJECT_ROOT}/pawd}"

if [[ ! -x "${PAWD_BIN}" ]]; then
  echo "[setup-validators] pawd binary not found at ${PAWD_BIN}" >&2
  exit 1
fi

echo "=== Setting up ${NUM_VALIDATORS}-validator genesis ==="

rm -rf "$GENESIS_DIR"
mkdir -p "$GENESIS_DIR" "$STATE_DIR"

# Create base genesis
BASE_HOME="$GENESIS_DIR/base"
"${PAWD_BIN}" init genesis-builder --chain-id "$CHAIN_ID" --default-denom upaw --home "$BASE_HOME" >/dev/null 2>&1

echo "✓ Initialized base genesis"

# Ensure gentx directory exists for aggregation
mkdir -p "$BASE_HOME/config/gentx"

# Create validator keys and genesis accounts
for i in $(seq 1 $NUM_VALIDATORS); do
  echo "Setting up node${i} validator..."

  key_output=$("${PAWD_BIN}" keys add "node${i}_validator" --keyring-backend "$KEYRING_BACKEND" --home "$BASE_HOME" 2>&1)
  mnemonic=$(echo "$key_output" | grep -v "^-" | grep -v "address:" | grep -v "pubkey:" | grep -v "type:" | tail -1)
  echo "$mnemonic" > "${STATE_DIR}/node${i}_validator.mnemonic"
  chmod 600 "${STATE_DIR}/node${i}_validator.mnemonic"

  "${PAWD_BIN}" add-genesis-account "node${i}_validator" 500000000000upaw \
    --keyring-backend "$KEYRING_BACKEND" \
    --home "$BASE_HOME" >/dev/null 2>&1

  echo "  ✓ Created"
done

# Add test accounts
key_output=$("${PAWD_BIN}" keys add smoke-trader --keyring-backend "$KEYRING_BACKEND" --home "$BASE_HOME" 2>&1)
mnemonic=$(echo "$key_output" | grep -v "^-" | grep -v "address:" | grep -v "pubkey:" | grep -v "type:" | tail -1)
echo "$mnemonic" > "${STATE_DIR}/smoke-trader.mnemonic"
chmod 600 "${STATE_DIR}/smoke-trader.mnemonic"

"${PAWD_BIN}" add-genesis-account smoke-trader 150000000000upaw,150000000000ufoo,150000000000ubar \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$BASE_HOME" >/dev/null 2>&1

key_output=$("${PAWD_BIN}" keys add smoke-counterparty --keyring-backend "$KEYRING_BACKEND" --home "$BASE_HOME" 2>&1)
mnemonic=$(echo "$key_output" | grep -v "^-" | grep -v "address:" | grep -v "pubkey:" | grep -v "type:" | tail -1)
echo "$mnemonic" > "${STATE_DIR}/smoke-counterparty.mnemonic"
chmod 600 "${STATE_DIR}/smoke-counterparty.mnemonic"

"${PAWD_BIN}" add-genesis-account smoke-counterparty 50000000000upaw \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$BASE_HOME" >/dev/null 2>&1

echo "Setting up faucet account..."
key_output=$("${PAWD_BIN}" keys add faucet --keyring-backend "$KEYRING_BACKEND" --home "$BASE_HOME" 2>&1)
mnemonic=$(echo "$key_output" | grep -v "^-" | grep -v "address:" | grep -v "pubkey:" | grep -v "type:" | tail -1)
if [ -n "$mnemonic" ]; then
  echo "$mnemonic" > "${STATE_DIR}/faucet.mnemonic"
  chmod 600 "${STATE_DIR}/faucet.mnemonic"
fi
printf '%s\n' "$key_output" > "${STATE_DIR}/faucet_key.yaml"
chmod 600 "${STATE_DIR}/faucet_key.yaml"
"${PAWD_BIN}" add-genesis-account faucet 5000000000000upaw \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$BASE_HOME" >/dev/null 2>&1
F_ADDR=$(show_key_address "${PAWD_BIN}" faucet --keyring-backend "$KEYRING_BACKEND" --home "$BASE_HOME" || true)
if [[ -z "${F_ADDR}" ]]; then
  echo "[setup-validators] failed to derive faucet address" >&2
  exit 1
fi
printf '%s\n' "${F_ADDR}" > "${STATE_DIR}/faucet.address"
chmod 600 "${STATE_DIR}/faucet.address"

echo "✓ Created test accounts and faucet"

# Create gentxs - each in own home for unique priv_validator_key
for i in $(seq 1 $NUM_VALIDATORS); do
  NODE_HOME="$GENESIS_DIR/node${i}"

  mkdir -p "$NODE_HOME/config"
  cp -r "$BASE_HOME/keyring-test" "$NODE_HOME/"
  cp "$BASE_HOME/config/genesis.json" "$NODE_HOME/config/"

  # Init to get unique priv_validator_key
  "${PAWD_BIN}" init "node${i}" --chain-id "$CHAIN_ID" --home "$NODE_HOME" --overwrite >/dev/null 2>&1
  cp "$BASE_HOME/config/genesis.json" "$NODE_HOME/config/genesis.json"

  echo "Creating gentx for node${i}..."
  "${PAWD_BIN}" gentx "node${i}_validator" 250000000000upaw \
    --chain-id "$CHAIN_ID" \
    --moniker "node${i}" \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --min-self-delegation "1" \
    --keyring-backend "$KEYRING_BACKEND" \
    --home "$NODE_HOME" >/dev/null 2>&1

  # Copy gentx to base
  cp "$NODE_HOME/config/gentx/"* "$BASE_HOME/config/gentx/"

  # Save priv_validator_key
  cp "$NODE_HOME/config/priv_validator_key.json" "${STATE_DIR}/node${i}.priv_validator_key.json"

  echo "  ✓ Created"
done

# Some SDK utilities rewrite initial_height as a number; enforce string encoding for CometBFT parsing
python3 - "$BASE_HOME/config/genesis.json" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)
if not isinstance(data.get("initial_height"), str):
    data["initial_height"] = str(data.get("initial_height", "1"))
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(data, fh, indent=2)
PY

# Collect gentxs
echo "Collecting gentxs..."
"${PAWD_BIN}" collect-gentxs --home "$BASE_HOME" >/dev/null 2>&1

# Add ValidatorSigningInfo for each validator to genesis
# This fixes the "no validator signing info found" error in SDK v0.50.x
# where AfterValidatorBonded hook isn't called for genesis validators
echo "Adding validator signing info to genesis..."
python3 - "$BASE_HOME/config/genesis.json" "$NUM_VALIDATORS" "$CHAIN_ID" <<'PY'
import json, sys
from bech32 import bech32_encode, convertbits

genesis_path = sys.argv[1]
num_validators = int(sys.argv[2])
chain_id = sys.argv[3]

# Determine bech32 prefix for consensus addresses based on chain ID
if "devnet" in chain_id or "testnet" in chain_id or "mainnet" in chain_id:
    # PAW uses "pawvalcons" for validator consensus addresses
    cons_prefix = "pawvalcons"
else:
    cons_prefix = "cosmosvalcons"

def hex_to_bech32(hex_addr, prefix):
    """Convert hex address to bech32 format"""
    addr_bytes = bytes.fromhex(hex_addr)
    five_bit_data = convertbits(addr_bytes, 8, 5)
    if five_bit_data is None:
        raise ValueError(f"Invalid hex address: {hex_addr}")
    return bech32_encode(prefix, five_bit_data)

with open(genesis_path, 'r') as f:
    genesis = json.load(f)

# Get validator consensus addresses from the validators array and convert to bech32
signing_infos = []
for validator in genesis.get('validators', []):
    hex_cons_addr = validator['address']
    try:
        bech32_cons_addr = hex_to_bech32(hex_cons_addr, cons_prefix)
        signing_info = {
            "address": bech32_cons_addr,
            "validator_signing_info": {
                "address": bech32_cons_addr,
                "start_height": "0",
                "index_offset": "0",
                "jailed_until": "1970-01-01T00:00:00Z",
                "tombstoned": False,
                "missed_blocks_counter": "0"
            }
        }
        signing_infos.append(signing_info)
        print(f"Converted {hex_cons_addr} -> {bech32_cons_addr}", file=sys.stderr)
    except Exception as e:
        print(f"Error converting address {hex_cons_addr}: {e}", file=sys.stderr)
        sys.exit(1)

# Update slashing module genesis
if 'app_state' in genesis and 'slashing' in genesis['app_state']:
    genesis['app_state']['slashing']['signing_infos'] = signing_infos
    print(f"Added {len(signing_infos)} signing info entries", file=sys.stderr)

with open(genesis_path, 'w') as f:
    json.dump(genesis, f, indent=2)
PY

# Validate
"${PAWD_BIN}" validate-genesis --home "$BASE_HOME" >/dev/null 2>&1

# Sanity-check bond denom and validator count
STAKING_VAL_COUNT=$(jq '.app_state.staking.validators | length' "$BASE_HOME/config/genesis.json")
COMETBFT_VAL_COUNT=$(jq '.validators | length' "$BASE_HOME/config/genesis.json")
BOND_DENOM=$(jq -r '.app_state.staking.params.bond_denom' "$BASE_HOME/config/genesis.json")

echo "Genesis structure:"
echo "  Staking validators: ${STAKING_VAL_COUNT}"
echo "  CometBFT validators: ${COMETBFT_VAL_COUNT}"
echo "  Bond denom: ${BOND_DENOM}"

if [ "$STAKING_VAL_COUNT" -ne "$NUM_VALIDATORS" ]; then
  echo "Error: staking validator count mismatch (expected ${NUM_VALIDATORS}, found ${STAKING_VAL_COUNT})" >&2
  exit 1
fi
if [ "$BOND_DENOM" != "upaw" ]; then
  echo "Error: bond denom mismatch (expected upaw, found ${BOND_DENOM})" >&2
  exit 1
fi

# Save
cp "$BASE_HOME/config/genesis.json" "${STATE_DIR}/genesis.json"
chmod 644 "${STATE_DIR}/genesis.json"

echo ""
echo "=== ✓ Multi-validator genesis complete ==="
echo "Validators: ${NUM_VALIDATORS}"
echo "Genesis: ${STATE_DIR}/genesis.json"

rm -rf "$GENESIS_DIR"
