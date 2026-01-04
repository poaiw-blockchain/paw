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

require ARCHIVE_RPC_URL
require CHAIN_ID

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

log "Checking archive node at height 1"
RESP=$(curl -fsS "$ARCHIVE_RPC_URL/block?height=1")
CHAIN=$(echo "$RESP" | jq -r '.result.block.header.chain_id // empty')
HEIGHT=$(echo "$RESP" | jq -r '.result.block.header.height // empty')

if [ "$CHAIN" != "$CHAIN_ID" ]; then
  echo "Archive chain ID mismatch: expected $CHAIN_ID got ${CHAIN:-<empty>}" >&2
  exit 1
fi
if [ "$HEIGHT" != "1" ]; then
  echo "Archive height mismatch: expected 1 got ${HEIGHT:-<empty>}" >&2
  exit 1
fi

log "Archive check complete"
