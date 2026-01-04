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

require RPC_URL
require REST_URL
require CHAIN_ID
require DENOM
require BIN
require HOME_DIR
require KEY_NAME
require KEY_NAME_DST

need_cmd "$BIN"
need_cmd curl
need_cmd jq

RUN_ID=${RUN_ID:-$(date -u +%Y%m%d-%H%M%S)}
OUT_DIR=${OUT_DIR:-./out}/${RUN_ID}
mkdir -p "$OUT_DIR"

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

log "Resolve keys"
ADDR=$($BIN keys show "$KEY_NAME" --home "$HOME_DIR" -a 2>/dev/null || $BIN keys add "$KEY_NAME" --home "$HOME_DIR" -a)
DST_ADDR=$($BIN keys show "$KEY_NAME_DST" --home "$HOME_DIR" -a 2>/dev/null || $BIN keys add "$KEY_NAME_DST" --home "$HOME_DIR" -a)

if [ -n "${FAUCET_URL:-}" ]; then
  log "Requesting faucet funds"
  curl -fsS -X POST "$FAUCET_URL" \
    -H 'Content-Type: application/json' \
    -d "{\"address\":\"$ADDR\"}" \
    -o "$OUT_DIR/faucet.json"
fi

log "Waiting for balance"
BAL=0
for _ in $(seq 1 20); do
  BAL=$(curl -fsS "$REST_URL/cosmos/bank/v1beta1/balances/$ADDR" \
    | jq -r --arg denom "$DENOM" '.balances[]? | select(.denom==$denom) | .amount' \
    | head -n1)
  if [[ "$BAL" =~ ^[0-9]+$ ]] && [ "$BAL" -gt 0 ]; then
    break
  fi
  sleep 3
done
if ! [[ "$BAL" =~ ^[0-9]+$ ]] || [ "$BAL" -le 0 ]; then
  echo "No balance available for $ADDR" >&2
  exit 1
fi

SEND_AMOUNT=${SEND_AMOUNT:-1}
FEE_AMOUNT=${FEE_AMOUNT:-500}
GAS_ADJUSTMENT=${GAS_ADJUSTMENT:-1.3}

log "Broadcasting tx"
$BIN tx bank send "$ADDR" "$DST_ADDR" "${SEND_AMOUNT}${DENOM}" \
  --chain-id "$CHAIN_ID" \
  --node "$RPC_URL" \
  --home "$HOME_DIR" \
  --fees "${FEE_AMOUNT}${DENOM}" \
  --gas auto \
  --gas-adjustment "$GAS_ADJUSTMENT" \
  --broadcast-mode sync \
  --yes \
  --output json \
  > "$OUT_DIR/tx.json"

TXHASH=$(jq -r '.txhash // empty' "$OUT_DIR/tx.json")
if [ -z "$TXHASH" ] || [ "$TXHASH" = "null" ]; then
  echo "Failed to get txhash" >&2
  cat "$OUT_DIR/tx.json" >&2
  exit 1
fi

log "Waiting for tx inclusion"
FOUND=0
for _ in $(seq 1 20); do
  if curl -fsS "$REST_URL/cosmos/tx/v1beta1/txs/$TXHASH" \
    | jq -e '.tx_response.txhash' >/dev/null; then
    FOUND=1
    break
  fi
  sleep 2
done

if [ "$FOUND" -ne 1 ]; then
  echo "Tx not found after waiting: $TXHASH" >&2
  exit 1
fi

log "Tx test complete"
