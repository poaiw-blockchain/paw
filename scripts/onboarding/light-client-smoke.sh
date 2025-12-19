#!/usr/bin/env bash
# Verify a light-profile node is serving wallet-friendly RPC.

set -euo pipefail

RPC="${RPC_ENDPOINT:-http://localhost:26657}"
HOME_DIR="${PAW_HOME:-$HOME/.paw}"
EXPECTED_PRUNING="${EXPECTED_PRUNING:-custom}"
EXPECTED_STATE_SYNC="${EXPECTED_STATE_SYNC:-true}"

log() { printf "[light-smoke] %s\n" "$*"; }
fail() { echo "error: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rpc) RPC="$2"; shift 2;;
    --home) HOME_DIR="$2"; shift 2;;
    --expect-pruning) EXPECTED_PRUNING="$2"; shift 2;;
    --expect-state-sync) EXPECTED_STATE_SYNC="$2"; shift 2;;
    -h|--help)
      cat <<'EOF'
Usage: light-client-smoke.sh [--rpc URL] [--home DIR]
                             [--expect-pruning custom|default] [--expect-state-sync true|false]
EOF
      exit 0;;
    *) fail "unknown argument: $1";;
  esac
done

command -v curl >/dev/null || fail "curl is required"
command -v jq >/dev/null || fail "jq is required"

log "RPC endpoint: $RPC"

curl -fsSL "$RPC/health" >/dev/null || fail "health endpoint unavailable"
STATUS=$(curl -fsSL "$RPC/status")
CATCHING_UP=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up')
LATEST_HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height')
[[ "$CATCHING_UP" == "false" ]] || fail "node is still catching up"
[[ "$LATEST_HEIGHT" =~ ^[0-9]+$ ]] || fail "invalid height from status"
log "Latest height: $LATEST_HEIGHT"

ABCI_INFO=$(curl -fsSL "$RPC/abci_info")
LAST_BLOCK_APP_HASH=$(echo "$ABCI_INFO" | jq -r '.result.response.last_block_app_hash')
[[ -n "$LAST_BLOCK_APP_HASH" && "$LAST_BLOCK_APP_HASH" != "null" ]] || fail "missing ABCI info"

NET_INFO=$(curl -fsSL "$RPC/net_info")
PEERS=$(echo "$NET_INFO" | jq -r '.result.n_peers')
[[ "$PEERS" =~ ^[0-9]+$ ]] || fail "invalid peer count"
log "Connected peers: $PEERS"

if [[ -f "$HOME_DIR/config/app.toml" ]]; then
  PRUNING=$(grep -E '^pruning *= *' "$HOME_DIR/config/app.toml" | head -n1 | cut -d'"' -f2)
  [[ "$PRUNING" == "$EXPECTED_PRUNING" ]] || fail "pruning=$PRUNING (expected $EXPECTED_PRUNING)"
  KEEP_RECENT=$(grep -E '^pruning-keep-recent' "$HOME_DIR/config/app.toml" | awk -F'= ' '{print $2}')
  log "Pruning keep-recent: ${KEEP_RECENT:-unknown}"
fi

if [[ -f "$HOME_DIR/config/config.toml" ]]; then
  ENABLE_STATE_SYNC=$(awk '/^\[statesync\]/{f=1} f && /^enable/{print $3; exit}' "$HOME_DIR/config/config.toml" || true)
  [[ "${ENABLE_STATE_SYNC,,}" == "${EXPECTED_STATE_SYNC,,}" ]] || fail "state sync enable=$ENABLE_STATE_SYNC (expected $EXPECTED_STATE_SYNC)"
  TRUST_HEIGHT=$(awk '/^\[statesync\]/{f=1} f && /^trust_height/{print $3; exit}' "$HOME_DIR/config/config.toml" || true)
  log "State sync trust height: ${TRUST_HEIGHT:-unset}"
fi

TARGET_HEIGHT=$((LATEST_HEIGHT>5 ? LATEST_HEIGHT-3 : LATEST_HEIGHT))
curl -fsSL "$RPC/block?height=$TARGET_HEIGHT" >/dev/null || fail "cannot fetch historical block $TARGET_HEIGHT"

log "✅ light client RPC smoke checks passed"
