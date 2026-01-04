#!/usr/bin/env bash
set -euo pipefail

require() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    echo "Missing required env: $name" >&2
    exit 1
  fi
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

need_cmd curl
need_cmd jq

require RPC_URL
require REST_URL
require CHAIN_ID

RUN_ID=${RUN_ID:-$(date -u +%Y%m%d-%H%M%S)}
OUT_DIR=${OUT_DIR:-./out}/${RUN_ID}
mkdir -p "$OUT_DIR"

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

log "RPC health"
curl -fsS "$RPC_URL/health" -o "$OUT_DIR/rpc_health.json"

log "RPC status"
curl -fsS "$RPC_URL/status" -o "$OUT_DIR/rpc_status.json"
CHAIN=$(jq -r '.result.node_info.network // empty' "$OUT_DIR/rpc_status.json")
HEIGHT=$(jq -r '.result.sync_info.latest_block_height // empty' "$OUT_DIR/rpc_status.json")
if [ -z "$CHAIN" ] || [ "$CHAIN" != "$CHAIN_ID" ]; then
  echo "Chain ID mismatch: expected $CHAIN_ID got ${CHAIN:-<empty>}" >&2
  exit 1
fi
if ! [[ "$HEIGHT" =~ ^[0-9]+$ ]]; then
  echo "Invalid latest_block_height: ${HEIGHT:-<empty>}" >&2
  exit 1
fi

log "RPC net_info"
curl -fsS "$RPC_URL/net_info" -o "$OUT_DIR/rpc_net_info.json"
PEERS=$(jq -r '.result.n_peers // empty' "$OUT_DIR/rpc_net_info.json")
if ! [[ "$PEERS" =~ ^[0-9]+$ ]] || [ "$PEERS" -lt 1 ]; then
  echo "Peer count too low: ${PEERS:-<empty>}" >&2
  exit 1
fi

log "REST latest block"
curl -fsS "$REST_URL/cosmos/base/tendermint/v1beta1/blocks/latest" -o "$OUT_DIR/rest_latest_block.json"
REST_CHAIN=$(jq -r '.block.header.chain_id // empty' "$OUT_DIR/rest_latest_block.json")
REST_HEIGHT=$(jq -r '.block.header.height // empty' "$OUT_DIR/rest_latest_block.json")
if [ -z "$REST_CHAIN" ] || [ "$REST_CHAIN" != "$CHAIN_ID" ]; then
  echo "REST chain_id mismatch: expected $CHAIN_ID got ${REST_CHAIN:-<empty>}" >&2
  exit 1
fi
if ! [[ "$REST_HEIGHT" =~ ^[0-9]+$ ]]; then
  echo "Invalid REST block height: ${REST_HEIGHT:-<empty>}" >&2
  exit 1
fi

if [ -n "${GRAPHQL_URL:-}" ]; then
  log "GraphQL basic query"
  curl -fsS -X POST "$GRAPHQL_URL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"{__typename}"}' \
    -o "$OUT_DIR/graphql.json"
  jq -e '.data' "$OUT_DIR/graphql.json" >/dev/null
fi

log "Smoke checks complete"
