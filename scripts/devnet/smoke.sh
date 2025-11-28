#!/usr/bin/env bash
set -euo pipefail

# Devnet smoke test: exercises bank send, staking, DEX pool/liquidity/swap, and gov proposal/deposit/vote.
# Expects pawd running inside the paw-node1 container (brought up via docker-compose.devnet.yml).
# Usage: ./scripts/devnet/smoke.sh

NODE_CONTAINER="paw-node1"
CHAIN_ID="paw-devnet"
HOME_DIR="/root/.paw/node1"
KEYRING_BACKEND="test"
NODE_RPC="tcp://localhost:26657"
DENOM="upaw"
TOKEN_A="upaw"
TOKEN_B="ufoo"

docker exec \
  -e CHAIN_ID="$CHAIN_ID" \
  -e HOME_DIR="$HOME_DIR" \
  -e KEYRING_BACKEND="$KEYRING_BACKEND" \
  -e NODE_RPC="$NODE_RPC" \
  -e DENOM="$DENOM" \
  -e TOKEN_A="$TOKEN_A" \
  -e TOKEN_B="$TOKEN_B" \
  "$NODE_CONTAINER" bash <<'IN_CONTAINER'
set -euo pipefail

CHAIN_ID=${CHAIN_ID:-paw-devnet}
HOME_DIR=${HOME_DIR:-/root/.paw/node1}
KEYRING_BACKEND=${KEYRING_BACKEND:-test}
NODE_RPC=${NODE_RPC:-tcp://localhost:26657}
DENOM=${DENOM:-upaw}
TOKEN_A=${TOKEN_A:-upaw}
TOKEN_B=${TOKEN_B:-ufoo}

log() { echo "[smoke] $*"; }
addr() { pawd keys show "$1" -a --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR"; }

TRADER=$(addr smoke-trader)
COUNTERPARTY=$(addr smoke-counterparty)
VALOPER=$(pawd keys show validator --bech val -a --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR")

log "status"
pawd status --node "$NODE_RPC" >/dev/null

log "balances before"
pawd query bank balances "$TRADER" --node "$NODE_RPC" --home "$HOME_DIR"
pawd query bank balances "$COUNTERPARTY" --node "$NODE_RPC" --home "$HOME_DIR"

log "bank send"
pawd tx bank send "$TRADER" "$COUNTERPARTY" 1000000${DENOM} \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "ensure validator"
if ! pawd query staking validator "$VALOPER" --node "$NODE_RPC" --home "$HOME_DIR" >/dev/null 2>&1; then
  pawd tx staking create-validator \
    --amount 50000000${DENOM} \
    --pubkey "$(pawd tendermint show-validator --home "$HOME_DIR")" \
    --moniker "devnet-validator" \
    --chain-id "$CHAIN_ID" \
    --commission-rate 0.10 \
    --commission-max-rate 0.20 \
    --commission-max-change-rate 0.01 \
    --min-self-delegation 1 \
    --from validator \
    --keyring-backend "$KEYRING_BACKEND" \
    --home "$HOME_DIR" \
    --yes \
    --broadcast-mode block
fi

log "delegate"
pawd tx staking delegate "$VALOPER" 2000000${DENOM} \
  --from smoke-trader \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "create pool"
pawd tx dex create-pool "$TOKEN_A" "$TOKEN_B" 1000000 1000000 \
  --from smoke-trader \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

POOL_ID=1

log "add liquidity"
pawd tx dex add-liquidity "$POOL_ID" 500000 500000 \
  --from smoke-trader \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "swap"
pawd tx dex swap "$POOL_ID" "$TOKEN_A" "$TOKEN_B" 10000 1 \
  --from smoke-trader \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "gov submit"
pawd tx gov submit-legacy-proposal text "Smoke Test" \
  --description "devnet smoke coverage" \
  --deposit 1000000${DENOM} \
  --from smoke-trader \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

PROPOSAL_ID=1

log "gov deposit"
pawd tx gov deposit "$PROPOSAL_ID" 500000${DENOM} \
  --from smoke-counterparty \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "gov vote"
pawd tx gov vote "$PROPOSAL_ID" yes \
  --from validator \
  --chain-id "$CHAIN_ID" \
  --keyring-backend "$KEYRING_BACKEND" \
  --home "$HOME_DIR" \
  --yes \
  --broadcast-mode block

log "balances after"
pawd query bank balances "$TRADER" --node "$NODE_RPC" --home "$HOME_DIR"
pawd query bank balances "$COUNTERPARTY" --node "$NODE_RPC" --home "$HOME_DIR"

log "done"
IN_CONTAINER
