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

ITERATIONS=${ITERATIONS:-6}
SLEEP_SECONDS=${SLEEP_SECONDS:-10}
MAX_LAG=${MAX_LAG:-5}

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

prev_rpc=0
prev_rest=0

log "Watching heights for $ITERATIONS iterations"
for i in $(seq 1 "$ITERATIONS"); do
  rpc_height=$(curl -fsS "$RPC_URL/status" | jq -r '.result.sync_info.latest_block_height // empty')
  rest_height=$(curl -fsS "$REST_URL/cosmos/base/tendermint/v1beta1/blocks/latest" | jq -r '.block.header.height // empty')

  if ! [[ "$rpc_height" =~ ^[0-9]+$ ]]; then
    echo "Invalid RPC height: ${rpc_height:-<empty>}" >&2
    exit 1
  fi
  if ! [[ "$rest_height" =~ ^[0-9]+$ ]]; then
    echo "Invalid REST height: ${rest_height:-<empty>}" >&2
    exit 1
  fi

  if [ "$rpc_height" -lt "$prev_rpc" ]; then
    echo "RPC height decreased: $prev_rpc -> $rpc_height" >&2
    exit 1
  fi
  if [ "$rest_height" -lt "$prev_rest" ]; then
    echo "REST height decreased: $prev_rest -> $rest_height" >&2
    exit 1
  fi

  lag=$((rpc_height - rest_height))
  if [ "$lag" -lt 0 ]; then
    lag=$((rest_height - rpc_height))
  fi
  if [ "$lag" -gt "$MAX_LAG" ]; then
    echo "RPC/REST height lag too large: $lag (max $MAX_LAG)" >&2
    exit 1
  fi

  log "Iteration $i: rpc=$rpc_height rest=$rest_height lag=$lag"
  prev_rpc=$rpc_height
  prev_rest=$rest_height
  sleep "$SLEEP_SECONDS"
done

log "Height watch complete"
