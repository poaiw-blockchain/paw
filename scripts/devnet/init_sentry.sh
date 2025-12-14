#!/usr/bin/env bash
set -euo pipefail

# Sentry node initialization script for PAW testnet
# Sentries are non-validator full nodes that protect validators and relay to public network

set_genesis_denoms() {
  local genesis_file=$1
  python3 - <<'PY' "$genesis_file"
import json, sys
path = sys.argv[1]
with open(path, 'r') as fh:
    data = json.load(fh)

app_state = data.get("app_state", {})

staking = app_state.get("staking", {}).get("params")
if staking:
    staking["bond_denom"] = "upaw"

mint = app_state.get("mint", {}).get("params")
if mint:
    mint["mint_denom"] = "upaw"

crisis = app_state.get("crisis", {}).get("constant_fee")
if crisis:
    crisis["denom"] = "upaw"

gov = app_state.get("gov", {})
deposit_params = gov.get("deposit_params")
if deposit_params:
    deposits = deposit_params.get("min_deposit") or []
    for coin in deposits:
        coin["denom"] = "upaw"

with open(path, 'w') as fh:
    json.dump(data, fh, indent=2, sort_keys=True)
    fh.write("\n")
PY
}

SENTRY_NAME=${1:?"sentry name required (e.g., sentry1, sentry2)"}
RPC_PORT=${2:-26657}
GRPC_PORT=${3:-9090}
API_PORT=${4:-1317}

CHAIN_ID="paw-devnet"
KEYRING_BACKEND="test"
IAVL_DISABLE_FASTNODE="${IAVL_DISABLE_FASTNODE:-false}"
HOME_DIR="/root/.paw/${SENTRY_NAME}"
STATE_DIR="/paw/scripts/devnet/.state"
STATE_OWNER_UID="${STATE_OWNER_UID:-}"
STATE_OWNER_GID="${STATE_OWNER_GID:-}"
GENESIS_SHARE="${STATE_DIR}/genesis.json"
CLIENT_TOML="${HOME_DIR}/config/client.toml"

# Capture output so we can debug container exits from the host
mkdir -p "$STATE_DIR"
if [ -z "$STATE_OWNER_UID" ] || [ -z "$STATE_OWNER_GID" ]; then
  STATE_OWNER_UID=${STATE_OWNER_UID:-$(stat -c '%u' "$STATE_DIR" 2>/dev/null || echo 0)}
  STATE_OWNER_GID=${STATE_OWNER_GID:-$(stat -c '%g' "$STATE_DIR" 2>/dev/null || echo 0)}
fi
chmod 777 "$STATE_DIR"
exec > >(tee -a "${STATE_DIR}/init_${SENTRY_NAME}.log") 2>&1

restore_state_permissions() {
  if chown -R "${STATE_OWNER_UID}:${STATE_OWNER_GID}" "$STATE_DIR" >/dev/null 2>&1; then
    echo "[init:${SENTRY_NAME}] normalized state dir ownership ${STATE_OWNER_UID}:${STATE_OWNER_GID}"
  else
    echo "[init:${SENTRY_NAME}] warning: failed to chown ${STATE_DIR}" >&2
  fi
}
restore_state_permissions
trap restore_state_permissions EXIT

mkdir -p "$HOME_DIR"

CONFIG_TOML="$HOME_DIR/config/config.toml"
APP_TOML="$HOME_DIR/config/app.toml"

echo "[init] starting sentry node ${SENTRY_NAME}"

# Wait for multi-validator genesis to be available
echo "[init:${SENTRY_NAME}] waiting for genesis from validators..."
WAIT_COUNT=0
until [ -f "$GENESIS_SHARE" ] && [ -s "$GENESIS_SHARE" ]; do
  if [ $WAIT_COUNT -ge 120 ]; then
    echo "[init:${SENTRY_NAME}] ERROR: timeout waiting for genesis after 120 seconds"
    exit 1
  fi
  sleep 1
  WAIT_COUNT=$((WAIT_COUNT + 1))
done

# Detect validator count from genesis
VALIDATOR_COUNT=$(python3 - "$GENESIS_SHARE" <<'PY' 2>/dev/null || echo "0"
import json, sys
try:
    with open(sys.argv[1], 'r') as f:
        data = json.load(f)
    count = len(data.get('app_state', {}).get('staking', {}).get('validators', []))
    print(count)
except:
    print("0")
PY
)

echo "[init:${SENTRY_NAME}] found ${VALIDATOR_COUNT}-validator genesis"

if [ "$VALIDATOR_COUNT" -lt 2 ]; then
  echo "[init:${SENTRY_NAME}] ERROR: sentry nodes require multi-validator genesis (found ${VALIDATOR_COUNT})"
  exit 1
fi

# Initialize sentry node structure
if [ ! -d "$HOME_DIR/config" ]; then
  pawd init "$SENTRY_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
fi

# Use the pre-generated genesis
cp "$GENESIS_SHARE" "$HOME_DIR/config/genesis.json"

# Restore or generate node key
NODE_KEY_FILE="${STATE_DIR}/${SENTRY_NAME}.node_key.json"
if [ -f "$NODE_KEY_FILE" ]; then
  cp "$NODE_KEY_FILE" "$HOME_DIR/config/node_key.json"
  echo "[init:${SENTRY_NAME}] restored node_key from state directory"
else
  cp "$HOME_DIR/config/node_key.json" "$NODE_KEY_FILE"
  echo "[init:${SENTRY_NAME}] saved node_key to state directory"
fi

# Export node ID for peer discovery
pawd tendermint show-node-id --home "$HOME_DIR" > "${STATE_DIR}/${SENTRY_NAME}.id"
SENTRY_ID=$(cat "${STATE_DIR}/${SENTRY_NAME}.id")
echo "[init:${SENTRY_NAME}] node ID: ${SENTRY_ID}"

# CRITICAL: Remove priv_validator_key - sentries don't sign blocks
if [ -f "$HOME_DIR/config/priv_validator_key.json" ]; then
  rm -f "$HOME_DIR/config/priv_validator_key.json"
  echo "[init:${SENTRY_NAME}] removed priv_validator_key.json (sentries don't validate)"
fi

# Configure app and RPC ports
sed -i 's/^minimum-gas-prices = ""/minimum-gas-prices = "0.025upaw"/' "$APP_TOML"
sed -i 's/^enable = false/enable = true/' "$APP_TOML"
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:'"${API_PORT}"'"|' "$APP_TOML"
sed -i 's|address = "tcp://0.0.0.0:1317"|address = "tcp://0.0.0.0:'"${API_PORT}"'"|' "$APP_TOML"
sed -i 's|address = "0.0.0.0:9090"|address = "0.0.0.0:'"${GRPC_PORT}"'"|' "$APP_TOML"
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:'"${GRPC_PORT}"'"|' "$APP_TOML"
sed -i 's/^pruning *=.*/pruning = "nothing"/' "$APP_TOML"
sed -i "s/^iavl-disable-fastnode *=.*/iavl-disable-fastnode = ${IAVL_DISABLE_FASTNODE}/" "$APP_TOML"

sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:'"${RPC_PORT}"'"|' "$CONFIG_TOML"
sed -i 's/addr_book_strict = true/addr_book_strict = false/' "$CONFIG_TOML"
sed -i 's/allow_duplicate_ip = false/allow_duplicate_ip = true/' "$CONFIG_TOML"
sed -i 's/seeds = ""/seeds = ""/' "$CONFIG_TOML"
sed -i 's/^log_level = .*/log_level = "info"/' "$CONFIG_TOML"

# Increase consensus timeouts (match validator configuration)
sed -i 's/^timeout_propose = .*/timeout_propose = "10s"/' "$CONFIG_TOML"
sed -i 's/^timeout_propose_delta = .*/timeout_propose_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_prevote = .*/timeout_prevote = "5s"/' "$CONFIG_TOML"
sed -i 's/^timeout_prevote_delta = .*/timeout_prevote_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_precommit = .*/timeout_precommit = "5s"/' "$CONFIG_TOML"
sed -i 's/^timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG_TOML"

# Sentry-specific P2P configuration
# Enable PEX for accepting public connections and discovering new peers
sed -i 's/^pex = .*/pex = true/' "$CONFIG_TOML"

# Increase inbound peer limit (sentries accept public connections)
sed -i 's/^max_num_inbound_peers = .*/max_num_inbound_peers = 100/' "$CONFIG_TOML"
sed -i 's/^max_num_outbound_peers = .*/max_num_outbound_peers = 20/' "$CONFIG_TOML"

if [ -f "$CLIENT_TOML" ]; then
  sed -i 's/^keyring-backend = .*/keyring-backend = "'"${KEYRING_BACKEND}"'"/' "$CLIENT_TOML"
  sed -i 's|^node *=.*|node = "tcp://localhost:'"${RPC_PORT}"'"|' "$CLIENT_TOML"
fi

# Configure persistent peers - connect to ALL validators and other sentries
echo "[init:${SENTRY_NAME}] configuring persistent peers for ${VALIDATOR_COUNT}-validator network + sentries"

PEER_LIST=""

# Add all validator nodes
for i in $(seq 1 $VALIDATOR_COUNT); do
  PEER_NODE="node${i}"
  # Wait for validator's node ID file
  WAIT_COUNT=0
  PEER_ID=""
  while [ -z "$PEER_ID" ] && [ $WAIT_COUNT -lt 30 ]; do
    if [ -f "${STATE_DIR}/${PEER_NODE}.id" ]; then
      PEER_ID=$(cat "${STATE_DIR}/${PEER_NODE}.id" | tr -d '[:space:]')
    fi
    if [ -z "$PEER_ID" ]; then
      sleep 1
      WAIT_COUNT=$((WAIT_COUNT + 1))
    fi
  done

  if [ -n "$PEER_ID" ]; then
    if [ -n "$PEER_LIST" ]; then
      PEER_LIST="${PEER_LIST},"
    fi
    PEER_LIST="${PEER_LIST}${PEER_ID}@paw-${PEER_NODE}:26656"
    echo "[init:${SENTRY_NAME}]   adding validator peer: ${PEER_NODE} (${PEER_ID})"
  else
    echo "[init:${SENTRY_NAME}]   warning: timeout waiting for ${PEER_NODE} ID"
  fi
done

# Add other sentry nodes (sentry1 connects to sentry2 and vice versa)
for PEER_SENTRY in sentry1 sentry2; do
  if [ "$PEER_SENTRY" != "$SENTRY_NAME" ]; then
    # Wait for other sentry's node ID (may not exist yet on first boot)
    WAIT_COUNT=0
    PEER_ID=""
    while [ -z "$PEER_ID" ] && [ $WAIT_COUNT -lt 10 ]; do
      if [ -f "${STATE_DIR}/${PEER_SENTRY}.id" ]; then
        PEER_ID=$(cat "${STATE_DIR}/${PEER_SENTRY}.id" | tr -d '[:space:]')
      fi
      if [ -z "$PEER_ID" ]; then
        sleep 1
        WAIT_COUNT=$((WAIT_COUNT + 1))
      fi
    done

    if [ -n "$PEER_ID" ]; then
      if [ -n "$PEER_LIST" ]; then
        PEER_LIST="${PEER_LIST},"
      fi
      PEER_LIST="${PEER_LIST}${PEER_ID}@paw-${PEER_SENTRY}:26656"
      echo "[init:${SENTRY_NAME}]   adding sentry peer: ${PEER_SENTRY} (${PEER_ID})"
    else
      echo "[init:${SENTRY_NAME}]   note: ${PEER_SENTRY} not available yet (will connect later via PEX)"
    fi
  fi
done

if [ -n "$PEER_LIST" ]; then
  sed -i 's/persistent_peers = ""/persistent_peers = "'"${PEER_LIST}"'"/' "$CONFIG_TOML"
  echo "[init:${SENTRY_NAME}] configured persistent_peers: ${PEER_LIST}"
else
  echo "[init:${SENTRY_NAME}] warning: no peer IDs found, starting without persistent peers"
fi

echo "[init:${SENTRY_NAME}] sentry node ready - starting sync with validators"
echo "[init:${SENTRY_NAME}] note: sentries do NOT sign blocks, they relay and protect validators"
exec pawd start --home "$HOME_DIR"
