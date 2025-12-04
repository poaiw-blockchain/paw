#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-compose/docker-compose.devnet.yml}"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")
CHAIN_ID="paw-devnet"
KEYRING="test"
NODE_CONTAINER="paw-node1"
PAW_HOME="/root/.paw/node1"
RPC_ENDPOINT="${RPC_ENDPOINT:-http://localhost:26657}"
REQUIRED_BINARIES=(docker curl jq)

for bin in "${REQUIRED_BINARIES[@]}"; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "[smoke] missing dependency: $bin" >&2
    exit 1
  fi
done

echo "[smoke] starting devnet stack (${COMPOSE_FILE})"
"${COMPOSE_CMD[@]}" up -d --remove-orphans

cleanup() {
  if [[ "${PAW_SMOKE_KEEP_STACK:-0}" != "1" ]]; then
    echo "[smoke] tearing down devnet stack"
    "${COMPOSE_CMD[@]}" down -v
  fi
}
trap cleanup EXIT

pawd_exec() {
  docker exec -i "$NODE_CONTAINER" pawd --home "$PAW_HOME" "$@"
}

echo "[smoke] waiting for RPC at ${RPC_ENDPOINT}"
for attempt in {1..60}; do
  if curl -sf "${RPC_ENDPOINT}/status" | jq -e '.result.sync_info.catching_up == false' >/dev/null 2>&1; then
    break
  fi
  if [[ $attempt -eq 60 ]]; then
    echo "[smoke] RPC never became ready" >&2
    exit 1
  fi
  sleep 2
done

TRADER_ADDR=$(pawd_exec keys show smoke-trader -a --keyring-backend "$KEYRING")
COUNTER_ADDR=$(pawd_exec keys show smoke-counterparty -a --keyring-backend "$KEYRING")

start_balance=$(pawd_exec query bank balances "$COUNTER_ADDR" -o json | jq -r '.balances[] | select(.denom=="upaw") | .amount')
start_balance=${start_balance:-0}

echo "[smoke] executing bank send from smoke-trader -> smoke-counterparty"
pawd_exec tx bank send smoke-trader "$COUNTER_ADDR" 5000000upaw \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING" \
  --yes \
  --broadcast-mode block \
  --gas 200000 \
  --fees 5000upaw >/dev/null

end_balance=$(pawd_exec query bank balances "$COUNTER_ADDR" -o json | jq -r '.balances[] | select(.denom=="upaw") | .amount')
end_balance=${end_balance:-0}

if (( end_balance - start_balance < 5000000 )); then
  echo "[smoke] bank transfer failed validation" >&2
  exit 1
fi

echo "[smoke] creating liquidity pool for upaw/ufoo"
pawd_exec tx dex create-pool upaw 1000000 ufoo 1000000 \
  --chain-id "$CHAIN_ID" \
  --from smoke-trader \
  --keyring-backend "$KEYRING" \
  --yes \
  --broadcast-mode block \
  --gas 400000 \
  --fees 12000upaw >/dev/null

pool_id=$(pawd_exec query dex pools -o json | jq -r '.pools[0].id')
if [[ -z "$pool_id" || "$pool_id" == "null" ]]; then
  echo "[smoke] failed to discover liquidity pool" >&2
  exit 1
fi

echo "[smoke] swapping 100000 upaw -> ufoo on pool ${pool_id}"
pawd_exec tx dex swap "$pool_id" upaw 100000 ufoo 1 \
  --chain-id "$CHAIN_ID" \
  --from smoke-trader \
  --keyring-backend "$KEYRING" \
  --yes \
  --broadcast-mode block \
  --gas 400000 \
  --fees 12000upaw >/dev/null

swap_count=$(pawd_exec query dex pools -o json | jq '.pools | length')

echo "[smoke] summary"
echo "  trader address:      $TRADER_ADDR"
echo "  counterparty address:$COUNTER_ADDR"
echo "  pool id:             $pool_id"
echo "  pools indexed:       $swap_count"
echo "  counterparty delta:  $(( end_balance - start_balance )) upaw"
echo
echo "[smoke] ✅ devnet smoke tests completed successfully"
