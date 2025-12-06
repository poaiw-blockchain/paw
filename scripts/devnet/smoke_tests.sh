#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-compose/docker-compose.devnet.yml}"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")
CHAIN_ID="paw-devnet"
KEYRING="test"
NODE_CONTAINER="paw-node1"
PAW_HOME="/root/.paw/node1"
RPC_ENDPOINT="${RPC_ENDPOINT:-http://localhost:26657}"
API_ENDPOINT="${API_ENDPOINT:-http://localhost:1317}"
# Building pawd inside the golang container is cold-cache heavy; allow a longer
# readiness window so we don't flake while the first build completes.
READY_RETRIES=${READY_RETRIES:-300}
READY_SLEEP_SECONDS=${READY_SLEEP_SECONDS:-2}
REQUIRED_BINARIES=(docker curl jq)
# Comma separated list of phases. Default covers the whole flow but operators
# can run `PAW_SMOKE_PHASES=setup` or `PAW_SMOKE_PHASES=bank` to isolate a step.
PAW_SMOKE_PHASES=${PAW_SMOKE_PHASES:-setup,bank,dex,swap,summary}
IFS=',' read -ra REQUESTED_PHASES <<< "${PAW_SMOKE_PHASES}"

for bin in "${REQUIRED_BINARIES[@]}"; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "[smoke] missing dependency: $bin" >&2
    exit 1
  fi
done

STACK_OWNER=0
RPC_READY=0
BANK_DELTA=0
POOL_ID=""
SWAP_COUNT=""
TRADER_ADDR=""
COUNTER_ADDR=""

log() {
  local phase="$1"; shift
  echo "[smoke][${phase}] $*"
}

pawd_exec() {
  docker exec -i "$NODE_CONTAINER" pawd --home "$PAW_HOME" "$@"
}

query_balance() {
  local addr=$1
  local amount
  amount=$(curl -sf "${API_ENDPOINT}/cosmos/bank/v1beta1/balances/${addr}" | jq -r '.balances[] | select(.denom=="upaw") | .amount' 2>/dev/null || true)
  amount=${amount:-0}
  echo "${amount}"
}

stack_running() {
  docker ps --format '{{.Names}}' | grep -qx "${NODE_CONTAINER}"
}

start_stack() {
  log setup "starting devnet stack (${COMPOSE_FILE})"
  "${COMPOSE_CMD[@]}" up -d --remove-orphans
  STACK_OWNER=1
}

cleanup() {
  if (( STACK_OWNER == 1 )) && [[ "${PAW_SMOKE_KEEP_STACK:-0}" != "1" ]]; then
    log cleanup "tearing down devnet stack"
    "${COMPOSE_CMD[@]}" down -v
  fi
}
trap cleanup EXIT

ensure_stack_running() {
  if ! stack_running; then
    start_stack
  fi
}

wait_for_rpc() {
  if (( RPC_READY == 1 )); then
    return
  fi
  log setup "waiting for RPC at ${RPC_ENDPOINT}"
  for ((attempt=1; attempt<=READY_RETRIES; attempt++)); do
    if curl -sf "${RPC_ENDPOINT}/status" | jq -e '.result.sync_info.catching_up == false' >/dev/null 2>&1; then
      RPC_READY=1
      log setup "RPC ready on attempt ${attempt}"
      return
    fi
    if (( attempt % 30 == 0 )); then
      log setup "still waiting for RPC (attempt ${attempt}/${READY_RETRIES})"
    fi
    if (( attempt == READY_RETRIES )); then
      log setup "RPC never became ready"
      exit 1
    fi
    sleep "${READY_SLEEP_SECONDS}"
  done
}

fetch_addresses() {
  if [[ -z "${TRADER_ADDR}" ]]; then
    TRADER_ADDR=$(pawd_exec keys show smoke-trader --keyring-backend "$KEYRING" | awk '/address:/ {print $2; exit}')
  fi
  if [[ -z "${COUNTER_ADDR}" ]]; then
    COUNTER_ADDR=$(pawd_exec keys show smoke-counterparty --keyring-backend "$KEYRING" | awk '/address:/ {print $2; exit}')
  fi
}

bank_phase() {
  fetch_addresses
  local start_balance end_balance
  start_balance=$(query_balance "$COUNTER_ADDR")

  log bank "executing bank send smoke-trader -> smoke-counterparty"
  pawd_exec tx bank send smoke-trader "$COUNTER_ADDR" 5000000upaw \
    --chain-id "$CHAIN_ID" \
    --keyring-backend "$KEYRING" \
    --yes \
    --broadcast-mode block \
    --gas 200000 \
    --fees 5000upaw >/dev/null

  end_balance=$(query_balance "$COUNTER_ADDR")
  BANK_DELTA=$(( end_balance - start_balance ))

  if (( BANK_DELTA < 5000000 )); then
    log bank "bank transfer failed validation (delta ${BANK_DELTA})"
    exit 1
  fi

  log bank "bank transfer delta ${BANK_DELTA} upaw"
}

dex_phase() {
  fetch_addresses
  log dex "creating liquidity pool upaw/ufoo"
  pawd_exec tx dex create-pool upaw 1000000 ufoo 1000000 \
    --chain-id "$CHAIN_ID" \
    --from smoke-trader \
    --keyring-backend "$KEYRING" \
    --yes \
    --broadcast-mode block \
    --gas 400000 \
    --fees 12000upaw >/dev/null

  POOL_ID=$(pawd_exec query dex pools -o json | jq -r '.pools[0].id // empty')
  if [[ -z "$POOL_ID" ]]; then
    log dex "failed to discover liquidity pool"
    exit 1
  fi
  log dex "pool created with id ${POOL_ID}"
}

swap_phase() {
  if [[ -z "$POOL_ID" ]]; then
    POOL_ID=$(pawd_exec query dex pools -o json | jq -r '.pools[0].id // empty')
  fi
  if [[ -z "$POOL_ID" ]]; then
    log swap "no pool available for swap"
    exit 1
  fi

  log swap "swapping 100000 upaw -> ufoo on pool ${POOL_ID}"
  pawd_exec tx dex swap "$POOL_ID" upaw 100000 ufoo 1 \
    --chain-id "$CHAIN_ID" \
    --from smoke-trader \
    --keyring-backend "$KEYRING" \
    --yes \
    --broadcast-mode block \
    --gas 400000 \
    --fees 12000upaw >/dev/null

  SWAP_COUNT=$(pawd_exec query dex pools -o json | jq '.pools | length')
  log swap "swap complete, pools indexed: ${SWAP_COUNT}"
}

summary_phase() {
  fetch_addresses
  if [[ -z "$POOL_ID" ]]; then
    POOL_ID=$(pawd_exec query dex pools -o json | jq -r '.pools[0].id // empty')
  fi
  if [[ -z "$SWAP_COUNT" ]]; then
    SWAP_COUNT=$(pawd_exec query dex pools -o json | jq '.pools | length')
  fi
  log summary "trader address:      ${TRADER_ADDR}"
  log summary "counterparty address:${COUNTER_ADDR}"
  log summary "pool id:             ${POOL_ID:-N/A}"
  log summary "pools indexed:       ${SWAP_COUNT:-0}"
  log summary "counterparty delta:  ${BANK_DELTA} upaw"
  log summary "✅ smoke phase summary complete"
}

run_phase() {
  local phase="$1"
  case "$phase" in
    setup)
      ensure_stack_running
      wait_for_rpc
      ;;
    bank)
      ensure_stack_running
      wait_for_rpc
      bank_phase
      ;;
    dex)
      ensure_stack_running
      wait_for_rpc
      dex_phase
      ;;
    swap)
      ensure_stack_running
      wait_for_rpc
      swap_phase
      ;;
    summary)
      ensure_stack_running
      wait_for_rpc
      summary_phase
      ;;
    *)
      echo "[smoke] unknown phase '${phase}'" >&2
      exit 1
      ;;
  esac
}

for phase in "${REQUESTED_PHASES[@]}"; do
  run_phase "${phase}"
done
