#!/usr/bin/env bash
set -euo pipefail

# ensure genesis denom uses upaw
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

extract_mnemonic_from_file() {
  local path=$1
  python3 - "$path" <<'PY'
import sys
path = sys.argv[1]
mnemonic = ""
try:
    with open(path, 'r') as fh:
        for line in fh:
            stripped = line.strip()
            if stripped:
                mnemonic = stripped
except FileNotFoundError:
    mnemonic = ""

if mnemonic and len(mnemonic.split()) >= 12:
    print(mnemonic)
PY
}

ensure_mnemonic_backup() {
  local key_name=$1; shift
  local mnemonic_file="${STATE_DIR}/${key_name}.mnemonic"
  if [ -f "$mnemonic_file" ]; then
    return 0
  fi

  for candidate in "$@"; do
    local candidate_path="${STATE_DIR}/${candidate}"
    if [ ! -f "$candidate_path" ]; then
      continue
    fi
    local mnemonic
    mnemonic=$(extract_mnemonic_from_file "$candidate_path")
    if [ -n "$mnemonic" ]; then
      printf '%s\n' "$mnemonic" > "$mnemonic_file"
      chmod 600 "$mnemonic_file"
      echo "[init:${NODE_NAME}] backfilled ${key_name} mnemonic from ${candidate}"
      return 0
    fi
  done

  return 1
}

save_key_material() {
  local key_name=$1
  local info_file=$2
  local output=$3
  printf '%s\n' "$output" > "${STATE_DIR}/${info_file}"
  local mnemonic
  mnemonic=$(printf '%s\n' "$output" | python3 - <<'PY'
import sys
mnemonic = ""
for line in sys.stdin:
    stripped = line.strip()
    if stripped:
        mnemonic = stripped
if not mnemonic:
    mnemonic = ""
print(mnemonic)
PY
)
  if [ -n "$mnemonic" ]; then
    printf '%s\n' "$mnemonic" > "${STATE_DIR}/${key_name}.mnemonic"
    chmod 600 "${STATE_DIR}/${key_name}.mnemonic"
  fi
}

restore_key_if_missing() {
  local key_name=$1
  shift
  local mnemonic_file="${STATE_DIR}/${key_name}.mnemonic"
  if pawd keys show "${key_name}" --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" >/dev/null 2>&1; then
    return
  fi
  if [ ! -f "$mnemonic_file" ] && ! ensure_mnemonic_backup "$key_name" "$@"; then
    echo "[init:${NODE_NAME}] warning: no mnemonic backup for ${key_name}; keyring entry not restored"
    return
  fi
  pawd keys recover "${key_name}" --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" < "${mnemonic_file}" >/dev/null
  echo "[init:${NODE_NAME}] restored key ${key_name} from backup"
}

NODE_NAME=${1:?"node name required"}
RPC_PORT=${2:-26657}
GRPC_PORT=${3:-9090}
API_PORT=${4:-1317}

CHAIN_ID="${CHAIN_ID:-paw-devnet}"
KEYRING_BACKEND="test"
IAVL_DISABLE_FASTNODE="${IAVL_DISABLE_FASTNODE:-false}"
HOME_DIR="/root/.paw/${NODE_NAME}"
STATE_DIR="/paw/scripts/devnet/.state"
STATE_OWNER_UID="${STATE_OWNER_UID:-}"
STATE_OWNER_GID="${STATE_OWNER_GID:-}"
GENESIS_SHARE="${STATE_DIR}/genesis.json"
CLIENT_TOML="${HOME_DIR}/config/client.toml"
PRIV_VALIDATOR_STATE="${STATE_DIR}/${NODE_NAME}.priv_validator_key.json"

# Capture output so we can debug container exits from the host.
mkdir -p "$STATE_DIR"
if [ -z "$STATE_OWNER_UID" ] || [ -z "$STATE_OWNER_GID" ]; then
  STATE_OWNER_UID=${STATE_OWNER_UID:-$(stat -c '%u' "$STATE_DIR" 2>/dev/null || echo 0)}
  STATE_OWNER_GID=${STATE_OWNER_GID:-$(stat -c '%g' "$STATE_DIR" 2>/dev/null || echo 0)}
fi
chmod 777 "$STATE_DIR"
exec > >(tee -a "${STATE_DIR}/init_${NODE_NAME}.log") 2>&1

restore_state_permissions() {
  if chown -R "${STATE_OWNER_UID}:${STATE_OWNER_GID}" "$STATE_DIR" >/dev/null 2>&1; then
    echo "[init:${NODE_NAME}] normalized state dir ownership ${STATE_OWNER_UID}:${STATE_OWNER_GID}"
  else
    echo "[init:${NODE_NAME}] warning: failed to chown ${STATE_DIR}" >&2
  fi
}
restore_state_permissions
trap restore_state_permissions EXIT

mkdir -p "$HOME_DIR"

CONFIG_TOML="$HOME_DIR/config/config.toml"
APP_TOML="$HOME_DIR/config/app.toml"

echo "[init] starting ${NODE_NAME}"

# Check if a multi-validator genesis was pre-generated
# This check is global so all nodes can detect multi-validator setup
VALIDATOR_COUNT=0
if [ -f "$GENESIS_SHARE" ]; then
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
fi

if [ "$NODE_NAME" = "node1" ]; then

  if [ "$VALIDATOR_COUNT" -ge 2 ]; then
    echo "[init:${NODE_NAME}] using pre-generated ${VALIDATOR_COUNT}-validator genesis"

    # Initialize node structure
    if [ ! -d "$HOME_DIR/config" ]; then
      pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
    fi

    # Use the pre-generated genesis
    cp "$GENESIS_SHARE" "$HOME_DIR/config/genesis.json"

    # Restore node1 keys if they exist
    if [ -f "${STATE_DIR}/node1.node_key.json" ]; then
      cp "${STATE_DIR}/node1.node_key.json" "$HOME_DIR/config/node_key.json"
    else
      cp "$HOME_DIR/config/node_key.json" "${STATE_DIR}/node1.node_key.json"
    fi
    pawd tendermint show-node-id --home "$HOME_DIR" > "${STATE_DIR}/node1.id"

  elif [ ! -f "$GENESIS_SHARE" ]; then
    echo "[init:${NODE_NAME}] creating single-validator genesis (fallback mode)"

    # ensure a clean home before init
    rm -rf "$HOME_DIR"
    pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
    set_genesis_denoms "$HOME_DIR/config/genesis.json"

    key_output=$(pawd keys add validator --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")
    save_key_material "validator" "validator_key.yaml" "$key_output"
    key_output=$(pawd keys add smoke-trader --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")
    save_key_material "smoke-trader" "trader_key.yaml" "$key_output"
    key_output=$(pawd keys add smoke-counterparty --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")
    save_key_material "smoke-counterparty" "counterparty_key.yaml" "$key_output"
    key_output=$(pawd keys add faucet --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")
    save_key_material "faucet" "faucet_key.yaml" "$key_output"

    pawd add-genesis-account validator 2000000000000upaw \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"
    pawd add-genesis-account smoke-trader 150000000000upaw,150000000000ufoo,150000000000ubar \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"
    pawd add-genesis-account smoke-counterparty 50000000000upaw \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"
    pawd add-genesis-account faucet 5000000000000upaw \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"

    pawd gentx validator 1000000000000upaw \
      --chain-id "$CHAIN_ID" \
      --moniker "$NODE_NAME" \
      --commission-rate "0.10" \
      --commission-max-rate "0.20" \
      --commission-max-change-rate "0.01" \
      --min-self-delegation "1" \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"

    pawd collect-gentxs --home "$HOME_DIR"
    pawd validate --home "$HOME_DIR"

    cp "$HOME_DIR/config/genesis.json" "$GENESIS_SHARE"
    cp "$HOME_DIR/config/node_key.json" "${STATE_DIR}/node1.node_key.json"
    pawd tendermint show-node-id --home "$HOME_DIR" > "${STATE_DIR}/node1.id"
  else
    echo "[init:${NODE_NAME}] using existing genesis"
    if [ ! -d "$HOME_DIR/config" ]; then
      pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
    fi
    cp "$GENESIS_SHARE" "$HOME_DIR/config/genesis.json"
    if [ -f "${STATE_DIR}/node1.node_key.json" ]; then
      cp "${STATE_DIR}/node1.node_key.json" "$HOME_DIR/config/node_key.json"
    else
      cp "$HOME_DIR/config/node_key.json" "${STATE_DIR}/node1.node_key.json"
    fi
  fi
else
  until [ -f "$GENESIS_SHARE" ]; do
    echo "[init:${NODE_NAME}] waiting for genesis from node1..."
    sleep 1
  done

  if [ ! -d "$HOME_DIR/config" ]; then
    pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
  fi
  cp "$GENESIS_SHARE" "$HOME_DIR/config/genesis.json"
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
echo "[init:${NODE_NAME}] iavl-disable-fastnode=${IAVL_DISABLE_FASTNODE}"

sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:'"${RPC_PORT}"'"|' "$CONFIG_TOML"
sed -i 's/addr_book_strict = true/addr_book_strict = false/' "$CONFIG_TOML"
sed -i 's/allow_duplicate_ip = false/allow_duplicate_ip = true/' "$CONFIG_TOML"
sed -i 's/seeds = ""/seeds = ""/' "$CONFIG_TOML"
sed -i 's/^log_level = .*/log_level = "debug"/' "$CONFIG_TOML"
sed -i 's/^prometheus = false/prometheus = true/' "$CONFIG_TOML" || true
if grep -q '^prometheus_listen_addr' "$CONFIG_TOML"; then
  sed -i 's|^prometheus_listen_addr = .*|prometheus_listen_addr = "0.0.0.0:26660"|' "$CONFIG_TOML"
else
  printf 'prometheus_listen_addr = "0.0.0.0:26660"\n' >> "$CONFIG_TOML"
fi

# Increase consensus timeouts to allow for block preparation (fix for "ProposalBlock is nil")
sed -i 's/^timeout_propose = .*/timeout_propose = "10s"/' "$CONFIG_TOML"
sed -i 's/^timeout_propose_delta = .*/timeout_propose_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_prevote = .*/timeout_prevote = "5s"/' "$CONFIG_TOML"
sed -i 's/^timeout_prevote_delta = .*/timeout_prevote_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_precommit = .*/timeout_precommit = "5s"/' "$CONFIG_TOML"
sed -i 's/^timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' "$CONFIG_TOML"
sed -i 's/^timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG_TOML"
if [ -f "$CLIENT_TOML" ]; then
  sed -i 's/^keyring-backend = .*/keyring-backend = "'"${KEYRING_BACKEND}"'"/' "$CLIENT_TOML"
  sed -i 's|^node *=.*|node = "tcp://localhost:'"${RPC_PORT}"'"|' "$CLIENT_TOML"
fi

# Export node ID FIRST for peer discovery (before configuring peers to avoid deadlock)
NODE_KEY_FILE="${STATE_DIR}/${NODE_NAME}.node_key.json"
if [ -f "$NODE_KEY_FILE" ]; then
  cp "$NODE_KEY_FILE" "$HOME_DIR/config/node_key.json"
  echo "[init:${NODE_NAME}] restored node_key from state directory"
else
  cp "$HOME_DIR/config/node_key.json" "$NODE_KEY_FILE"
  echo "[init:${NODE_NAME}] saved node_key to state directory"
fi

# Export node ID immediately so other nodes can discover us
pawd tendermint show-node-id --home "$HOME_DIR" > "${STATE_DIR}/${NODE_NAME}.id"
echo "[init:${NODE_NAME}] node ID: $(cat "${STATE_DIR}/${NODE_NAME}.id")"

# Configure persistent peers - connect to all other nodes in multi-validator setup
if [ "$VALIDATOR_COUNT" -ge 2 ] || [ -f "${STATE_DIR}/node1.id" ]; then
  echo "[init:${NODE_NAME}] configuring persistent peers for ${VALIDATOR_COUNT}-validator network"

  # Build peer list (all nodes except self)
  PEER_LIST=""
  for i in $(seq 1 $VALIDATOR_COUNT); do
    PEER_NODE="node${i}"
    if [ "$PEER_NODE" != "$NODE_NAME" ]; then
      # Wait for peer's node ID file with content (up to 10 seconds)
      WAIT_COUNT=0
      PEER_ID=""
      while [ -z "$PEER_ID" ] && [ $WAIT_COUNT -lt 20 ]; do
        if [ -f "${STATE_DIR}/${PEER_NODE}.id" ]; then
          PEER_ID=$(cat "${STATE_DIR}/${PEER_NODE}.id" | tr -d '[:space:]')
        fi
        if [ -z "$PEER_ID" ]; then
          sleep 0.5
          WAIT_COUNT=$((WAIT_COUNT + 1))
        fi
      done

      if [ -n "$PEER_ID" ]; then
        if [ -n "$PEER_LIST" ]; then
          PEER_LIST="${PEER_LIST},"
        fi
        PEER_LIST="${PEER_LIST}${PEER_ID}@paw-${PEER_NODE}:26656"
        echo "[init:${NODE_NAME}]   adding peer: ${PEER_NODE} (${PEER_ID})"
      else
        echo "[init:${NODE_NAME}]   warning: timeout waiting for ${PEER_NODE} ID"
      fi
    fi
  done

  if [ -n "$PEER_LIST" ]; then
    sed -i 's/persistent_peers = ""/persistent_peers = "'"${PEER_LIST}"'"/' "$CONFIG_TOML"
    echo "[init:${NODE_NAME}] configured persistent_peers: ${PEER_LIST}"
  else
    echo "[init:${NODE_NAME}] warning: no peer IDs found, starting without persistent peers"
  fi
elif [ "$NODE_NAME" != "node1" ]; then
  # Fallback: single validator mode, non-node1 nodes connect to node1
  WAIT_COUNT=0
  while [ ! -f "${STATE_DIR}/node1.id" ] && [ $WAIT_COUNT -lt 60 ]; do
    echo "[init:${NODE_NAME}] waiting for node1.id..."
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
  done

  if [ -f "${STATE_DIR}/node1.id" ]; then
    PEER_ID=$(cat "${STATE_DIR}/node1.id")
    sed -i 's/persistent_peers = ""/persistent_peers = "'"${PEER_ID}@paw-node1:26656"'"/' "$CONFIG_TOML"
    echo "[init:${NODE_NAME}] configured persistent_peers with node1 ID: ${PEER_ID}"
  else
    echo "[init:${NODE_NAME}] warning: node1.id not found after 60s, starting without persistent peers"
  fi
fi

if [ -f "${PRIV_VALIDATOR_STATE}" ]; then
  cp "${PRIV_VALIDATOR_STATE}" "$HOME_DIR/config/priv_validator_key.json"
else
  cp "$HOME_DIR/config/priv_validator_key.json" "${PRIV_VALIDATOR_STATE}"
fi
chmod 600 "${PRIV_VALIDATOR_STATE}"

if [ "$NODE_NAME" = "node1" ]; then
  restore_key_if_missing "validator" "validator_key.yaml"
  restore_key_if_missing "smoke-trader" "trader_key.yaml"
  restore_key_if_missing "smoke-counterparty" "counterparty_key.yaml"
  restore_key_if_missing "faucet" "faucet_key.yaml"
fi

echo "[init:${NODE_NAME}] starting node"
exec pawd start --home "$HOME_DIR"
