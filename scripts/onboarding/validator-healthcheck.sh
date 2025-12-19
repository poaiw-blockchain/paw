#!/usr/bin/env bash
# Basic readiness probe for PAW validators (process + RPC + voting power).

set -euo pipefail

RPC="${RPC_ENDPOINT:-http://localhost:26657}"
HOME_DIR="${PAW_HOME:-$HOME/.paw}"
EXPECT_CHAIN_ID="${EXPECT_CHAIN_ID:-paw-testnet-1}"

log() { printf "[validator-health] %s\n" "$*"; }
fail() { echo "error: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rpc) RPC="$2"; shift 2;;
    --home) HOME_DIR="$2"; shift 2;;
    --expect-chain-id) EXPECT_CHAIN_ID="$2"; shift 2;;
    -h|--help)
      cat <<'EOF'
Usage: validator-healthcheck.sh [--rpc URL] [--home DIR] [--expect-chain-id CHAIN_ID]
EOF
      exit 0;;
    *) fail "unknown argument: $1";;
  esac
done

command -v curl >/dev/null || fail "curl is required"
command -v jq >/dev/null || fail "jq is required"

log "RPC endpoint: $RPC"
curl -fsSL "$RPC/health" >/dev/null || fail "/health unavailable"

STATUS=$(curl -fsSL "$RPC/status")
CHAIN_ID=$(echo "$STATUS" | jq -r '.result.node_info.network')
[[ "$CHAIN_ID" == "$EXPECT_CHAIN_ID" ]] || fail "chain-id mismatch (got $CHAIN_ID, expected $EXPECT_CHAIN_ID)"

CATCHING_UP=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up')
[[ "$CATCHING_UP" == "false" ]] || fail "node is still catching up"

LATEST_HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height')
VOTING_POWER=$(echo "$STATUS" | jq -r '.result.validator_info.voting_power')
[[ "$LATEST_HEIGHT" =~ ^[0-9]+$ ]] || fail "invalid height"
[[ "$VOTING_POWER" =~ ^[0-9]+$ ]] || fail "invalid voting power"
log "Height: $LATEST_HEIGHT | Voting power: $VOTING_POWER"
[[ "$VOTING_POWER" -gt 0 ]] || fail "validator has zero voting power"

NET_INFO=$(curl -fsSL "$RPC/net_info")
PEERS=$(echo "$NET_INFO" | jq -r '.result.n_peers')
[[ "$PEERS" =~ ^[0-9]+$ ]] || fail "invalid peer count"
log "Connected peers: $PEERS"
[[ "$PEERS" -ge 2 ]] || log "warning: low peer count ($PEERS)"

if [[ -f "$HOME_DIR/config/app.toml" ]]; then
  GAS_PRICE=$(grep -E '^minimum-gas-prices' "$HOME_DIR/config/app.toml" | awk -F'"' '{print $2}')
  log "Minimum gas prices: ${GAS_PRICE:-unset}"
fi

BLOCK_TIME=$(curl -fsSL "$RPC/block?height=$LATEST_HEIGHT" | jq -r '.result.block.header.time' || true)
log "Last block time: ${BLOCK_TIME:-unknown}"

log "✅ validator healthcheck passed"
