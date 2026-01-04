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

require REST_URL

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

INVALID_ADDR="invalid1address"
URL="$REST_URL/cosmos/bank/v1beta1/balances/$INVALID_ADDR"

log "Expecting client error for invalid address"
status=$(curl -s -o /tmp/error_contract.json -w "%{http_code}" "$URL")
if [ "$status" -eq 200 ]; then
  echo "Expected error for invalid address, got 200" >&2
  exit 1
fi

log "Error contract check complete (status $status)"
