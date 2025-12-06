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

NODE_NAME=${1:?"node name required"}
RPC_PORT=${2:-26657}
GRPC_PORT=${3:-9090}
API_PORT=${4:-1317}

CHAIN_ID="paw-devnet"
KEYRING_BACKEND="test"
HOME_DIR="/root/.paw/${NODE_NAME}"
STATE_DIR="/paw/scripts/devnet/.state"
GENESIS_SHARE="${STATE_DIR}/genesis.json"

# Capture output so we can debug container exits from the host.
mkdir -p "$STATE_DIR"
exec > >(tee -a "${STATE_DIR}/init_${NODE_NAME}.log") 2>&1

mkdir -p "$HOME_DIR"

CONFIG_TOML="$HOME_DIR/config/config.toml"
APP_TOML="$HOME_DIR/config/app.toml"

echo "[init] starting ${NODE_NAME}"

if [ "$NODE_NAME" = "node1" ]; then
  # Only build a fresh genesis once, then reuse the shared copy for restarts.
  if [ ! -f "$GENESIS_SHARE" ]; then
    echo "[init:${NODE_NAME}] creating genesis"

    # ensure a clean home before init
    rm -rf "$HOME_DIR"
    pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
    set_genesis_denoms "$HOME_DIR/config/genesis.json"

    pawd keys add validator --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" --no-backup > "${STATE_DIR}/validator_key.yaml"
    pawd keys add smoke-trader --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" --no-backup > "${STATE_DIR}/trader_key.yaml"
    pawd keys add smoke-counterparty --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" --no-backup > "${STATE_DIR}/counterparty_key.yaml"

    pawd add-genesis-account validator 2000000000000upaw \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"
    pawd add-genesis-account smoke-trader 150000000000upaw,150000000000ufoo,150000000000ubar \
      --keyring-backend "$KEYRING_BACKEND" \
      --home "$HOME_DIR"
    pawd add-genesis-account smoke-counterparty 50000000000upaw \
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
    pawd tendermint show-node-id --home "$HOME_DIR" > "${STATE_DIR}/node1.id"
  else
    if [ ! -d "$HOME_DIR/config" ]; then
      pawd init "$NODE_NAME" --chain-id "$CHAIN_ID" --default-denom upaw --home "$HOME_DIR"
    fi
    cp "$GENESIS_SHARE" "$HOME_DIR/config/genesis.json"
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

sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:'"${RPC_PORT}"'"|' "$CONFIG_TOML"
sed -i 's/addr_book_strict = true/addr_book_strict = false/' "$CONFIG_TOML"
sed -i 's/seeds = ""/seeds = ""/' "$CONFIG_TOML"

if [ "$NODE_NAME" != "node1" ] && [ -f "${STATE_DIR}/node1.id" ]; then
  PEER_ID=$(cat "${STATE_DIR}/node1.id")
  sed -i 's/persistent_peers = ""/persistent_peers = "'"${PEER_ID}@paw-node1:26656"'"/' "$CONFIG_TOML"
fi

echo "[init:${NODE_NAME}] starting node"
exec pawd start --home "$HOME_DIR"
