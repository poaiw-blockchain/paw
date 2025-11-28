#!/usr/bin/env bash
set -euo pipefail

# Wait for paw-node1 to finish building pawd and start, then run smoke.
# Usage: ./scripts/devnet/wait_and_smoke.sh

NODE_CONTAINER="paw-node1"
BIN_PATH="/usr/local/bin/pawd"
MAX_WAIT_SEC=900
SLEEP_SEC=5

start_ts=$(date +%s)
while true; do
  if docker exec "$NODE_CONTAINER" test -x "$BIN_PATH"; then
    echo "pawd ready; running smoke";
    ./scripts/devnet/smoke.sh
    exit 0
  fi
  now=$(date +%s)
  if (( now - start_ts > MAX_WAIT_SEC )); then
    echo "timeout waiting for pawd in $NODE_CONTAINER" >&2
    exit 1
  fi
  sleep "$SLEEP_SEC"
done
