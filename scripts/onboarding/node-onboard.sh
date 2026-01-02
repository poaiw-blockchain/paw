#!/usr/bin/env bash
# One-line friendly node bootstrapper for PAW full/light profiles.

set -euo pipefail

MODE="full"                     # full | light
CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
NETWORK_BASE="${NETWORK_BASE:-https://networks.paw-testnet.io/paw-testnet-1}"
RPC_ENDPOINT="${RPC_ENDPOINT:-https://rpc1.paw-testnet.io}"
MONIKER="paw-$(hostname -s)"
START_AFTER=0

log() { printf "[node-onboard] %s\n" "$*"; }
fail() { echo "error: $*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage: node-onboard.sh [--mode full|light] [--chain-id CHAIN_ID] [--home DIR]
                       [--network-base URL] [--rpc URL] [--moniker NAME] [--start]

Defaults:
  --mode          full
  --chain-id      paw-testnet-1
  --home          $HOME/.paw
  --network-base  https://networks.paw-testnet.io/paw-testnet-1
  --rpc           https://rpc1.paw-testnet.io
  --moniker       paw-$(hostname -s)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="$2"; shift 2;;
    --chain-id) CHAIN_ID="$2"; shift 2;;
    --home) PAW_HOME="$2"; shift 2;;
    --network-base) NETWORK_BASE="$2"; shift 2;;
    --rpc) RPC_ENDPOINT="$2"; shift 2;;
    --moniker) MONIKER="$2"; shift 2;;
    --start) START_AFTER=1; shift;;
    -h|--help) usage; exit 0;;
    *) fail "unknown argument: $1";;
  esac
done

[[ "$MODE" == "full" || "$MODE" == "light" ]] || fail "mode must be full or light"

command -v curl >/dev/null || fail "curl is required"
command -v jq >/dev/null || fail "jq is required"

if command -v pawd >/dev/null; then
  PAWD_BIN="$(command -v pawd)"
elif command -v go >/dev/null; then
  log "pawd not found, installing from source (go install ./cmd/pawd@main)"
  GO111MODULE=on go install github.com/paw-chain/paw/cmd/pawd@main
  PAWD_BIN="$(command -v pawd)"
else
  fail "pawd binary not found and Go is unavailable"
fi

CHAIN_DIR="${PAW_HOME%/}"
CONFIG_DIR="$CHAIN_DIR/config"
mkdir -p "$CONFIG_DIR"

BASE="${NETWORK_BASE%/}"
MANIFEST_URL="$BASE/${CHAIN_ID}-manifest.json"
GENESIS_URL="$BASE/genesis.json"
PEERS_URL="$BASE/peers.txt"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

log "Fetching manifest from $MANIFEST_URL"
curl -fsSL "$MANIFEST_URL" -o "$TMP_DIR/manifest.json"

EXPECTED_GENESIS_SHA=$(jq -r '.genesis_sha256' "$TMP_DIR/manifest.json")
PERSISTENT_PEERS=$(jq -r '.persistent_peers' "$TMP_DIR/manifest.json")
SEEDS=$(jq -r '.seeds' "$TMP_DIR/manifest.json")

log "Downloading genesis.json"
curl -fsSL "$GENESIS_URL" -o "$TMP_DIR/genesis.json"
ACTUAL_SHA=$(sha256sum "$TMP_DIR/genesis.json" | awk '{print $1}')
[[ "$ACTUAL_SHA" == "$EXPECTED_GENESIS_SHA" ]] || fail "genesis sha mismatch ($ACTUAL_SHA != $EXPECTED_GENESIS_SHA)"

log "Syncing genesis into $CONFIG_DIR/genesis.json"
cp "$TMP_DIR/genesis.json" "$CONFIG_DIR/genesis.json"

if [[ ! -f "$CONFIG_DIR/config.toml" ]]; then
  log "Initializing new home at $CHAIN_DIR (moniker=$MONIKER, chain-id=$CHAIN_ID)"
  "$PAWD_BIN" init "$MONIKER" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"
fi

log "Applying seeds/persistent peers"
curl -fsSL "$PEERS_URL" -o "$TMP_DIR/peers.txt"
perl -0pi -e "s/^seeds *=.*$/seeds = \"$SEEDS\"/m" "$CONFIG_DIR/config.toml"
perl -0pi -e "s/^persistent_peers *=.*$/persistent_peers = \"$PERSISTENT_PEERS\"/m" "$CONFIG_DIR/config.toml"

CONFIG_TOML="$CONFIG_DIR/config.toml"
APP_TOML="$CONFIG_DIR/app.toml"

if [[ "$MODE" == "full" ]]; then
  log "Configuring full node pruning and state"
  perl -0pi -e 's/^pruning *=.*$/pruning = "default"/m' "$APP_TOML"
  perl -0pi -e 's/^pruning-keep-recent *=.*$/pruning-keep-recent = "0"/m' "$APP_TOML"
  perl -0pi -e 's/^pruning-interval *=.*$/pruning-interval = "0"/m' "$APP_TOML"
  perl -0pi -e 's/^snapshot-interval *=.*$/snapshot-interval = 1000/m' "$APP_TOML"
else
  log "Configuring light profile (state sync + aggressive pruning)"
  perl -0pi -e 's/^pruning *=.*$/pruning = "custom"/m' "$APP_TOML"
  perl -0pi -e 's/^pruning-keep-recent *=.*$/pruning-keep-recent = "1000"/m' "$APP_TOML"
  perl -0pi -e 's/^pruning-interval *=.*$/pruning-interval = "50"/m' "$APP_TOML"
  perl -0pi -e 's/^snapshot-interval *=.*$/snapshot-interval = 0/m' "$APP_TOML"
fi

perl -0pi -e 's/^minimum-gas-prices *=.*$/minimum-gas-prices = "0.025upaw"/m' "$APP_TOML"

sed -i '/^\[rpc\]/,/^\[/{s/^cors_allowed_origins = .*/cors_allowed_origins = ["*"]/}' "$CONFIG_TOML"

if [[ "$MODE" == "light" ]]; then
  log "Deriving trust height/hash from $RPC_ENDPOINT"
  LATEST_HEIGHT=$(curl -fsSL "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height')
  [[ "$LATEST_HEIGHT" =~ ^[0-9]+$ ]] || fail "could not read latest height from $RPC_ENDPOINT"
  TRUST_HEIGHT=$((LATEST_HEIGHT-2000))
  (( TRUST_HEIGHT > 1 )) || TRUST_HEIGHT=1
  TRUST_HASH=$(curl -fsSL "$RPC_ENDPOINT/block?height=$TRUST_HEIGHT" | jq -r '.result.block_id.hash')
  [[ "$TRUST_HASH" != "null" && -n "$TRUST_HASH" ]] || fail "could not fetch trust hash"

  sed -i '/^\[statesync\]/,/^\[/{s/^enable = .*/enable = true/}' "$CONFIG_TOML"
  sed -i "/^\[statesync\]/,/^\[/{s|^rpc_servers = .*|rpc_servers = \"$RPC_ENDPOINT,$RPC_ENDPOINT\"|}" "$CONFIG_TOML"
  sed -i "/^\[statesync\]/,/^\[/{s/^trust_height = .*/trust_height = $TRUST_HEIGHT/}" "$CONFIG_TOML"
  sed -i "/^\[statesync\]/,/^\[/{s/^trust_hash = .*/trust_hash = \"$TRUST_HASH\"/}" "$CONFIG_TOML"
  sed -i "/^\[statesync\]/,/^\[/{s/^trust_period = .*/trust_period = \"168h\"/}" "$CONFIG_TOML"
  sed -i "/^\[statesync\]/,/^\[/{s/^discovery_time = .*/discovery_time = \"10s\"/}" "$CONFIG_TOML"
else
  sed -i '/^\[statesync\]/,/^\[/{s/^enable = .*/enable = false/}' "$CONFIG_TOML"
fi

log "Home prepared at $CHAIN_DIR"
log "Start command: $PAWD_BIN start --home $CHAIN_DIR"

if [[ $START_AFTER -eq 1 ]]; then
  exec "$PAWD_BIN" start --home "$CHAIN_DIR"
fi
